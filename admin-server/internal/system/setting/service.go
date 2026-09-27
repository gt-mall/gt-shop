package setting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"mall/admin_core/internal/model"
	"mall/admin_core/internal/system/file"
	"mall/admin_core/pkg/cache"
)

const (
	ConfigKeyShopBasic      = "shop_basic"
	ConfigKeySiteBasic      = "site_basic"
	ConfigKeyMpBasic        = "mp_basic"
	ConfigKeySearchBasic    = "search_basic"
	ConfigKeyTradeBasic     = "trade_basic"
	ConfigKeyPaymentBasic   = "payment_basic"
	ConfigKeyLogisticsBasic = "logistics_basic"
	ConfigKeyMemberBasic    = "member_basic"
	ConfigKeyReviewBasic    = "review_basic"
	ConfigKeyAgreementBasic = "agreement_basic"
)

var (
	ErrUnavailable = errors.New("database unavailable")
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

func DefaultShopConfig() ShopBasicConfig {
	return ShopBasicConfig{
		Name:        "Mall 商城",
		ServiceName: "商城客服",
		Description: "全场景高性能电商系统",
	}
}

func DefaultSiteConfig() SiteBasicConfig {
	return SiteBasicConfig{
		Title:       "Mall 管理后台",
		Keywords:    "Mall,电商系统,管理后台",
		Description: "Mall 电商管理系统",
		Copyright:   "© 2026 Mall. All rights reserved.",
	}
}

func DefaultMpConfig() MpConfig {
	return MpConfig{
		Name: "Mall 小程序",
	}
}

func (s *Service) query(ctx context.Context) *gorm.DB {
	return s.db.WithContext(ctx)
}

func (s *Service) resolveFiles(ctx context.Context, config *ShopBasicConfig) {
	if s.db == nil || config == nil {
		return
	}
	ids := make([]int64, 0, 3)
	if config.LogoID > 0 {
		ids = append(ids, config.LogoID)
	}
	if config.FaviconID > 0 {
		ids = append(ids, config.FaviconID)
	}
	if config.ServiceWechatQRCodeID > 0 {
		ids = append(ids, config.ServiceWechatQRCodeID)
	}
	if len(ids) == 0 {
		return
	}
	files, err := file.NewService(s.db).Resolve(ctx, file.IDsRequest{IDs: ids})
	if err != nil {
		return
	}
	for _, f := range files {
		if config.LogoID > 0 && f.ID == config.LogoID {
			config.Logo = f.URL
		}
		if config.FaviconID > 0 && f.ID == config.FaviconID {
			config.Favicon = f.URL
		}
		if config.ServiceWechatQRCodeID > 0 && f.ID == config.ServiceWechatQRCodeID {
			config.ServiceWechatQRCode = f.URL
		}
	}
}

func (s *Service) GetShop(ctx context.Context) (ShopBasicConfig, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	loader := func(innerCtx context.Context) (ShopBasicConfig, error) {
		config := DefaultShopConfig()
		if s.db == nil {
			return config, ErrUnavailable
		}
		var row model.SystemConfig
		err := s.query(innerCtx).Where("config_key = ?", ConfigKeyShopBasic).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return config, nil
		}
		if err != nil {
			return config, err
		}
		if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
			return config, err
		}
		s.resolveFiles(innerCtx, &config)
		return config, nil
	}

	if s.cache == nil || s.cache.IsDisabled() {
		return loader(ctx)
	}

	gen, err := cache.GetSystemConfigGeneration(ctx, s.cache, ConfigKeyShopBasic)
	if err != nil {
		// "不知道当前是哪一代，就不用缓存。" 绕过 Redis 直接回源 MySQL
		return loader(ctx)
	}

	key := cache.SystemConfigKey(ConfigKeyShopBasic, gen)
	return cache.Get(ctx, s.cache, key, cache.TTLSystemConfig, loader)
}

