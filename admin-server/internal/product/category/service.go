package category

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
	"mall/admin_core/pkg/cache"
)

var (
	ErrDatabaseUnavailable    = errors.New("数据库不可用")
	ErrCategoryNotFound       = errors.New("分类不存在")
	ErrParentCategoryNotFound = errors.New("父级分类不存在")
	ErrParentCategorySelf     = errors.New("父级分类不能是自身")
	ErrParentCategoryCycle    = errors.New("父级分类不能是其子分类")
	ErrInvalidName            = errors.New("分类名称不能为空且最多255个字符")
	ErrInvalidStatus          = errors.New("分类状态无效")
	ErrHasSubcategories       = errors.New("该分类下存在子分类，无法删除")
	ErrCategoryInUse          = errors.New("该分类下存在关联商品，无法删除")
	ErrImageNotFound          = errors.New("指定的图片素材不存在或已被删除")
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

func (s *Service) invalidateProductListCache(op string) {
	if s.cache == nil || s.cache.IsDisabled() {
		return
	}
	bgCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if _, err := cache.BumpProductListGeneration(bgCtx, s.cache); err != nil {
		log.Printf("[Cache] category %s: bump product list generation failed: %v", op, err)
	}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (CategoryResponse, error) {
	if s.db == nil {
		return CategoryResponse{}, ErrDatabaseUnavailable
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 255 {
		return CategoryResponse{}, ErrInvalidName
	}

	if req.ParentID < 0 {
		req.ParentID = 0
	}

	var imageID *int64
	if req.ImageID != nil && *req.ImageID > 0 {
		if err := validateImage(s.db.WithContext(ctx), *req.ImageID); err != nil {
			return CategoryResponse{}, err
		}
		id := *req.ImageID
		imageID = &id
	}

	status := int8(1)
	if req.Status != nil {
		if *req.Status != 0 && *req.Status != 1 {
			return CategoryResponse{}, ErrInvalidStatus
		}
		status = *req.Status
	}

	sort := 0
	if req.Sort != nil {
		sort = *req.Sort
	}

	cat := model.ProductCategory{
		ParentID: req.ParentID,
		Name:     name,
		ImageID:  imageID,
		Sort:     sort,
		Status:   status,
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		categories, err := lockCategoryTree(tx)
		if err != nil {
			return err
		}
		if cat.ParentID > 0 {
			if _, ok := categories[cat.ParentID]; !ok {
				return ErrParentCategoryNotFound
			}
		}
		return tx.Create(&cat).Error
	}); err != nil {
		return CategoryResponse{}, fmt.Errorf("create category: %w", err)
	}

	s.invalidateProductListCache("create")

	var imageURL string
	if cat.ImageID != nil && *cat.ImageID > 0 {
		resolved, err := systemfile.NewService(s.db).Resolve(ctx, systemfile.IDsRequest{IDs: []int64{*cat.ImageID}})
		if err == nil && len(resolved) > 0 {
			imageURL = resolved[0].URL
		}
	}

	return CategoryResponse{
		ID:        cat.ID,
		ParentID:  cat.ParentID,
		Name:      cat.Name,
		ImageID:   cat.ImageID,
		ImageURL:  imageURL,
		Sort:      cat.Sort,
		Status:    cat.Status,
		CreatedAt: cat.CreatedAt,
		UpdatedAt: cat.UpdatedAt,
	}, nil
}

func (s *Service) Update(ctx context.Context, rawBody []byte, req UpdateRequest) (CategoryResponse, error) {
	if s.db == nil {
		return CategoryResponse{}, ErrDatabaseUnavailable
	}

	var cat model.ProductCategory
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		categories, err := lockCategoryTree(tx)
		if err != nil {
			return err
		}
		var ok bool
		cat, ok = categories[req.ID]
		if !ok {
			return ErrCategoryNotFound
		}

		updates := make(map[string]any)

		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if name == "" || utf8.RuneCountInString(name) > 255 {
				return ErrInvalidName
			}
			updates["name"] = name
			cat.Name = name
		}

		if req.ParentID != nil {
			newParentID := *req.ParentID
			if newParentID == cat.ID {
				return ErrParentCategorySelf
			}
			if newParentID > 0 {
				if _, ok := categories[newParentID]; !ok {
					return ErrParentCategoryNotFound
				}
				if err := checkCycle(categories, cat.ID, newParentID); err != nil {
					return err
				}
			} else {
				newParentID = 0
			}
			updates["parent_id"] = newParentID
			cat.ParentID = newParentID
		}

		hasImageID := hasField(rawBody, "image_id")
		if hasImageID {
			if req.ImageID != nil && *req.ImageID > 0 {
				newID := *req.ImageID
				if cat.ImageID == nil || *cat.ImageID != newID {
					if err := validateImage(tx, newID); err != nil {
						return err
					}
				}
				updates["image_id"] = newID
				cat.ImageID = &newID
			} else {
				updates["image_id"] = nil
				cat.ImageID = nil
			}
		}

		if req.Sort != nil {
			updates["sort"] = *req.Sort
			cat.Sort = *req.Sort
		}

		if req.Status != nil {
			if *req.Status != 0 && *req.Status != 1 {
				return ErrInvalidStatus
			}
			updates["status"] = *req.Status
			cat.Status = *req.Status
		}

		if len(updates) > 0 {
			if err := tx.Model(&model.ProductCategory{}).
				Where("id = ?", cat.ID).Updates(updates).Error; err != nil {
				return fmt.Errorf("update category: %w", err)
			}
			// Refresh updated_at
			if err := tx.Select("updated_at").First(&cat, cat.ID).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return CategoryResponse{}, err
	}

	s.invalidateProductListCache("update")

	var imageURL string
	if cat.ImageID != nil && *cat.ImageID > 0 {
		resolved, err := systemfile.NewService(s.db).Resolve(ctx, systemfile.IDsRequest{IDs: []int64{*cat.ImageID}})
		if err == nil && len(resolved) > 0 {
			imageURL = resolved[0].URL
		}
	}

	return CategoryResponse{
		ID:        cat.ID,
		ParentID:  cat.ParentID,
		Name:      cat.Name,
		ImageID:   cat.ImageID,
		ImageURL:  imageURL,
		Sort:      cat.Sort,
		Status:    cat.Status,
		CreatedAt: cat.CreatedAt,
		UpdatedAt: cat.UpdatedAt,
	}, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if s.db == nil {
		return ErrDatabaseUnavailable
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		categories, err := lockCategoryTree(tx)
		if err != nil {
			return err
		}
		cat, ok := categories[id]
		if !ok {
			return ErrCategoryNotFound
		}
		for _, child := range categories {
			if child.ParentID == id {
				return ErrHasSubcategories
			}
		}

		var productCount int64
		if err := tx.Model(&model.Product{}).
			Where("category_id = ?", id).Count(&productCount).Error; err != nil {
			return err
		}
		if productCount > 0 {
			return ErrCategoryInUse
		}
		return tx.Delete(&cat).Error
	})
	if err != nil {
		return err
	}

	s.invalidateProductListCache("delete")
	return nil
}

