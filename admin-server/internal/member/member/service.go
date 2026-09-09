package member

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
)

var (
	ErrUnavailable    = errors.New("服务不可用")
	ErrMemberNotFound = errors.New("会员不存在")
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context, req MemberListRequest) ([]MemberRow, int64, error) {
	if s == nil || s.db == nil {
		return nil, 0, ErrUnavailable
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	query := s.db.WithContext(ctx).Model(&model.Member{})

	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	keyword := strings.TrimSpace(req.Keyword)
	if keyword != "" {
		kw := "%" + keyword + "%"
		subQuery := s.db.WithContext(ctx).Model(&model.MemberAuth{}).Select("member_id").Where("openid LIKE ?", kw)
		idVal, err := strconv.ParseInt(keyword, 10, 64)
		if err == nil {
			query = query.Where("(id = ? OR nickname LIKE ? OR mobile LIKE ? OR id IN (?))", idVal, kw, kw, subQuery)
		} else {
			query = query.Where("(nickname LIKE ? OR mobile LIKE ? OR id IN (?))", kw, kw, subQuery)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.Member
	if err := query.Order("id DESC").Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	// 批量解析头像，避免 N+1
	avatarIDs := make([]int64, 0, len(rows))
	seenIDs := make(map[int64]bool)
	for _, m := range rows {
		if m.AvatarID != nil && *m.AvatarID > 0 && !seenIDs[*m.AvatarID] {
			avatarIDs = append(avatarIDs, *m.AvatarID)
			seenIDs[*m.AvatarID] = true
		}
	}

	avatarURLMap := make(map[int64]string)
	if len(avatarIDs) > 0 {
		files, err := systemfile.NewService(s.db).Resolve(ctx, systemfile.IDsRequest{IDs: avatarIDs})
		if err == nil {
			for _, f := range files {
				avatarURLMap[f.ID] = f.URL
			}
		}
	}

	result := make([]MemberRow, 0, len(rows))
	for _, m := range rows {
		var avatarURL string
		if m.AvatarID != nil {
			avatarURL = avatarURLMap[*m.AvatarID]
		}
		result = append(result, MemberRow{
			ID:          m.ID,
			Nickname:    m.Nickname,
			AvatarID:    m.AvatarID,
			AvatarURL:   avatarURL,
			Mobile:      m.Mobile,
			Status:      m.Status,
			LastLoginAt: m.LastLoginAt,
			CreatedAt:   m.CreatedAt,
		})
	}

	return result, total, nil
}

func (s *Service) Detail(ctx context.Context, id int64) (MemberDetailResponse, error) {
	if s == nil || s.db == nil {
		return MemberDetailResponse{}, ErrUnavailable
	}

	var member model.Member
	if err := s.db.WithContext(ctx).First(&member, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return MemberDetailResponse{}, ErrMemberNotFound
		}
		return MemberDetailResponse{}, err
	}

	var avatarURL string
	if member.AvatarID != nil && *member.AvatarID > 0 {
		files, err := systemfile.NewService(s.db).Resolve(ctx, systemfile.IDsRequest{IDs: []int64{*member.AvatarID}})
		if err == nil && len(files) > 0 {
			avatarURL = files[0].URL
		}
	}

	var authRows []model.MemberAuth
	if err := s.db.WithContext(ctx).Where("member_id = ?", member.ID).Order("id ASC").Find(&authRows).Error; err != nil {
		return MemberDetailResponse{}, err
	}

	var addressRows []model.MemberAddress
	if err := s.db.WithContext(ctx).Where("member_id = ?", member.ID).Order("is_default DESC, id DESC").Find(&addressRows).Error; err != nil {
		return MemberDetailResponse{}, err
	}

	auths := make([]MemberAuthRow, 0, len(authRows))
	for _, a := range authRows {
		var providerDataFormatted *string
		if a.ProviderData != nil && strings.TrimSpace(*a.ProviderData) != "" {
			var buf bytes.Buffer
			if err := json.Indent(&buf, []byte(*a.ProviderData), "", "  "); err == nil {
				s := buf.String()
				providerDataFormatted = &s
			} else {
				providerDataFormatted = a.ProviderData
			}
		}
		auths = append(auths, MemberAuthRow{
			AuthType:     a.AuthType,
			AppID:        a.AppID,
			OpenID:       a.OpenID,
			UnionID:      a.UnionID,
			ProviderData: providerDataFormatted,
			CreatedAt:    a.CreatedAt,
		})
	}

	addresses := make([]MemberAddressRow, 0, len(addressRows))
	for _, addr := range addressRows {
		addresses = append(addresses, MemberAddressRow{
			ID:           addr.ID,
			MemberID:     addr.MemberID,
			Name:         addr.Name,
			Mobile:       addr.Mobile,
			ProvinceCode: addr.ProvinceCode,
			Province:     addr.Province,
			CityCode:     addr.CityCode,
			City:         addr.City,
			DistrictCode: addr.DistrictCode,
			District:     addr.District,
			Address:      addr.Address,
			PostalCode:   addr.PostalCode,
			IsDefault:    addr.IsDefault,
			CreatedAt:    addr.CreatedAt,
		})
	}

	return MemberDetailResponse{
		Member: MemberRow{
			ID:          member.ID,
			Nickname:    member.Nickname,
			AvatarID:    member.AvatarID,
			AvatarURL:   avatarURL,
			Mobile:      member.Mobile,
			Status:      member.Status,
			LastLoginAt: member.LastLoginAt,
			CreatedAt:   member.CreatedAt,
		},
		Auth:      auths,
		Addresses: addresses,
	}, nil
}