func (s *Service) SaveShop(ctx context.Context, config ShopBasicConfig) error {
	if s.db == nil {
		return ErrUnavailable
	}
	config.Name = strings.TrimSpace(config.Name)
	config.ServiceName = strings.TrimSpace(config.ServiceName)
	config.ServicePhone = strings.TrimSpace(config.ServicePhone)
	config.ServiceWechat = strings.TrimSpace(config.ServiceWechat)
	config.Description = strings.TrimSpace(config.Description)
	if config.LogoID == 0 {
		config.Logo = ""
	}
	if config.FaviconID == 0 {
		config.Favicon = ""
	}
	if config.ServiceWechatQRCodeID == 0 {
		config.ServiceWechatQRCode = ""
	}
	s.resolveFiles(ctx, &config)

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := model.SystemConfig{
		ConfigKey: ConfigKeyShopBasic,
		Value:     string(data),
	}
	if err := s.query(ctx).
		Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "config_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error; err != nil {
		return err
	}

	if s.cache != nil && !s.cache.IsDisabled() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if _, err := cache.BumpSystemConfigGeneration(bgCtx, s.cache, ConfigKeyShopBasic); err != nil {
			log.Printf("[Setting] failed to bump shop config generation: %v", err)
		}
	}
	return nil
}

func (s *Service) GetSite(ctx context.Context) (SiteBasicConfig, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	loader := func(innerCtx context.Context) (SiteBasicConfig, error) {
		config := DefaultSiteConfig()
		if s.db == nil {
			return config, ErrUnavailable
		}
		var row model.SystemConfig
		err := s.query(innerCtx).Where("config_key = ?", ConfigKeySiteBasic).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return config, nil
		}
		if err != nil {
			return config, err
		}
		if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
			return config, err
		}
		return config, nil
	}

	if s.cache == nil || s.cache.IsDisabled() {
		return loader(ctx)
	}

	gen, err := cache.GetSystemConfigGeneration(ctx, s.cache, ConfigKeySiteBasic)
	if err != nil {
		// "不知道当前是哪一代，就不用缓存。" 绕过 Redis 直接回源 MySQL
		return loader(ctx)
	}

	key := cache.SystemConfigKey(ConfigKeySiteBasic, gen)
	return cache.Get(ctx, s.cache, key, cache.TTLSystemConfig, loader)
}

func (s *Service) SaveSite(ctx context.Context, config SiteBasicConfig) error {
	if s.db == nil {
		return ErrUnavailable
	}
	config.Title = strings.TrimSpace(config.Title)
	config.Keywords = strings.TrimSpace(config.Keywords)
	config.Description = strings.TrimSpace(config.Description)
	config.ICP = strings.TrimSpace(config.ICP)
	config.PoliceRecord = strings.TrimSpace(config.PoliceRecord)
	config.Copyright = strings.TrimSpace(config.Copyright)

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := model.SystemConfig{
		ConfigKey: ConfigKeySiteBasic,
		Value:     string(data),
	}
	if err := s.query(ctx).
		Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "config_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error; err != nil {
		return err
	}

	if s.cache != nil && !s.cache.IsDisabled() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if _, err := cache.BumpSystemConfigGeneration(bgCtx, s.cache, ConfigKeySiteBasic); err != nil {
			log.Printf("[Setting] failed to bump site config generation: %v", err)
		}
	}
	return nil
}

func (s *Service) resolveMpFiles(ctx context.Context, config *MpConfig) {
	if s.db == nil || config == nil || config.QRCodeID <= 0 {
		return
	}
	files, err := file.NewService(s.db).Resolve(ctx, file.IDsRequest{IDs: []int64{config.QRCodeID}})
	if err != nil {
		return
	}
	for _, f := range files {
		if f.ID == config.QRCodeID {
			config.QRCode = f.URL
		}
	}
}

// GetMp returns full MP config directly from MySQL without caching sensitive secrets.
func (s *Service) GetMp(ctx context.Context) (MpConfig, error) {
	config := DefaultMpConfig()
	if s.db == nil {
		return config, ErrUnavailable
	}
	var row model.SystemConfig
	err := s.query(ctx).Where("config_key = ?", ConfigKeyMpBasic).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return config, nil
	}
	if err != nil {
		return config, err
	}
	if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
		return config, err
	}
	s.resolveMpFiles(ctx, &config)
	return config, nil
}

