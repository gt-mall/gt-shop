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
	Title       string `json:"title"`       // 网站标题
	Keywords    string `json:"keywords"`    // 网站关键词
	Description string `json:"description"` // 网站描述
	ICP         string `json:"icp"`         // ICP备案号
	Copyright   string `json:"copyright"`   // 版权信息
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
