package share

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WeChatAPIError 微信开放接口返回的业务错误
type WeChatAPIError struct {
	ErrCode int
	ErrMsg  string
}

func (e *WeChatAPIError) Error() string {
	return fmt.Sprintf("微信接口返回错误 (code: %d): %s", e.ErrCode, e.ErrMsg)
}

// WeChatShareClient 微信分享码客户端接口
type WeChatShareClient interface {
	GetAccessToken(ctx context.Context, appID, appSecret string) (string, int, error)
	GetUnlimitedQRCode(ctx context.Context, accessToken string, params WeChatQRCodeParams) ([]byte, string, error)
}

type defaultWeChatShareClient struct {
	httpClient *http.Client
}

// NewDefaultWeChatShareClient 创建默认微信接口客户端
func NewDefaultWeChatShareClient() WeChatShareClient {
	return &defaultWeChatShareClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetAccessToken 获取微信接口调用凭证 access_token
func (c *defaultWeChatShareClient) GetAccessToken(ctx context.Context, appID, appSecret string) (string, int, error) {
	if strings.TrimSpace(appID) == "" || strings.TrimSpace(appSecret) == "" {
		return "", 0, errors.New("小程序 app_id 或 app_secret 未配置")
	}

	endpoint := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		url.QueryEscape(appID),
		url.QueryEscape(appSecret),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", 0, fmt.Errorf("创建微信 token 请求失败: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("调用微信 token 接口失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10)) // 64 KB limit
	if err != nil {
		return "", 0, fmt.Errorf("读取微信 token 响应失败: %w", err)
	}

	var tokenResp WeChatTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", 0, fmt.Errorf("解析微信 token 响应失败: %w", err)
	}

	if tokenResp.ErrCode != 0 {
		return "", 0, &WeChatAPIError{
			ErrCode: tokenResp.ErrCode,
			ErrMsg:  tokenResp.ErrMsg,
		}
	}

	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", 0, errors.New("微信接口未返回有效的 access_token")
	}

	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 7200
	}

	return tokenResp.AccessToken, expiresIn, nil
}

// GetUnlimitedQRCode 调用获取小程序码接口 wxacode.getUnlimitedQRCode
func (c *defaultWeChatShareClient) GetUnlimitedQRCode(ctx context.Context, accessToken string, params WeChatQRCodeParams) ([]byte, string, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, "", errors.New("缺少 access_token")
	}

	endpoint := fmt.Sprintf(
		"https://api.weixin.qq.com/wxa/getwxacodeunlimit?access_token=%s",
		url.QueryEscape(accessToken),
	)

	reqBody, err := json.Marshal(params)
	if err != nil {
		return nil, "", fmt.Errorf("序列化小程序码参数失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, "", fmt.Errorf("创建生成小程序码请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("调用生成小程序码接口失败: %w", err)
	}
	defer resp.Body.Close()

	// 最多读取 2MB 数据（小程序码二进制通常在 20-100KB）
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, "", fmt.Errorf("读取小程序码响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("微信接口返回异常 HTTP 状态码: %d", resp.StatusCode)
	}

	// 检查是否为 JSON 错误返回
	contentType := resp.Header.Get("Content-Type")
	isJSON := strings.Contains(strings.ToLower(contentType), "application/json") ||
		(len(data) > 0 && data[0] == '{')

	if isJSON {
		var errResp WeChatErrorResponse
		if err := json.Unmarshal(data, &errResp); err == nil && errResp.ErrCode != 0 {
			return nil, "", &WeChatAPIError{
				ErrCode: errResp.ErrCode,
				ErrMsg:  errResp.ErrMsg,
			}
		}
	}

	// 嗅探图片类型
	mimeType := http.DetectContentType(data)
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, "", fmt.Errorf("微信返回内容非合法图片 (mime: %s)", mimeType)
	}

	return data, mimeType, nil
}