// GetPublicMp returns non-sensitive MP config, cached with generation-scoped TTL.
func (s *Service) GetPublicMp(ctx context.Context) (PublicMpConfig, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	loader := func(innerCtx context.Context) (PublicMpConfig, error) {
		fullMp, err := s.GetMp(innerCtx)
		publicConfig := PublicMpConfig{
			Name:       fullMp.Name,
			OriginalID: fullMp.OriginalID,
			QRCode:     fullMp.QRCode,
		}
		if err != nil {
			return publicConfig, err
		}
		return publicConfig, nil
	}

	if s.cache == nil || s.cache.IsDisabled() {
		return loader(ctx)
	}

	gen, err := cache.GetSystemConfigGeneration(ctx, s.cache, "public_mp")
	if err != nil {
		// "不知道当前是哪一代，就不用缓存。" 绕过 Redis 直接回源 MySQL
		return loader(ctx)
	}

	key := cache.SystemConfigKey("public_mp", gen)
	return cache.Get(ctx, s.cache, key, cache.TTLSystemConfig, loader)
}

func (s *Service) SaveMp(ctx context.Context, config MpConfig) error {
	if s.db == nil {
		return ErrUnavailable
	}
	config.Name = strings.TrimSpace(config.Name)
	config.OriginalID = strings.TrimSpace(config.OriginalID)
	config.AppID = strings.TrimSpace(config.AppID)
	config.AppSecret = strings.TrimSpace(config.AppSecret)
	if config.AppSecret == "" {
		old, err := s.GetMp(ctx)
		if err != nil {
			return err
		}
		config.AppSecret = old.AppSecret
	}
	config.RequestDomain = strings.TrimSpace(config.RequestDomain)
	config.SocketDomain = strings.TrimSpace(config.SocketDomain)
	config.UploadDomain = strings.TrimSpace(config.UploadDomain)
	config.DownloadDomain = strings.TrimSpace(config.DownloadDomain)
	config.UDPDomain = strings.TrimSpace(config.UDPDomain)
	config.TCPDomain = strings.TrimSpace(config.TCPDomain)
	config.BusinessDomain = strings.TrimSpace(config.BusinessDomain)
	config.UploadPrivateKey = strings.TrimSpace(config.UploadPrivateKey)
	if config.UploadPrivateKey == "" {
		old, err := s.GetMp(ctx)
		if err != nil {
			return err
		}
		config.UploadPrivateKey = old.UploadPrivateKey
	}
	if config.QRCodeID == 0 {
		config.QRCode = ""
	}
	s.resolveMpFiles(ctx, &config)

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := model.SystemConfig{
		ConfigKey: ConfigKeyMpBasic,
		Value:     string(data),
	}
	if err := s.query(ctx).
		Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "config_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error; err != nil {
		return err
	}

	if s.cache != nil && !s.cache.IsDisabled() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if _, err := cache.BumpSystemConfigGeneration(bgCtx, s.cache, "public_mp"); err != nil {
			log.Printf("[Setting] failed to bump public mp config generation: %v", err)
		}
	}
	return nil
}

func DefaultSearchConfig() SearchBasicConfig {
	return SearchBasicConfig{
		HotKeywords: make([]HotKeywordItem, 0),
	}
}

func (s *Service) GetSearch(ctx context.Context) (SearchBasicConfig, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	loader := func(innerCtx context.Context) (SearchBasicConfig, error) {
		config := DefaultSearchConfig()
		if s.db == nil {
			return config, ErrUnavailable
		}
		var row model.SystemConfig
		err := s.query(innerCtx).Where("config_key = ?", ConfigKeySearchBasic).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return config, nil
		}
		if err != nil {
			return config, err
		}
		if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
			return config, err
		}
		if config.HotKeywords == nil {
			config.HotKeywords = make([]HotKeywordItem, 0)
		}
		return config, nil
	}

	if s.cache == nil || s.cache.IsDisabled() {
		return loader(ctx)
	}

	gen, err := cache.GetSystemConfigGeneration(ctx, s.cache, ConfigKeySearchBasic)
	if err != nil {
		return loader(ctx)
	}

	key := cache.SystemConfigKey(ConfigKeySearchBasic, gen)
	return cache.Get(ctx, s.cache, key, cache.TTLSystemConfig, loader)
}

