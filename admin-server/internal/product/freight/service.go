package freight

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	"mall/admin_core/internal/model"
	"mall/admin_core/internal/region"
	"mall/admin_core/pkg/util"
)

var (
	ErrDatabaseUnavailable = errors.New("数据库服务不可用")
	ErrTemplateNotFound    = errors.New("运费模板不存在")
	ErrTemplateDisabled    = errors.New("运费模板已停用/不可用")
	ErrTemplateInUse       = errors.New("该运费模板正在被商品使用，无法删除")
	ErrInvalidTemplateName = errors.New("模板名称不能为空且不超过100个字符")
	ErrInvalidBaseFee      = errors.New("默认运费格式错误")
	ErrDuplicateRule       = errors.New("地区规则存在重复配置")
	ErrConflictRule        = errors.New("同一省市不能同时配置附加运费和不配送")
	ErrInvalidRuleFee      = errors.New("附加运费必须是非负合法金额")
	ErrInvalidProvinceCode = errors.New("省份编码不能为空")
)

type Service struct {
	db        *gorm.DB
	regionSvc *region.Service
}

func NewService(db *gorm.DB) *Service {
	regSvc, _ := region.NewService()
	return &Service{
		db:        db,
		regionSvc: regSvc,
	}
}

func (s *Service) SetRegionService(regSvc *region.Service) {
	s.regionSvc = regSvc
}

func (s *Service) GetRegions() []region.RegionNode {
	if s.regionSvc == nil {
		return nil
	}
	return s.regionSvc.GetRegions()
}

func (s *Service) validateRegionRule(pCode, cCode string) error {
	if s.regionSvc == nil {
		return nil
	}
	regions := s.regionSvc.GetRegions()
	var provNode *region.RegionNode
	for i := range regions {
		if regions[i].Code == pCode {
			provNode = &regions[i]
			break
		}
	}
	if provNode == nil {
		return fmt.Errorf("无效的省份编码: %s", pCode)
	}
	if cCode != "" && cCode != "0" {
		foundCity := false
		for i := range provNode.Children {
			if provNode.Children[i].Code == cCode {
				foundCity = true
				break
			}
		}
		if !foundCity {
			return fmt.Errorf("无效的城市编码 [%s] 或该城市不属于省份 [%s(%s)]", cCode, provNode.Name, pCode)
		}
	}
	return nil
}

// ValidateRules validates base fee, extra fee rules and no delivery rules,
// returning normalized base fee, extra fees, and no delivery rules with whitespace trimmed.
func (s *Service) ValidateRules(baseFee string, extraFees []ExtraFeeRule, noDelivery []NoDeliveryRule) (string, []ExtraFeeRule, []NoDeliveryRule, error) {
	normBaseFee, err := util.NormalizeDecimalYuan(baseFee)
	if err != nil {
		return "", nil, nil, fmt.Errorf("%w: %v", ErrInvalidBaseFee, err)
	}

	normExtraFees := make([]ExtraFeeRule, 0, len(extraFees))
	extraFeeKeySet := make(map[string]struct{}, len(extraFees))
	for _, ef := range extraFees {
		pCode := strings.TrimSpace(ef.ProvinceCode)
		cCode := strings.TrimSpace(ef.CityCode)
		if pCode == "" {
			return "", nil, nil, ErrInvalidProvinceCode
		}
		if err := s.validateRegionRule(pCode, cCode); err != nil {
			return "", nil, nil, err
		}
		normFee, err := util.NormalizeDecimalYuan(ef.Fee)
		if err != nil {
			return "", nil, nil, fmt.Errorf("%w: %v", ErrInvalidRuleFee, err)
		}

		key := pCode + "#" + cCode
		if _, exists := extraFeeKeySet[key]; exists {
			return "", nil, nil, fmt.Errorf("%w: 省份 %s 城市 %s 在附加运费中重复", ErrDuplicateRule, pCode, cCode)
		}
		extraFeeKeySet[key] = struct{}{}
		normExtraFees = append(normExtraFees, ExtraFeeRule{
			ProvinceCode: pCode,
			CityCode:     cCode,
			Fee:          normFee,
		})
	}

	normNoDelivery := make([]NoDeliveryRule, 0, len(noDelivery))
	noDelKeySet := make(map[string]struct{}, len(noDelivery))
	for _, nd := range noDelivery {
		pCode := strings.TrimSpace(nd.ProvinceCode)
		cCode := strings.TrimSpace(nd.CityCode)
		if pCode == "" {
			return "", nil, nil, ErrInvalidProvinceCode
		}
		if err := s.validateRegionRule(pCode, cCode); err != nil {
			return "", nil, nil, err
		}

		key := pCode + "#" + cCode
		if _, exists := noDelKeySet[key]; exists {
			return "", nil, nil, fmt.Errorf("%w: 省份 %s 城市 %s 在不配送地区中重复", ErrDuplicateRule, pCode, cCode)
		}
		noDelKeySet[key] = struct{}{}

		if _, conflict := extraFeeKeySet[key]; conflict {
			return "", nil, nil, fmt.Errorf("%w: 省份 %s 城市 %s 同时配置了附加运费与不配送", ErrConflictRule, pCode, cCode)
		}
		normNoDelivery = append(normNoDelivery, NoDeliveryRule{
			ProvinceCode: pCode,
			CityCode:     cCode,
		})
	}

	return normBaseFee, normExtraFees, normNoDelivery, nil
}

