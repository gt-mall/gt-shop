package file

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"mall/admin_core/internal/model"
)

type DAO struct{ db *gorm.DB }

func (d *DAO) query(ctx context.Context) *gorm.DB { return d.db.WithContext(ctx) }

func (d *DAO) List(ctx context.Context, req ListRequest) ([]model.SystemFile, int64, error) {
	// Material management is strictly isolated to admin_material scene
	q := d.query(ctx).Model(&model.SystemFile{}).Where("scene = ?", model.SceneAdminMaterial)
	if req.GroupID != nil {
		q = q.Where("group_id = ?", *req.GroupID)
	}
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		q = q.Where("name LIKE ?", "%"+keyword+"%")
	}
	if fileType := strings.TrimSpace(req.FileType); fileType != "" && fileType != "all" {
		q = q.Where("file_type = ?", fileType)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []model.SystemFile{}
	err := q.Order("id DESC").Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Find(&rows).Error
	return rows, total, err
}

func (d *DAO) Resolve(ctx context.Context, ids []int64) ([]model.SystemFile, error) {
	rows := []model.SystemFile{}
	err := d.query(ctx).Unscoped().Where("id IN ?", ids).Find(&rows).Error
	return rows, err
}

func lockGroup(tx *gorm.DB, id int64) error {
	if id == 0 {
		return nil
	}
	var group model.SystemFileGroup
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGroupNotFound
		}
		return err
	}
	return nil
}

func (d *DAO) Create(ctx context.Context, row *model.SystemFile) error {
	return d.query(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockGroup(tx, row.GroupID); err != nil {
			return err
		}
		return tx.Create(row).Error
	})
}

type groupStat struct {
	GroupID     int64 `gorm:"column:group_id"`
	Count       int64 `gorm:"column:count"`
	StorageSize int64 `gorm:"column:storage_size"`
}

func (d *DAO) Groups(ctx context.Context) ([]GroupResponse, error) {
	rows := []GroupResponse{}
	err := d.query(ctx).Table("system_file_group AS g").
		Select("g.id, g.name, COUNT(f.id) AS count, COALESCE(SUM(f.size), 0) AS storage_size").
		Joins("LEFT JOIN system_file AS f ON f.group_id = g.id AND f.deleted_at IS NULL AND f.scene = ?", model.SceneAdminMaterial).
		Group("g.id, g.name").
		Order("g.id").
		Scan(&rows).Error
	return rows, err
}

func (d *DAO) GroupCounts(ctx context.Context) (GroupCountsResponse, error) {
	var resp GroupCountsResponse
	resp.Groups = []GroupResponse{}

	var stats []groupStat
	err := d.query(ctx).Model(&model.SystemFile{}).
		Where("scene = ?", model.SceneAdminMaterial).
		Select("group_id, COUNT(1) AS count, COALESCE(SUM(size), 0) AS storage_size").
		Group("group_id").
		Scan(&stats).Error
	if err != nil {
		return resp, err
	}

	statMap := make(map[int64]groupStat, len(stats))
	for _, s := range stats {
		statMap[s.GroupID] = s
		resp.Total += s.Count
		resp.TotalStorageSize += s.StorageSize
		if s.GroupID == 0 {
			resp.Ungrouped = s.Count
			resp.UngroupedStorageSize = s.StorageSize
		}
	}

	var groupModels []model.SystemFileGroup
	if err := d.query(ctx).Order("id ASC").Find(&groupModels).Error; err != nil {
		return resp, err
	}

	for _, g := range groupModels {
		s := statMap[g.ID]
		resp.Groups = append(resp.Groups, GroupResponse{
			ID:          g.ID,
			Name:        g.Name,
			Count:       s.Count,
			StorageSize: s.StorageSize,
		})
	}

	return resp, nil
}

func (d *DAO) SaveGroup(ctx context.Context, req GroupSaveRequest) error {
	if req.ID == 0 {
		return d.query(ctx).Create(&model.SystemFileGroup{Name: req.Name}).Error
	}
	return d.query(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockGroup(tx, req.ID); err != nil {
			return err
		}
		return tx.Model(&model.SystemFileGroup{}).Where("id = ?", req.ID).Update("name", req.Name).Error
	})
}

func (d *DAO) DeleteGroup(ctx context.Context, id int64) error {
	return d.query(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockGroup(tx, id); err != nil {
			return err
		}
		if err := tx.Unscoped().Model(&model.SystemFile{}).Where("group_id = ?", id).Update("group_id", 0).Error; err != nil {
			return err
		}
		return tx.Delete(&model.SystemFileGroup{}, id).Error
	})
}

func (d *DAO) Rename(ctx context.Context, req RenameRequest) error {
	result := d.query(ctx).Model(&model.SystemFile{}).Where("id = ?", req.ID).Update("name", req.Name)
	return result.Error
}

func (d *DAO) Move(ctx context.Context, req MoveRequest) error {
	return d.query(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockGroup(tx, req.GroupID); err != nil {
			return err
		}
		return tx.Model(&model.SystemFile{}).Where("id IN ?", req.IDs).Update("group_id", req.GroupID).Error
	})
}

func (d *DAO) Delete(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	// Check if any file is in use by after-sale proof
	if d.query(ctx).Migrator().HasTable("after_sale") {
		var afterSaleProofs []string
		if err := d.query(ctx).Table("after_sale").
			Where("proof_file_ids IS NOT NULL").
			Pluck("proof_file_ids", &afterSaleProofs).Error; err == nil {
			idSet := make(map[int64]bool, len(ids))
			for _, id := range ids {
				idSet[id] = true
			}
			for _, proofJSON := range afterSaleProofs {
				if strings.TrimSpace(proofJSON) == "" || proofJSON == "null" {
					continue
				}
				var pids []int64
				if err := json.Unmarshal([]byte(proofJSON), &pids); err == nil {
					for _, pid := range pids {
						if idSet[pid] {
							return ErrFileInUse
						}
					}
				}
			}
		}
	}

	return d.query(ctx).Where("id IN ?", ids).Delete(&model.SystemFile{}).Error
}

func (d *DAO) UsedStorages(ctx context.Context) ([]string, error) {
	var values []string
	err := d.query(ctx).Unscoped().Model(&model.SystemFile{}).Distinct().Pluck("storage", &values).Error
	return values, err
}