func (s *Service) SaveSearch(ctx context.Context, config SearchBasicConfig) error {
	if s.db == nil {
		return ErrUnavailable
	}

	// 规范化与去重校验
	cleaned := make([]HotKeywordItem, 0, len(config.HotKeywords))
	seen := make(map[string]bool)

	for _, item := range config.HotKeywords {
		kw := strings.TrimSpace(item.Keyword)
		kw = strings.Join(strings.Fields(kw), " ")
		if kw == "" {
			continue
		}
		runes := []rune(kw)
		if len(runes) > 50 {
			kw = string(runes[:50])
		}
		lowerKw := strings.ToLower(kw)
		if seen[lowerKw] {
			continue
		}
		seen[lowerKw] = true

		cleaned = append(cleaned, HotKeywordItem{
			Keyword: kw,
			Sort:    item.Sort,
			Enabled: item.Enabled,
		})

		if len(cleaned) >= 20 {
			break
		}
	}
	config.HotKeywords = cleaned

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := model.SystemConfig{
		ConfigKey: ConfigKeySearchBasic,
		Value:     string(data),
	}
	if err := s.query(ctx).
		Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "config_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error; err != nil {
		return err
	}

	if s.cache != nil && !s.cache.IsDisabled() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if _, err := cache.BumpSystemConfigGeneration(bgCtx, s.cache, ConfigKeySearchBasic); err != nil {
			log.Printf("[Setting] failed to bump search config generation: %v", err)
		}
	}
	return nil
}

func (s *Service) GetPublicSearchConfig(ctx context.Context) (MiniappSearchConfigResponse, error) {
	cfg, err := s.GetSearch(ctx)
	if err != nil {
		return MiniappSearchConfigResponse{HotKeywords: make([]string, 0)}, err
	}

	type sortableItem struct {
		keyword string
		sort    int
	}
	enabledItems := make([]sortableItem, 0, len(cfg.HotKeywords))
	for _, item := range cfg.HotKeywords {
		if item.Enabled && item.Keyword != "" {
			enabledItems = append(enabledItems, sortableItem{
				keyword: item.Keyword,
				sort:    item.Sort,
			})
		}
	}

	sort.SliceStable(enabledItems, func(i, j int) bool {
		if enabledItems[i].sort != enabledItems[j].sort {
			return enabledItems[i].sort < enabledItems[j].sort
		}
		return enabledItems[i].keyword < enabledItems[j].keyword
	})

	result := make([]string, 0, len(enabledItems))
	for _, item := range enabledItems {
		result = append(result, item.keyword)
	}

	return MiniappSearchConfigResponse{HotKeywords: result}, nil
}

func DefaultTradeConfig() TradeBasicConfig {
	return TradeBasicConfig{
		OrderCloseMinutes:    30,
		AutoReceiveDays:      7,
		AfterSaleDays:        7,
		AllowCancelUnshipped: true,
		AutoRefundOnCancel:   true,
	}
}

func DefaultPaymentConfig() PaymentBasicConfig {
	return PaymentBasicConfig{
		WechatEnabled: true,
	}
}

func DefaultLogisticsConfig() LogisticsBasicConfig {
	return LogisticsBasicConfig{
		SenderName:    "",
		SenderPhone:   "",
		SenderAddress: "",
	}
}

func DefaultMemberConfig() MemberBasicConfig {
	return MemberBasicConfig{
		RegisterEnabled:      true,
		PasswordLoginEnabled: true,
		WechatLoginEnabled:   true,
	}
}

func DefaultReviewConfig() ReviewBasicConfig {
	return ReviewBasicConfig{
		Enabled:           true,
		NeedAudit:         true,
		DefaultAnonymous:  false,
		AutoReviewEnabled: false,
		AutoReviewDays:    7,
	}
}

