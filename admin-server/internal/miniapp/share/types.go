package share

// ProductShareCodeResponse 商品分享码响应
type ProductShareCodeResponse struct {
	ProductID int64  `json:"product_id"`
	FileID    int64  `json:"file_id"`
	URL       string `json:"url"`
} // @name ProductShareCodeResponse

// WeChatQRCodeParams 微信小程序码生成参数
type WeChatQRCodeParams struct {
	Scene      string `json:"scene"`
	Page       string `json:"page"`
	CheckPath  bool   `json:"check_path"`
	EnvVersion string `json:"env_version,omitempty"`
	Width      int    `json:"width,omitempty"`
}

// WeChatTokenResponse 微信接口凭据响应
type WeChatTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// WeChatErrorResponse 微信通用错误响应
type WeChatErrorResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}