// CalculateTemplateFreight calculates freight for a specific freight template given province and city codes.
func CalculateTemplateFreight(tmpl *model.FreightTemplate, provinceCode, cityCode string) (int64, bool, error) {
	if tmpl == nil {
		return 0, false, ErrTemplateNotFound
	}
	if tmpl.Status != 1 {
		return 0, false, ErrTemplateDisabled
	}

	provinceCode = strings.TrimSpace(provinceCode)
	cityCode = strings.TrimSpace(cityCode)

	// 1. Check no_delivery rules
	if tmpl.NoDelivery != nil && *tmpl.NoDelivery != "" {
		var noDeliveryRules []NoDeliveryRule
		if err := json.Unmarshal([]byte(*tmpl.NoDelivery), &noDeliveryRules); err == nil {
			// Priority 1: City level no delivery
			if cityCode != "" {
				for _, r := range noDeliveryRules {
					if strings.TrimSpace(r.ProvinceCode) == provinceCode && strings.TrimSpace(r.CityCode) == cityCode {
						return 0, false, nil
					}
				}
			}
			// Priority 2: Province level no delivery
			for _, r := range noDeliveryRules {
				rPCode := strings.TrimSpace(r.ProvinceCode)
				rCCode := strings.TrimSpace(r.CityCode)
				if rPCode == provinceCode && (rCCode == "" || rCCode == "0") {
					return 0, false, nil
				}
			}
		}
	}

	baseCents, err := util.ParseDecimalYuanToCent(tmpl.BaseFee)
	if err != nil {
		return 0, false, fmt.Errorf("解析模板默认运费失败: %w", err)
	}

	// 2. Check extra_fees rules
	if tmpl.ExtraFees != nil && *tmpl.ExtraFees != "" {
		var extraFeeRules []ExtraFeeRule
		if err := json.Unmarshal([]byte(*tmpl.ExtraFees), &extraFeeRules); err == nil {
			// Priority 3: City level extra fee
			if cityCode != "" {
				for _, r := range extraFeeRules {
					if strings.TrimSpace(r.ProvinceCode) == provinceCode && strings.TrimSpace(r.CityCode) == cityCode {
						extraCents, err := util.ParseDecimalYuanToCent(r.Fee)
						if err != nil {
							return 0, false, fmt.Errorf("解析城市附加运费失败: %w", err)
						}
						return baseCents + extraCents, true, nil
					}
				}
			}
			// Priority 4: Province level extra fee
			for _, r := range extraFeeRules {
				rPCode := strings.TrimSpace(r.ProvinceCode)
				rCCode := strings.TrimSpace(r.CityCode)
				if rPCode == provinceCode && (rCCode == "" || rCCode == "0") {
					extraCents, err := util.ParseDecimalYuanToCent(r.Fee)
					if err != nil {
						return 0, false, fmt.Errorf("解析省份附加运费失败: %w", err)
					}
					return baseCents + extraCents, true, nil
				}
			}
		}
	}

	// Priority 5: base_fee
	return baseCents, true, nil
}

