package returnaddress

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"mall/admin_core/internal/model"
)

var (
	ErrNotFound       = errors.New("退货地址不存在")
	ErrUnavailable    = errors.New("数据库服务不可用")
	ErrInvalidAddress = errors.New("退货地址信息不完整")
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func toItem(r model.OrderReturnAddress) ReturnAddressItem {
	return ReturnAddressItem{
		ID:           r.ID,
		Name:         r.Name,
		Receiver:     r.Receiver,
		Phone:        r.Phone,
		ProvinceCode: r.ProvinceCode,
		CityCode:     r.CityCode,
		DistrictCode: r.DistrictCode,
		Address:      r.Address,
		IsDefault:    r.IsDefault,
		Status:       r.Status,
		Sort:         r.Sort,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

func (s *Service) List(ctx context.Context, req ListReturnAddressesRequest) ([]ReturnAddressItem, error) {
	if s.db == nil {
		return nil, ErrUnavailable
	}
	db := s.db.WithContext(ctx).Model(&model.OrderReturnAddress{})
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}
	search := strings.TrimSpace(req.Search)
	if search != "" {
		pat := "%" + search + "%"
		db = db.Where("name LIKE ? OR receiver LIKE ? OR phone LIKE ?", pat, pat, pat)
	}

	var rows []model.OrderReturnAddress
	if err := db.Order("is_default DESC, sort ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]ReturnAddressItem, len(rows))
	for i, r := range rows {
		items[i] = toItem(r)
	}
	return items, nil
}

func (s *Service) Get(ctx context.Context, id int64) (ReturnAddressItem, error) {
	if s.db == nil {
		return ReturnAddressItem{}, ErrUnavailable
	}
	var r model.OrderReturnAddress
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ReturnAddressItem{}, ErrNotFound
		}
		return ReturnAddressItem{}, err
	}
	return toItem(r), nil
}

func (s *Service) GetDefault(ctx context.Context) (ReturnAddressItem, error) {
	if s.db == nil {
		return ReturnAddressItem{}, ErrUnavailable
	}
	var r model.OrderReturnAddress
	// 优先查找默认且启用的地址
	err := s.db.WithContext(ctx).Where("is_default = 1 AND status = 1").First(&r).Error
	if err == nil {
		return toItem(r), nil
	}
	// 其次查找启用的任意第一个地址
	err = s.db.WithContext(ctx).Where("status = 1").Order("sort ASC, id ASC").First(&r).Error
	if err == nil {
		return toItem(r), nil
	}
	// 没有任何地址
	return ReturnAddressItem{}, ErrNotFound
}

func (s *Service) Create(ctx context.Context, req CreateReturnAddressRequest) (int64, error) {
	if s.db == nil {
		return 0, ErrUnavailable
	}
	name := strings.TrimSpace(req.Name)
	receiver := strings.TrimSpace(req.Receiver)
	phone := strings.TrimSpace(req.Phone)
	address := strings.TrimSpace(req.Address)
	if name == "" || receiver == "" || phone == "" || address == "" {
		return 0, ErrInvalidAddress
	}

	var createdID int64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var totalCount int64
		if err := tx.Model(&model.OrderReturnAddress{}).Count(&totalCount).Error; err != nil {
			return err
		}

		isDefault := req.IsDefault
		if totalCount == 0 {
			isDefault = 1
		} else if isDefault == 1 {
			if err := tx.Model(&model.OrderReturnAddress{}).Where("is_default = 1").Update("is_default", 0).Error; err != nil {
				return err
			}
		}

		now := time.Now()
		status := req.Status
		if status == 0 && req.Status != 0 {
			status = 1
		}
		record := model.OrderReturnAddress{
			Name:         name,
			Receiver:     receiver,
			Phone:        phone,
			ProvinceCode: strings.TrimSpace(req.ProvinceCode),
			CityCode:     strings.TrimSpace(req.CityCode),
			DistrictCode: strings.TrimSpace(req.DistrictCode),
			Address:      address,
			IsDefault:    isDefault,
			Status:       status,
			Sort:         req.Sort,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		createdID = record.ID
		return nil
	})
	return createdID, err
}

func (s *Service) Update(ctx context.Context, req UpdateReturnAddressRequest) error {
	if s.db == nil {
		return ErrUnavailable
	}
	name := strings.TrimSpace(req.Name)
	receiver := strings.TrimSpace(req.Receiver)
	phone := strings.TrimSpace(req.Phone)
	address := strings.TrimSpace(req.Address)
	if name == "" || receiver == "" || phone == "" || address == "" {
		return ErrInvalidAddress
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.OrderReturnAddress
		if err := tx.Where("id = ?", req.ID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if req.IsDefault == 1 && existing.IsDefault != 1 {
			if err := tx.Model(&model.OrderReturnAddress{}).Where("is_default = 1").Update("is_default", 0).Error; err != nil {
				return err
			}
		}

		now := time.Now()
		updates := map[string]any{
			"name":          name,
			"receiver":      receiver,
			"phone":         phone,
			"province_code": strings.TrimSpace(req.ProvinceCode),
			"city_code":     strings.TrimSpace(req.CityCode),
			"district_code": strings.TrimSpace(req.DistrictCode),
			"address":       address,
			"is_default":    req.IsDefault,
			"status":        req.Status,
			"sort":          req.Sort,
			"updated_at":    now,
		}
		return tx.Model(&model.OrderReturnAddress{}).Where("id = ?", req.ID).Updates(updates).Error
	})
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if s.db == nil {
		return ErrUnavailable
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.OrderReturnAddress
		if err := tx.Where("id = ?", id).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if err := tx.Delete(&model.OrderReturnAddress{}, id).Error; err != nil {
			return err
		}

		// 若删除的是默认地址，尝试将剩下的第一个地址设置为默认
		if existing.IsDefault == 1 {
			var next model.OrderReturnAddress
			if err := tx.Order("sort ASC, id ASC").First(&next).Error; err == nil {
				_ = tx.Model(&model.OrderReturnAddress{}).Where("id = ?", next.ID).Update("is_default", 1).Error
			}
		}
		return nil
	})
}

func (s *Service) SetDefault(ctx context.Context, id int64) error {
	if s.db == nil {
		return ErrUnavailable
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.OrderReturnAddress
		if err := tx.Where("id = ?", id).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if err := tx.Model(&model.OrderReturnAddress{}).Where("is_default = 1").Update("is_default", 0).Error; err != nil {
			return err
		}
		return tx.Model(&model.OrderReturnAddress{}).Where("id = ?", id).Update("is_default", 1).Error
	})
}
