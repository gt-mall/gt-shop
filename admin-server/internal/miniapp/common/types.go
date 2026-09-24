package common

// HelloResponse is the response payload for miniapp hello world.
type HelloResponse struct {
	Text string `json:"text" example:"Hello World"`
}

// BootstrapResponse 小程序公共启动配置
type BootstrapResponse struct {
	Shop       BootstrapShop       `json:"shop"`
	Site       BootstrapSite       `json:"site"`
	Mp         BootstrapMp         `json:"mp"`
	Payment    BootstrapPayment    `json:"payment"`
	Member     BootstrapMember     `json:"member"`
	Trade      BootstrapTrade      `json:"trade"`
	Review     BootstrapReview     `json:"review"`
	Categories []BootstrapCategory `json:"categories"`
} // @name BootstrapResponse

// BootstrapPayment 小程序支付配置
type BootstrapPayment struct {
	WechatEnabled bool `json:"wechat_enabled"`
} // @name BootstrapPayment

// BootstrapMember 小程序用户登录注册配置
type BootstrapMember struct {
	RegisterEnabled      bool `json:"register_enabled"`
	PasswordLoginEnabled bool `json:"password_login_enabled"`
	WechatLoginEnabled   bool `json:"wechat_login_enabled"`
} // @name BootstrapMember

// BootstrapTrade 小程序交易配置
type BootstrapTrade struct {
	OrderCloseMinutes    int  `json:"order_close_minutes"`
	AutoReceiveDays      int  `json:"auto_receive_days"`
	AfterSaleDays        int  `json:"after_sale_days"`
	AllowCancelUnshipped bool `json:"allow_cancel_unshipped"`
	AutoRefundOnCancel   bool `json:"auto_refund_on_cancel"`
} // @name BootstrapTrade

// BootstrapReview 小程序评价展示配置
type BootstrapReview struct {
	Enabled          bool `json:"enabled"`
	DefaultAnonymous bool `json:"default_anonymous"`
} // @name BootstrapReview

// MiniappAgreementResponse 小程序协议详情响应
type MiniappAgreementResponse struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
} // @name MiniappAgreementResponse

// BootstrapShop 商城公共基础信息
type BootstrapShop struct {
	Name                string `json:"name"`
	Logo                string `json:"logo"`
	Description         string `json:"description"`
	ServiceName         string `json:"service_name"`
	ServicePhone        string `json:"service_phone"`
	ServiceWechat       string `json:"service_wechat"`
	ServiceWechatQRCode string `json:"service_wechat_qrcode"`
} // @name BootstrapShop

// BootstrapSite 站点公共信息
type BootstrapSite struct {
	Title       string `json:"title"`
	Keywords    string `json:"keywords"`
	Description string `json:"description"`
	ICP         string `json:"icp"`
	Copyright   string `json:"copyright"`
} // @name BootstrapSite

// BootstrapMp 小程序公共信息
type BootstrapMp struct {
	Name       string `json:"name"`
	OriginalID string `json:"original_id"`
	QRCode     string `json:"qrcode"`
} // @name BootstrapMp

// BootstrapCategory 分类公共信息
type BootstrapCategory struct {
	ID       int64  `json:"id"`
	ParentID int64  `json:"parent_id"`
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
	Sort     int    `json:"sort"`
} // @name BootstrapCategory