// CalculateProductFreight calculates freight for a single product.
func CalculateProductFreight(prod *model.Product, tmpl *model.FreightTemplate, provinceCode, cityCode string) (int64, bool, error) {
	if prod == nil {
		return 0, false, errors.New("商品不存在")
	}

	switch prod.FreightType {
	case 1:
		// Free shipping
		return 0, true, nil
	case 2:
		// Fixed freight fee
		feeCents, err := util.ParseDecimalYuanToCent(prod.FreightFee)
		if err != nil {
			return 0, false, fmt.Errorf("商品固定运费格式错误: %w", err)
		}
		return feeCents, true, nil
	case 3:
		// Freight template
		if tmpl == nil {
			return 0, false, ErrTemplateNotFound
		}
		return CalculateTemplateFreight(tmpl, provinceCode, cityCode)
	default:
		return 0, false, fmt.Errorf("未知的商品配送方式: %d", prod.FreightType)
	}
}

func (s *Service) Create(ctx context.Context, req FreightTemplateCreateRequest) (int64, error) {
	if s.db == nil {
		return 0, ErrDatabaseUnavailable
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return 0, ErrInvalidTemplateName
	}

	normBaseFee, normExtraFees, normNoDelivery, err := s.ValidateRules(req.BaseFee, req.ExtraFees, req.NoDelivery)
	if err != nil {
		return 0, err
	}

	status := int8(1)
	if req.Status != nil {
		if *req.Status != 0 && *req.Status != 1 {
			return 0, errors.New("状态值无效")
		}
		status = *req.Status
	}

	var extraFeesJSON *string
	if len(normExtraFees) > 0 {
		b, err := json.Marshal(normExtraFees)
		if err != nil {
			return 0, err
		}
		str := string(b)
		extraFeesJSON = &str
	}

	var noDeliveryJSON *string
	if len(normNoDelivery) > 0 {
		b, err := json.Marshal(normNoDelivery)
		if err != nil {
			return 0, err
		}
		str := string(b)
		noDeliveryJSON = &str
	}

	tmpl := model.FreightTemplate{
		Name:       name,
		BaseFee:    normBaseFee,
		ExtraFees:  extraFeesJSON,
		NoDelivery: noDeliveryJSON,
		Status:     status,
	}

	if err := s.db.WithContext(ctx).Select("Name", "BaseFee", "ExtraFees", "NoDelivery", "Status").Create(&tmpl).Error; err != nil {
		return 0, err
	}

	return tmpl.ID, nil
}

func (s *Service) Update(ctx context.Context, req FreightTemplateUpdateRequest) error {
	if s.db == nil {
		return ErrDatabaseUnavailable
	}
	if req.ID <= 0 {
		return errors.New("模板ID无效")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return ErrInvalidTemplateName
	}

	normBaseFee, normExtraFees, normNoDelivery, err := s.ValidateRules(req.BaseFee, req.ExtraFees, req.NoDelivery)
	if err != nil {
		return err
	}

	var existing model.FreightTemplate
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", req.ID).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTemplateNotFound
		}
		return err
	}

	status := existing.Status
	if req.Status != nil {
		if *req.Status != 0 && *req.Status != 1 {
			return errors.New("状态值无效")
		}
		status = *req.Status
	}

	var extraFeesJSON *string
	if len(normExtraFees) > 0 {
		b, err := json.Marshal(normExtraFees)
		if err != nil {
			return err
		}
		str := string(b)
		extraFeesJSON = &str
	}

	var noDeliveryJSON *string
	if len(normNoDelivery) > 0 {
		b, err := json.Marshal(normNoDelivery)
		if err != nil {
			return err
		}
		str := string(b)
		noDeliveryJSON = &str
	}

	updates := map[string]interface{}{
		"name":        name,
		"base_fee":    normBaseFee,
		"extra_fees":  extraFeesJSON,
		"no_delivery": noDeliveryJSON,
		"status":      status,
	}

	return s.db.WithContext(ctx).Model(&model.FreightTemplate{}).Where("id = ?", req.ID).Updates(updates).Error
}

func (s *Service) Status(ctx context.Context, id int64, status int8) error {
	if s.db == nil {
		return ErrDatabaseUnavailable
	}
	if status != 0 && status != 1 {
		return errors.New("状态值无效")
	}

	var existing model.FreightTemplate
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTemplateNotFound
		}
		return err
	}

	return s.db.WithContext(ctx).Model(&model.FreightTemplate{}).Where("id = ?", id).Update("status", status).Error
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if s.db == nil {
		return ErrDatabaseUnavailable
	}

	var existing model.FreightTemplate
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTemplateNotFound
		}
		return err
	}

	// Check if any active (non-deleted) product is using this template
	var prodCount int64
	if err := s.db.WithContext(ctx).Model(&model.Product{}).
		Where("freight_template_id = ? AND deleted_at IS NULL", id).
		Count(&prodCount).Error; err != nil {
		return err
	}
	if prodCount > 0 {
		return ErrTemplateInUse
	}

	return s.db.WithContext(ctx).Delete(&model.FreightTemplate{}, id).Error
}

