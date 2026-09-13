package common

import (
	"context"

	"gorm.io/gorm"
	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
	systemsetting "mall/admin_core/internal/system/setting"
	"mall/admin_core/pkg/cache"
)

type Service struct {
	db    *gorm.DB
	cache cache.Cache
}

func NewService(db *gorm.DB, caches ...cache.Cache) *Service {
	var c cache.Cache = cache.NewNoop()
	if len(caches) > 0 && caches[0] != nil {
		c = caches[0]
	}
	return &Service{db: db, cache: c}
}

func (s *Service) Hello() HelloResponse {
	return HelloResponse{
		Text: "Hello World",
	}
}

func (s *Service) Bootstrap(ctx context.Context) (BootstrapResponse, error) {
	resp := BootstrapResponse{
		Categories: make([]BootstrapCategory, 0),
	}

	settingService := systemsetting.NewService(s.db, s.cache)

	shopConfig, _ := settingService.GetShop(ctx)
	resp.Shop = BootstrapShop{
		Name:                shopConfig.Name,
		Logo:                shopConfig.Logo,
		Description:         shopConfig.Description,
		ServiceName:         shopConfig.ServiceName,
		ServicePhone:        shopConfig.ServicePhone,
		ServiceWechat:       shopConfig.ServiceWechat,
		ServiceWechatQRCode: shopConfig.ServiceWechatQRCode,
	}

	siteConfig, _ := settingService.GetSite(ctx)
	resp.Site = BootstrapSite{
		Title:       siteConfig.Title,
		Keywords:    siteConfig.Keywords,
		Description: siteConfig.Description,
		ICP:         siteConfig.ICP,
		Copyright:   siteConfig.Copyright,
	}

	mpConfig, _ := settingService.GetPublicMp(ctx)
	resp.Mp = BootstrapMp{
		Name:       mpConfig.Name,
		OriginalID: mpConfig.OriginalID,
		QRCode:     mpConfig.QRCode,
	}

	if s.db == nil {
		return resp, nil
	}

	var rows []model.ProductCategory
	err := s.db.WithContext(ctx).
		Where("status = ? AND deleted_at IS NULL", 1).
		Order("sort ASC, id ASC").
		Find(&rows).Error
	if err != nil {
		return resp, err
	}

	imageIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row.ImageID != nil && *row.ImageID > 0 {
			imageIDs = append(imageIDs, *row.ImageID)
		}
	}

	urlMap := make(map[int64]string)
	if len(imageIDs) > 0 {
		resolved, err := systemfile.NewService(s.db).Resolve(ctx, systemfile.IDsRequest{IDs: imageIDs})
		if err == nil {
			for _, file := range resolved {
				urlMap[file.ID] = file.URL
			}
		}
	}

	for _, row := range rows {
		var imgURL string
		if row.ImageID != nil {
			imgURL = urlMap[*row.ImageID]
		}
		resp.Categories = append(resp.Categories, BootstrapCategory{
			ID:       row.ID,
			ParentID: row.ParentID,
			Name:     row.Name,
			ImageURL: imgURL,
			Sort:     row.Sort,
		})
	}

	return resp, nil
}
