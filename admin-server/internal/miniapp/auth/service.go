package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
	systemsetting "mall/admin_core/internal/system/setting"
)

const (
	TokenIssuer = "mall-miniapp-member"
)

var (
	ErrUnavailable    = errors.New("服务不可用")
	ErrInvalidToken   = errors.New("登录凭证无效或已过期")
	ErrMemberDisabled = errors.New("会员账号已被禁用")
)

type tokenClaims struct {
	jwt.RegisteredClaims
}

type WeChatSessionResult struct {
	OpenID     string
	SessionKey string
	UnionID    string
}

type WeChatClient interface {
	Code2Session(ctx context.Context, appID, appSecret, code string) (*WeChatSessionResult, error)
}

type defaultWeChatClient struct {
	httpClient *http.Client
}

func NewDefaultWeChatClient() WeChatClient {
	return &defaultWeChatClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *defaultWeChatClient) Code2Session(ctx context.Context, appID, appSecret, code string) (*WeChatSessionResult, error) {
	endpoint := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		url.QueryEscape(appID),
		url.QueryEscape(appSecret),
		url.QueryEscape(code),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("创建微信认证请求失败: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("微信认证请求网络失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取微信认证响应失败: %w", err)
	}

	var wxResp struct {
		OpenID     string `json:"openid"`
		SessionKey string `json:"session_key"`
		UnionID    string `json:"unionid"`
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &wxResp); err != nil {
		return nil, fmt.Errorf("解析微信认证响应失败: %w", err)
	}

	if wxResp.ErrCode != 0 {
		return nil, fmt.Errorf("微信认证错误 (错误码: %d, 信息: %s)", wxResp.ErrCode, wxResp.ErrMsg)
	}
	if wxResp.OpenID == "" {
		return nil, errors.New("微信认证失败: openid 为空")
	}

	return &WeChatSessionResult{
		OpenID:     wxResp.OpenID,
		SessionKey: wxResp.SessionKey,
		UnionID:    wxResp.UnionID,
	}, nil
}

type Service struct {
	db        *gorm.DB
	jwtSecret []byte
	tokenTTL  time.Duration
	wechat    WeChatClient
}

func NewService(db *gorm.DB, jwtSecret string, tokenTTL time.Duration) *Service {
	return &Service{
		db:        db,
		jwtSecret: []byte(strings.TrimSpace(jwtSecret)),
		tokenTTL:  tokenTTL,
		wechat:    NewDefaultWeChatClient(),
	}
}

func (s *Service) SetWeChatClient(client WeChatClient) {
	s.wechat = client
}

func (s *Service) WechatLogin(ctx context.Context, code string) (WechatLoginResponse, error) {
	if s == nil || s.db == nil || len(s.jwtSecret) == 0 {
		return WechatLoginResponse{}, ErrUnavailable
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return WechatLoginResponse{}, errors.New("登录凭证 code 不能为空")
	}

	settingService := systemsetting.NewService(s.db)
	mpConfig, err := settingService.GetMp(ctx)
	if err != nil {
		return WechatLoginResponse{}, fmt.Errorf("获取小程序设置失败: %w", err)
	}
	if mpConfig.AppID == "" || mpConfig.AppSecret == "" {
		return WechatLoginResponse{}, errors.New("微信小程序尚未配置 app_id 或 app_secret")
	}

	wxResult, err := s.wechat.Code2Session(ctx, mpConfig.AppID, mpConfig.AppSecret, code)
	if err != nil {
		return WechatLoginResponse{}, err
	}

	now := time.Now()
	var authRow model.MemberAuth
	findErr := s.db.WithContext(ctx).
		Where("auth_type = ? AND app_id = ? AND openid = ?", "wechat_miniapp", mpConfig.AppID, wxResult.OpenID).
		First(&authRow).Error

	var member model.Member
	isNewMember := false

	if findErr == nil {
		if err := s.db.WithContext(ctx).First(&member, authRow.MemberID).Error; err != nil {
			return WechatLoginResponse{}, fmt.Errorf("查询会员失败: %w", err)
		}
		if member.Status != 1 {
			return WechatLoginResponse{}, ErrMemberDisabled
		}
		member.LastLoginAt = &now
		_ = s.db.WithContext(ctx).Model(&member).Update("last_login_at", &now).Error
	} else if errors.Is(findErr, gorm.ErrRecordNotFound) {
		isNewMember = true
		err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			member = model.Member{
				Nickname:    "微信用户",
				Status:      1,
				LastLoginAt: &now,
			}
			if err := tx.Create(&member).Error; err != nil {
				return err
			}

			var unionidPtr *string
			if strings.TrimSpace(wxResult.UnionID) != "" {
				u := strings.TrimSpace(wxResult.UnionID)
				unionidPtr = &u
			}

			var providerDataPtr *string
			snapshot := map[string]any{
				"auth_type": "wechat_miniapp",
			}
			if unionidPtr != nil {
				snapshot["unionid"] = *unionidPtr
			}
			if snapBytes, err := json.Marshal(snapshot); err == nil {
				s := string(snapBytes)
				providerDataPtr = &s
			}

			authRow = model.MemberAuth{
				MemberID:     member.ID,
				AuthType:     "wechat_miniapp",
				AppID:        mpConfig.AppID,
				OpenID:       wxResult.OpenID,
				UnionID:      unionidPtr,
				ProviderData: providerDataPtr,
			}
			return tx.Create(&authRow).Error
		})
		if err != nil {
			return WechatLoginResponse{}, fmt.Errorf("创建会员账号失败: %w", err)
		}
	} else {
		return WechatLoginResponse{}, fmt.Errorf("查询会员认证失败: %w", findErr)
	}

	expiresAt := now.Add(s.tokenTTL)
	claims := tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    TokenIssuer,
			Subject:   strconv.FormatInt(member.ID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return WechatLoginResponse{}, fmt.Errorf("生成会员凭证失败: %w", err)
	}

	avatarURL := s.resolveAvatarURL(ctx, member.AvatarID)

	return WechatLoginResponse{
		Token:       token,
		ExpiresAt:   expiresAt,
		IsNewMember: isNewMember,
		Member: MemberInfo{
			ID:          member.ID,
			Nickname:    member.Nickname,
			AvatarID:    member.AvatarID,
			AvatarURL:   avatarURL,
			Mobile:      member.Mobile,
			Status:      member.Status,
			LastLoginAt: member.LastLoginAt,
			CreatedAt:   member.CreatedAt,
		},
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (model.Member, error) {
	if s == nil || s.db == nil || len(s.jwtSecret) == 0 {
		return model.Member{}, ErrUnavailable
	}

	claims := &tokenClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(TokenIssuer))
	if err != nil || !parsed.Valid {
		return model.Member{}, ErrInvalidToken
	}

	memberID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || memberID <= 0 {
		return model.Member{}, ErrInvalidToken
	}

	var member model.Member
	if err := s.db.WithContext(ctx).First(&member, memberID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Member{}, ErrInvalidToken
		}
		return model.Member{}, fmt.Errorf("查询认证会员失败: %w", err)
	}

	if member.Status != 1 {
		return model.Member{}, ErrMemberDisabled
	}

	return member, nil
}

func (s *Service) resolveAvatarURL(ctx context.Context, avatarID *int64) string {
	if s.db == nil || avatarID == nil || *avatarID <= 0 {
		return ""
	}
	files, err := systemfile.NewService(s.db).Resolve(ctx, systemfile.IDsRequest{IDs: []int64{*avatarID}})
	if err != nil || len(files) == 0 {
		return ""
	}
	return files[0].URL
}
