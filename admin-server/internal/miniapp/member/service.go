package member

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"gorm.io/gorm"

	"mall/admin_core/internal/model"
	"mall/admin_core/internal/region"
	systemfile "mall/admin_core/internal/system/file"
)

var (
	ErrUnavailable     = errors.New("服务不可用")
	ErrMemberNotFound  = errors.New("会员不存在")
	ErrAddressNotFound = errors.New("收货地址不存在或无权操作")
)

type Service struct {
	db        *gorm.DB
	regionSvc *region.Service
}

func NewService(db *gorm.DB) *Service {
	rs, _ := region.NewService()
	return &Service{db: db, regionSvc: rs}
}

func (s *Service) SetRegionService(rs *region.Service) {
	s.regionSvc = rs
}

func (s *Service) resolveRegion(pCode, pName, cCode, cName, dCode, dName string) (*region.ResolvedRegion, error) {
	if s.regionSvc == nil {
		rs, err := region.NewService()
		if err != nil {
			return nil, err
		}
		s.regionSvc = rs
	}
	return s.regionSvc.ValidateAndResolveAddress(pCode, pName, cCode, cName, dCode, dName)
}

func (s *Service) GetProfile(ctx context.Context, memberID int64) (MemberInfo, error) {
	if s == nil || s.db == nil {
		return MemberInfo{}, ErrUnavailable
	}

	var member model.Member
	if err := s.db.WithContext(ctx).First(&member, memberID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return MemberInfo{}, ErrMemberNotFound
		}
		return MemberInfo{}, err
	}

	return MemberInfo{
		ID:          member.ID,
		Nickname:    member.Nickname,
		AvatarID:    member.AvatarID,
		AvatarURL:   s.resolveAvatarURL(ctx, member.AvatarID),
		Mobile:      member.Mobile,
		Status:      member.Status,
		LastLoginAt: member.LastLoginAt,
		CreatedAt:   member.CreatedAt,
	}, nil
}

func (s *Service) UpdateProfile(ctx context.Context, memberID int64, req UpdateProfileRequest) (MemberInfo, error) {
	if s == nil || s.db == nil {
		return MemberInfo{}, ErrUnavailable
	}

	var member model.Member
	if err := s.db.WithContext(ctx).First(&member, memberID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return MemberInfo{}, ErrMemberNotFound
		}
		return MemberInfo{}, err
	}

	updates := map[string]any{}
	if req.Nickname != nil {
		member.Nickname = strings.TrimSpace(*req.Nickname)
		updates["nickname"] = member.Nickname
	}
	if req.Mobile != nil {
		m := strings.TrimSpace(*req.Mobile)
		member.Mobile = &m
		updates["mobile"] = member.Mobile
	}

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&member).Updates(updates).Error; err != nil {
			return MemberInfo{}, fmt.Errorf("更新会员资料失败: %w", err)
		}
	}

	return MemberInfo{
		ID:          member.ID,
		Nickname:    member.Nickname,
		AvatarID:    member.AvatarID,
		AvatarURL:   s.resolveAvatarURL(ctx, member.AvatarID),
		Mobile:      member.Mobile,
		Status:      member.Status,
		LastLoginAt: member.LastLoginAt,
		CreatedAt:   member.CreatedAt,
	}, nil
}

func (s *Service) UpdateAvatar(ctx context.Context, memberID int64, name string, reader io.Reader) (AvatarResponse, error) {
	if s == nil || s.db == nil {
		return AvatarResponse{}, ErrUnavailable
	}

	fileService := systemfile.NewService(s.db)
	fileResp, err := fileService.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:   name,
		GroupID:    0,
		Scene:      model.SceneMemberAvatar,
		OwnerType:  model.OwnerTypeMember,
		OwnerID:    memberID,
		Visibility: model.VisibilityPublic,
		Reader:     reader,
	})
	if err != nil {
		return AvatarResponse{}, err
	}

	if err := s.db.WithContext(ctx).Model(&model.Member{}).Where("id = ?", memberID).Update("avatar_id", fileResp.ID).Error; err != nil {
		return AvatarResponse{}, fmt.Errorf("更新头像失败: %w", err)
	}

	return AvatarResponse{
		AvatarID:  fileResp.ID,
		AvatarURL: fileResp.URL,
	}, nil
}

func (s *Service) ListAddresses(ctx context.Context, memberID int64) ([]AddressResponse, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}

	var rows []model.MemberAddress
	err := s.db.WithContext(ctx).
		Where("member_id = ?", memberID).
		Order("is_default DESC, id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	res := make([]AddressResponse, 0, len(rows))
	for _, row := range rows {
		res = append(res, toAddressResponse(row))
	}
	return res, nil
}