func DefaultAgreementConfig() AgreementBasicConfig {
	return AgreementBasicConfig{
		UserAgreement:   "",
		PrivacyPolicy:   "",
		AfterSalePolicy: "",
	}
}

// ----------------- Trade Settings -----------------

func (s *Service) GetTrade(ctx context.Context) (TradeBasicConfig, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	loader := func(innerCtx context.Context) (TradeBasicConfig, error) {
		config := DefaultTradeConfig()
		if s.db == nil {
			return config, ErrUnavailable
		}
		var row model.SystemConfig
		err := s.query(innerCtx).Where("config_key = ?", ConfigKeyTradeBasic).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return config, nil
		}
		if err != nil {
			return config, err
		}
		if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
			return config, err
		}
		return config, nil
	}

	if s.cache == nil || s.cache.IsDisabled() {
		return loader(ctx)
	}

	gen, err := cache.GetSystemConfigGeneration(ctx, s.cache, ConfigKeyTradeBasic)
	if err != nil {
		return loader(ctx)
	}

	key := cache.SystemConfigKey(ConfigKeyTradeBasic, gen)
	return cache.Get(ctx, s.cache, key, cache.TTLSystemConfig, loader)
}

func (s *Service) SaveTrade(ctx context.Context, config TradeBasicConfig) error {
	if s.db == nil {
		return ErrUnavailable
	}
	if config.OrderCloseMinutes < 1 || config.OrderCloseMinutes > 1440 {
		return errors.New("订单超时关闭时间必须在1到1440分钟之间")
	}
	if config.AutoReceiveDays < 1 || config.AutoReceiveDays > 30 {
		return errors.New("发货后自动收货时间必须在1到30天之间")
	}
	if config.AfterSaleDays < 0 || config.AfterSaleDays > 90 {
		return errors.New("售后申请有效期限必须在0到90天之间")
	}

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := model.SystemConfig{
		ConfigKey: ConfigKeyTradeBasic,
		Value:     string(data),
	}
	if err := s.query(ctx).
		Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "config_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error; err != nil {
		return err
	}

	if s.cache != nil && !s.cache.IsDisabled() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if _, err := cache.BumpSystemConfigGeneration(bgCtx, s.cache, ConfigKeyTradeBasic); err != nil {
			log.Printf("[Setting] failed to bump trade config generation: %v", err)
		}
	}
	return nil
}

// ----------------- Payment Settings -----------------

func (s *Service) GetPayment(ctx context.Context) (PaymentBasicConfig, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	loader := func(innerCtx context.Context) (PaymentBasicConfig, error) {
		config := DefaultPaymentConfig()
		if s.db == nil {
			return config, ErrUnavailable
		}
		var row model.SystemConfig
		err := s.query(innerCtx).Where("config_key = ?", ConfigKeyPaymentBasic).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return config, nil
		}
		if err != nil {
			return config, err
		}
		if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
			return config, err
		}
		return config, nil
	}

	if s.cache == nil || s.cache.IsDisabled() {
		return loader(ctx)
	}

	gen, err := cache.GetSystemConfigGeneration(ctx, s.cache, ConfigKeyPaymentBasic)
	if err != nil {
		return loader(ctx)
	}

	key := cache.SystemConfigKey(ConfigKeyPaymentBasic, gen)
	return cache.Get(ctx, s.cache, key, cache.TTLSystemConfig, loader)
}

func (s *Service) SavePayment(ctx context.Context, config PaymentBasicConfig) error {
	if s.db == nil {
		return ErrUnavailable
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := model.SystemConfig{
		ConfigKey: ConfigKeyPaymentBasic,
		Value:     string(data),
	}
	if err := s.query(ctx).
		Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "config_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error; err != nil {
		return err
	}

	if s.cache != nil && !s.cache.IsDisabled() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if _, err := cache.BumpSystemConfigGeneration(bgCtx, s.cache, ConfigKeyPaymentBasic); err != nil {
			log.Printf("[Setting] failed to bump payment config generation: %v", err)
		}
	}
	return nil
}

// ----------------- Logistics Settings -----------------