func (s *Service) Detail(ctx context.Context, id int64) (FreightTemplateDetail, error) {
	if s.db == nil {
		return FreightTemplateDetail{}, ErrDatabaseUnavailable
	}

	var tmpl model.FreightTemplate
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&tmpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return FreightTemplateDetail{}, ErrTemplateNotFound
		}
		return FreightTemplateDetail{}, err
	}

	var extraFees []ExtraFeeRule
	if tmpl.ExtraFees != nil && *tmpl.ExtraFees != "" {
		_ = json.Unmarshal([]byte(*tmpl.ExtraFees), &extraFees)
	}
	if extraFees == nil {
		extraFees = []ExtraFeeRule{}
	}

	var noDelivery []NoDeliveryRule
	if tmpl.NoDelivery != nil && *tmpl.NoDelivery != "" {
		_ = json.Unmarshal([]byte(*tmpl.NoDelivery), &noDelivery)
	}
	if noDelivery == nil {
		noDelivery = []NoDeliveryRule{}
	}

	return FreightTemplateDetail{
		ID:         tmpl.ID,
		Name:       tmpl.Name,
		BaseFee:    tmpl.BaseFee,
		ExtraFees:  extraFees,
		NoDelivery: noDelivery,
		Status:     tmpl.Status,
		CreatedAt:  tmpl.CreatedAt,
		UpdatedAt:  tmpl.UpdatedAt,
	}, nil
}

func (s *Service) List(ctx context.Context, req FreightTemplateListRequest) ([]FreightTemplateRow, int64, error) {
	if s.db == nil {
		return nil, 0, ErrDatabaseUnavailable
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	q := s.db.WithContext(ctx).Model(&model.FreightTemplate{}).Where("deleted_at IS NULL")

	if req.Keyword != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		q = q.Where("name LIKE ?", kw)
	}

	if req.Status != nil {
		q = q.Where("status = ?", *req.Status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []model.FreightTemplate
	offset := (page - 1) * pageSize
	if err := q.Order("id DESC").Limit(pageSize).Offset(offset).Find(&list).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]FreightTemplateRow, 0, len(list))
	for _, item := range list {
		extraCount := 0
		if item.ExtraFees != nil && *item.ExtraFees != "" {
			var rules []ExtraFeeRule
			if err := json.Unmarshal([]byte(*item.ExtraFees), &rules); err == nil {
				extraCount = len(rules)
			}
		}

		noDelCount := 0
		if item.NoDelivery != nil && *item.NoDelivery != "" {
			var rules []NoDeliveryRule
			if err := json.Unmarshal([]byte(*item.NoDelivery), &rules); err == nil {
				noDelCount = len(rules)
			}
		}

		rows = append(rows, FreightTemplateRow{
			ID:              item.ID,
			Name:            item.Name,
			BaseFee:         item.BaseFee,
			ExtraFeeCount:   extraCount,
			NoDeliveryCount: noDelCount,
			Status:          item.Status,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
		})
	}

	return rows, total, nil
}

func (s *Service) Options(ctx context.Context) ([]FreightTemplateOption, error) {
	if s.db == nil {
		return nil, ErrDatabaseUnavailable
	}

	var list []model.FreightTemplate
	if err := s.db.WithContext(ctx).Where("status = 1 AND deleted_at IS NULL").Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}

	opts := make([]FreightTemplateOption, 0, len(list))
	for _, item := range list {
		var extraFees []ExtraFeeRule
		if item.ExtraFees != nil && *item.ExtraFees != "" {
			_ = json.Unmarshal([]byte(*item.ExtraFees), &extraFees)
		}
		if extraFees == nil {
			extraFees = []ExtraFeeRule{}
		}

		var noDelivery []NoDeliveryRule
		if item.NoDelivery != nil && *item.NoDelivery != "" {
			_ = json.Unmarshal([]byte(*item.NoDelivery), &noDelivery)
		}
		if noDelivery == nil {
			noDelivery = []NoDeliveryRule{}
		}

		opts = append(opts, FreightTemplateOption{
			ID:         item.ID,
			Name:       item.Name,
			BaseFee:    item.BaseFee,
			Status:     item.Status,
			ExtraFees:  extraFees,
			NoDelivery: noDelivery,
		})
	}

	return opts, nil
}