func (s *Service) CreateAddress(ctx context.Context, memberID int64, req AddressCreateRequest) (AddressResponse, error) {
	if s == nil || s.db == nil {
		return AddressResponse{}, ErrUnavailable
	}

	resolved, err := s.resolveRegion(req.ProvinceCode, req.Province, req.CityCode, req.City, req.DistrictCode, req.District)
	if err != nil {
		return AddressResponse{}, err
	}

	var created model.MemberAddress
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.MemberAddress{}).Where("member_id = ?", memberID).Count(&count).Error; err != nil {
			return err
		}

		isDefault := int8(0)
		if count == 0 {
			isDefault = 1
		} else if req.IsDefault != nil && *req.IsDefault == 1 {
			if err := tx.Model(&model.MemberAddress{}).Where("member_id = ?", memberID).Update("is_default", 0).Error; err != nil {
				return err
			}
			isDefault = 1
		}

		var postalCode *string
		if req.PostalCode != nil && strings.TrimSpace(*req.PostalCode) != "" {
			p := strings.TrimSpace(*req.PostalCode)
			postalCode = &p
		}

		created = model.MemberAddress{
			MemberID:     memberID,
			Name:         strings.TrimSpace(req.Name),
			Mobile:       strings.TrimSpace(req.Mobile),
			ProvinceCode: resolved.ProvinceCode,
			Province:     resolved.Province,
			CityCode:     resolved.CityCode,
			City:         resolved.City,
			DistrictCode: resolved.DistrictCode,
			District:     resolved.District,
			Address:      strings.TrimSpace(req.Address),
			PostalCode:   postalCode,
			IsDefault:    isDefault,
		}

		return tx.Create(&created).Error
	})
	if err != nil {
		return AddressResponse{}, err
	}

	return toAddressResponse(created), nil
}

func (s *Service) UpdateAddress(ctx context.Context, memberID int64, req AddressUpdateRequest) (AddressResponse, error) {
	if s == nil || s.db == nil {
		return AddressResponse{}, ErrUnavailable
	}

	resolved, err := s.resolveRegion(req.ProvinceCode, req.Province, req.CityCode, req.City, req.DistrictCode, req.District)
	if err != nil {
		return AddressResponse{}, err
	}

	var row model.MemberAddress
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND member_id = ?", req.ID, memberID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAddressNotFound
			}
			return err
		}

		if req.IsDefault != nil && *req.IsDefault == 1 {
			if err := tx.Model(&model.MemberAddress{}).Where("member_id = ? AND id != ?", memberID, req.ID).Update("is_default", 0).Error; err != nil {
				return err
			}
			row.IsDefault = 1
		} else if req.IsDefault != nil && *req.IsDefault == 0 {
			row.IsDefault = 0
		}

		var postalCode *string
		if req.PostalCode != nil && strings.TrimSpace(*req.PostalCode) != "" {
			p := strings.TrimSpace(*req.PostalCode)
			postalCode = &p
		}

		row.Name = strings.TrimSpace(req.Name)
		row.Mobile = strings.TrimSpace(req.Mobile)
		row.ProvinceCode = resolved.ProvinceCode
		row.Province = resolved.Province
		row.CityCode = resolved.CityCode
		row.City = resolved.City
		row.DistrictCode = resolved.DistrictCode
		row.District = resolved.District
		row.Address = strings.TrimSpace(req.Address)
		row.PostalCode = postalCode

		return tx.Save(&row).Error
	})
	if err != nil {
		return AddressResponse{}, err
	}

	return toAddressResponse(row), nil
}

func (s *Service) DeleteAddress(ctx context.Context, memberID int64, addressID int64) error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.MemberAddress
		if err := tx.Where("id = ? AND member_id = ?", addressID, memberID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAddressNotFound
			}
			return err
		}

		if err := tx.Delete(&row).Error; err != nil {
			return err
		}

		if row.IsDefault == 1 {
			var first model.MemberAddress
			err := tx.Where("member_id = ?", memberID).Order("id ASC").First(&first).Error
			if err == nil {
				if err := tx.Model(&first).Update("is_default", 1).Error; err != nil {
					return err
				}
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		return nil
	})
}

func (s *Service) SetDefaultAddress(ctx context.Context, memberID int64, addressID int64) error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.MemberAddress
		if err := tx.Where("id = ? AND member_id = ?", addressID, memberID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAddressNotFound
			}
			return err
		}

		if err := tx.Model(&model.MemberAddress{}).Where("member_id = ?", memberID).Update("is_default", 0).Error; err != nil {
			return err
		}

		return tx.Model(&row).Update("is_default", 1).Error
	})
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

func toAddressResponse(row model.MemberAddress) AddressResponse {
	return AddressResponse{
		ID:           row.ID,
		MemberID:     row.MemberID,
		Name:         row.Name,
		Mobile:       row.Mobile,
		ProvinceCode: row.ProvinceCode,
		Province:     row.Province,
		CityCode:     row.CityCode,
		City:         row.City,
		DistrictCode: row.DistrictCode,
		District:     row.District,
		Address:      row.Address,
		PostalCode:   row.PostalCode,
		IsDefault:    row.IsDefault,
		CreatedAt:    row.CreatedAt,
	}
}