func (s *Service) GetLogistics(ctx context.Context) (LogisticsBasicConfig, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	loader := func(innerCtx context.Context) (LogisticsBasicConfig, error) {
		config := DefaultLogisticsConfig()
		if s.db == nil {
			return config, ErrUnavailable
		}
		var row model.SystemConfig
		err := s.query(innerCtx).Where("config_key = ?", ConfigKeyLogisticsBasic).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return config, nil
		}
		if err != nil {
			return config, err
		}
		if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
			return config, err
		}
		return config, nil
	}

	if s.cache == nil || s.cache.IsDisabled() {
		return loader(ctx)
	}

	gen, err := cache.GetSystemConfigGeneration(ctx, s.cache, ConfigKeyLogisticsBasic)
	if err != nil {
		return loader(ctx)
	}

	key := cache.SystemConfigKey(ConfigKeyLogisticsBasic, gen)
	return cache.Get(ctx, s.cache, key, cache.TTLSystemConfig, loader)
}

func (s *Service) SaveLogistics(ctx context.Context, config LogisticsBasicConfig) error {
	if s.db == nil {
		return ErrUnavailable
	}
	config.SenderName = strings.TrimSpace(config.SenderName)
	config.SenderPhone = strings.TrimSpace(config.SenderPhone)
	config.SenderAddress = strings.TrimSpace(config.SenderAddress)

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := model.SystemConfig{
		ConfigKey: ConfigKeyLogisticsBasic,
		Value:     string(data),
	}
	if err := s.query(ctx).
		Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "config_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error; err != nil {
		return err
	}

	if s.cache != nil && !s.cache.IsDisabled() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if _, err := cache.BumpSystemConfigGeneration(bgCtx, s.cache, ConfigKeyLogisticsBasic); err != nil {
			log.Printf("[Setting] failed to bump logistics config generation: %v", err)
		}
	}
	return nil
}

// ----------------- Member Settings -----------------

func (s *Service) GetMember(ctx context.Context) (MemberBasicConfig, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	loader := func(innerCtx context.Context) (MemberBasicConfig, error) {
		config := DefaultMemberConfig()
		if s.db == nil {
			return config, ErrUnavailable
		}
		var row model.SystemConfig
		err := s.query(innerCtx).Where("config_key = ?", ConfigKeyMemberBasic).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return config, nil
		}
		if err != nil {
			return config, err
		}
		if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
			return config, err
		}
		return config, nil
	}

	if s.cache == nil || s.cache.IsDisabled() {
		return loader(ctx)
	}

	gen, err := cache.GetSystemConfigGeneration(ctx, s.cache, ConfigKeyMemberBasic)
	if err != nil {
		return loader(ctx)
	}

	key := cache.SystemConfigKey(ConfigKeyMemberBasic, gen)
	return cache.Get(ctx, s.cache, key, cache.TTLSystemConfig, loader)
}

func (s *Service) SaveMember(ctx context.Context, config MemberBasicConfig) error {
	if s.db == nil {
		return ErrUnavailable
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := model.SystemConfig{
		ConfigKey: ConfigKeyMemberBasic,
		Value:     string(data),
	}
	if err := s.query(ctx).
		Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "config_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error; err != nil {
		return err
	}

	if s.cache != nil && !s.cache.IsDisabled() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if _, err := cache.BumpSystemConfigGeneration(bgCtx, s.cache, ConfigKeyMemberBasic); err != nil {
			log.Printf("[Setting] failed to bump member config generation: %v", err)
		}
	}
	return nil
}

// ----------------- Review Settings -----------------

func (s *Service) GetReview(ctx context.Context) (ReviewBasicConfig, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	loader := func(innerCtx context.Context) (ReviewBasicConfig, error) {
		config := DefaultReviewConfig()
		if s.db == nil {
			return config, ErrUnavailable
		}
		var row model.SystemConfig
		err := s.query(innerCtx).Where("config_key = ?", ConfigKeyReviewBasic).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return config, nil
		}
		if err != nil {
			return config, err
		}
		if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
			return config, err
		}
		return config, nil
	}

	if s.cache == nil || s.cache.IsDisabled() {
		return loader(ctx)
	}

	gen, err := cache.GetSystemConfigGeneration(ctx, s.cache, ConfigKeyReviewBasic)
	if err != nil {
		return loader(ctx)
	}

	key := cache.SystemConfigKey(ConfigKeyReviewBasic, gen)
	return cache.Get(ctx, s.cache, key, cache.TTLSystemConfig, loader)
}

