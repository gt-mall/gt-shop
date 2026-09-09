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
	Categories []BootstrapCategory `json:"categories"`
} // @name BootstrapResponse

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
