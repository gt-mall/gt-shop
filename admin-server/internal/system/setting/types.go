package setting

// ShopBasicConfig 商城基础设置
type ShopBasicConfig struct {
	Name                  string `json:"name"`                     // 商城名称
	Logo                  string `json:"logo"`                     // 商城 Logo 地址
	LogoID                int64  `json:"logo_id"`                  // 商城 Logo 图片 ID
	Favicon               string `json:"favicon"`                  // 浏览器图标地址
	FaviconID             int64  `json:"favicon_id"`               // 浏览器图标图片 ID
	ServiceName           string `json:"service_name"`             // 客服名称
	ServicePhone          string `json:"service_phone"`            // 客服电话
	ServiceWechat         string `json:"service_wechat"`           // 客服微信
	ServiceWechatQRCode   string `json:"service_wechat_qrcode"`    // 客服微信二维码地址
	ServiceWechatQRCodeID int64  `json:"service_wechat_qrcode_id"` // 客服微信二维码图片 ID
	Description           string `json:"description"`              // 商城简介
} // @name ShopBasicConfig

// SiteBasicConfig 站点基础设置
type SiteBasicConfig struct {
	Title        string `json:"title"`         // 网站标题
	Keywords     string `json:"keywords"`      // 网站关键词
	Description  string `json:"description"`   // 网站描述
	ICP          string `json:"icp"`           // ICP备案号
	PoliceRecord string `json:"police_record"` // 公安联网备案号
	Copyright    string `json:"copyright"`     // 版权信息
} // @name SiteBasicConfig

// MpConfig 小程序设置
type MpConfig struct {
	Name             string `json:"name"`               // 小程序名称
	OriginalID       string `json:"original_id"`        // 原始ID
	QRCode           string `json:"qrcode"`             // 小程序码图片地址
	QRCodeID         int64  `json:"qrcode_id"`          // 小程序码图片 ID
	AppID            string `json:"app_id"`             // AppID
	AppSecret        string `json:"app_secret"`         // AppSecret
	RequestDomain    string `json:"request_domain"`     // request合法域名
	SocketDomain     string `json:"socket_domain"`      // socket合法域名
	UploadDomain     string `json:"upload_domain"`      // uploadFile合法域名
	DownloadDomain   string `json:"download_domain"`    // downloadFile合法域名
	UDPDomain        string `json:"udp_domain"`         // udp合法域名
	TCPDomain        string `json:"tcp_domain"`         // tcp合法域名
	BusinessDomain   string `json:"business_domain"`    // 业务域名
	UploadPrivateKey string `json:"upload_private_key"` // 代码上传密钥
} // @name MpConfig

// PublicMpConfig 小程序公开基础信息（非敏感字段）
type PublicMpConfig struct {
	Name       string `json:"name"`        // 小程序名称
	OriginalID string `json:"original_id"` // 原始ID
	QRCode     string `json:"qrcode"`      // 小程序码图片地址
} // @name PublicMpConfig

// HotKeywordItem 热门搜索词配置项
type HotKeywordItem struct {
	Keyword string `json:"keyword"` // 搜索关键词 (最多50字符)
	Sort    int    `json:"sort"`    // 排序权重 (数字越小越靠前)
	Enabled bool   `json:"enabled"` // 是否启用
} // @name HotKeywordItem

// SearchBasicConfig 搜索配置
type SearchBasicConfig struct {
	HotKeywords []HotKeywordItem `json:"hot_keywords"` // 热门搜索关键词列表 (最多20条)
} // @name SearchBasicConfig

// MiniappSearchConfigResponse 小程序公开搜索配置
type MiniappSearchConfigResponse struct {
	HotKeywords []string `json:"hot_keywords"` // 启用并排序后的热门关键词列表
} // @name MiniappSearchConfigResponse

// TradeBasicConfig 交易基础设置
type TradeBasicConfig struct {
	OrderCloseMinutes    int  `json:"order_close_minutes"`    // 待付款订单自动关闭时间（分钟：1-1440）
	AutoReceiveDays      int  `json:"auto_receive_days"`      // 发货后自动确认收货天数（天：1-30）
	AfterSaleDays        int  `json:"after_sale_days"`        // 订单完成后允许申请售后天数（天：0-90）
	AllowCancelUnshipped bool `json:"allow_cancel_unshipped"` // 待发货订单是否允许用户主动取消
	AutoRefundOnCancel   bool `json:"auto_refund_on_cancel"`   // 取消待发货订单时是否自动退款
} // @name TradeBasicConfig

// PaymentBasicConfig 支付方式设置
type PaymentBasicConfig struct {
	WechatEnabled bool `json:"wechat_enabled"` // 是否开启微信支付
} // @name PaymentBasicConfig

// LogisticsBasicConfig 配送基础设置
type LogisticsBasicConfig struct {
	SenderName    string `json:"sender_name"`    // 默认发货人姓名
	SenderPhone   string `json:"sender_phone"`   // 默认发货人电话
	SenderAddress string `json:"sender_address"` // 默认发货详细地址
} // @name LogisticsBasicConfig

// MemberBasicConfig 用户与注册登录设置
type MemberBasicConfig struct {
	RegisterEnabled      bool `json:"register_enabled"`       // 是否允许新用户注册
	PasswordLoginEnabled bool `json:"password_login_enabled"` // 是否开启账号密码登录
	WechatLoginEnabled   bool `json:"wechat_login_enabled"`   // 是否开启微信小程序快捷登录
} // @name MemberBasicConfig

// ReviewBasicConfig 商品评价设置
type ReviewBasicConfig struct {
	Enabled           bool `json:"enabled"`             // 是否开启商品评价
	NeedAudit         bool `json:"need_audit"`          // 评价是否需要审核
	DefaultAnonymous  bool `json:"default_anonymous"`   // 评价是否默认匿名
	AutoReviewEnabled bool `json:"auto_review_enabled"` // 是否开启超时自动好评
	AutoReviewDays    int  `json:"auto_review_days"`    // 自动好评天数（天：1-30）
} // @name ReviewBasicConfig

// AgreementBasicConfig 政策协议设置
type AgreementBasicConfig struct {
	UserAgreement   string `json:"user_agreement"`   // 用户服务协议富文本
	PrivacyPolicy   string `json:"privacy_policy"`   // 隐私政策富文本
	AfterSalePolicy string `json:"after_sale_policy"` // 售后服务政策富文本
} // @name AgreementBasicConfig

// AgreementDetailResponse 协议详情公开响应
type AgreementDetailResponse struct {
	Type    string `json:"type"`    // 协议类型：user, privacy, after-sale
	Title   string `json:"title"`   // 协议标题
	Content string `json:"content"` // 协议内容
} // @name AgreementDetailResponse