func (s *Service) List(ctx context.Context) ([]CategoryResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if s.db == nil {
		return nil, ErrDatabaseUnavailable
	}

	var rows []model.ProductCategory
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL").
		Order("sort ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
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

	result := make([]CategoryResponse, 0, len(rows))
	for _, row := range rows {
		var imgURL string
		if row.ImageID != nil {
			imgURL = urlMap[*row.ImageID]
		}
		result = append(result, CategoryResponse{
			ID:        row.ID,
			ParentID:  row.ParentID,
			Name:      row.Name,
			ImageID:   row.ImageID,
			ImageURL:  imgURL,
			Sort:      row.Sort,
			Status:    row.Status,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}

	return result, nil
}

func (s *Service) Options(ctx context.Context) ([]OptionResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if s.db == nil {
		return nil, ErrDatabaseUnavailable
	}

	var rows []model.ProductCategory
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL").
		Order("sort ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]OptionResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, OptionResponse{
			ID:       row.ID,
			ParentID: row.ParentID,
			Name:     row.Name,
			Status:   row.Status,
			Sort:     row.Sort,
		})
	}

	return result, nil
}

// Category trees are small and changed infrequently. Lock every row in ID order
// before checking or changing the hierarchy, so concurrent transactions (including
// other server instances) validate the latest committed tree. Keep deleted rows
// in the locking query so all writers acquire the same first row.
func lockCategoryTree(tx *gorm.DB) (map[int64]model.ProductCategory, error) {
	var rows []model.ProductCategory
	if err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
		Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	categories := make(map[int64]model.ProductCategory, len(rows))
	for _, row := range rows {
		if !row.DeletedAt.Valid {
			categories[row.ID] = row
		}
	}
	return categories, nil
}

func validateImage(db *gorm.DB, imageID int64) error {
	var count int64
	if err := db.Model(&model.SystemFile{}).
		Where("id = ? AND deleted_at IS NULL AND scene = ? AND file_type = ? AND visibility = ?",
			imageID, model.SceneAdminMaterial, model.FileTypeImage, model.VisibilityPublic).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrImageNotFound
	}
	return nil
}

func checkCycle(categories map[int64]model.ProductCategory, targetID, newParentID int64) error {
	visited := make(map[int64]bool)
	for curr := newParentID; curr > 0; {
		if curr == targetID || visited[curr] {
			return ErrParentCategoryCycle
		}
		visited[curr] = true
		parent, ok := categories[curr]
		if !ok {
			return ErrParentCategoryNotFound
		}
		curr = parent.ParentID
	}
	return nil
}

func hasField(raw []byte, field string) bool {
	if len(raw) == 0 {
		return false
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	_, ok := m[field]
	return ok
}