func (s *Service) SaveReview(ctx context.Context, config ReviewBasicConfig) error {
	if s.db == nil {
		return ErrUnavailable
	}
	if config.AutoReviewEnabled {
		return errors.New("自动好评功能尚未开放")
	}
	if config.AutoReviewDays < 1 || config.AutoReviewDays > 30 {
		return errors.New("自动评价天数必须在1到30天之间")
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := model.SystemConfig{
		ConfigKey: ConfigKeyReviewBasic,
		Value:     string(data),
	}
	if err := s.query(ctx).
		Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "config_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error; err != nil {
		return err
	}

	if s.cache != nil && !s.cache.IsDisabled() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if _, err := cache.BumpSystemConfigGeneration(bgCtx, s.cache, ConfigKeyReviewBasic); err != nil {
			log.Printf("[Setting] failed to bump review config generation: %v", err)
		}
	}
	return nil
}

// ----------------- Agreement Settings -----------------

func (s *Service) GetAgreement(ctx context.Context) (AgreementBasicConfig, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	loader := func(innerCtx context.Context) (AgreementBasicConfig, error) {
		config := DefaultAgreementConfig()
		if s.db == nil {
			return config, ErrUnavailable
		}
		var row model.SystemConfig
		err := s.query(innerCtx).Where("config_key = ?", ConfigKeyAgreementBasic).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return config, nil
		}
		if err != nil {
			return config, err
		}
		if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
			return config, err
		}
		return config, nil
	}

	if s.cache == nil || s.cache.IsDisabled() {
		return loader(ctx)
	}

	gen, err := cache.GetSystemConfigGeneration(ctx, s.cache, ConfigKeyAgreementBasic)
	if err != nil {
		return loader(ctx)
	}

	key := cache.SystemConfigKey(ConfigKeyAgreementBasic, gen)
	return cache.Get(ctx, s.cache, key, cache.TTLSystemConfig, loader)
}

func (s *Service) SaveAgreement(ctx context.Context, config AgreementBasicConfig) error {
	if s.db == nil {
		return ErrUnavailable
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := model.SystemConfig{
		ConfigKey: ConfigKeyAgreementBasic,
		Value:     string(data),
	}
	if err := s.query(ctx).
		Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "config_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error; err != nil {
		return err
	}

	if s.cache != nil && !s.cache.IsDisabled() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if _, err := cache.BumpSystemConfigGeneration(bgCtx, s.cache, ConfigKeyAgreementBasic); err != nil {
			log.Printf("[Setting] failed to bump agreement config generation: %v", err)
		}
	}
	return nil
}

func (s *Service) GetAgreementByType(ctx context.Context, agreementType string) (AgreementDetailResponse, error) {
	cfg, err := s.GetAgreement(ctx)
	if err != nil {
		return AgreementDetailResponse{}, err
	}
	normalized := strings.ToLower(strings.TrimSpace(agreementType))
	switch normalized {
	case "user", "user_agreement":
		return AgreementDetailResponse{
			Type:    "user",
			Title:   "用户协议",
			Content: cfg.UserAgreement,
		}, nil
	case "privacy", "privacy_policy":
		return AgreementDetailResponse{
			Type:    "privacy",
			Title:   "隐私政策",
			Content: cfg.PrivacyPolicy,
		}, nil
	case "after-sale", "after_sale", "aftersale", "after_sale_policy":
		return AgreementDetailResponse{
			Type:    "after-sale",
			Title:   "售后政策",
			Content: cfg.AfterSalePolicy,
		}, nil
	default:
		return AgreementDetailResponse{}, fmt.Errorf("未知协议类型: %s", agreementType)
	}
}
