package region

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	commonregion "mall/admin_core/internal/region"
	"mall/admin_core/pkg/response"
)

type API struct {
	service *commonregion.Service
}

func NewAPI(service *commonregion.Service) *API {
	return &API{service: service}
}

// Regions godoc
// @Summary 获取全国省市区三级行政区划树
// @ID miniappRegions
// @Tags common
// @Produce json
// @Header 200 {string} Cache-Control "public, max-age=86400"
// @Header 200 {string} ETag "SHA256 ETag"
// @Success 200 {object} response.Response[MiniappRegionsData]
// @Success 304 "Not Modified"
// @Router /regions [get]
func (a *API) Regions(c *gin.Context) {
	if a == nil || a.service == nil {
		response.Error(c, "地区服务不可用")
		return
	}

	etag := a.service.GetETag()
	c.Header("Cache-Control", "public, max-age=86400")
	if etag != "" {
		c.Header("ETag", etag)
	}

	ifNoneMatch := c.GetHeader("If-None-Match")
	if ifNoneMatch != "" && matchETag(ifNoneMatch, etag) {
		c.Status(http.StatusNotModified)
		return
	}

	data := a.service.GetData(c.Request.Context())
	response.Data(c, MiniappRegionsData{
		Version: data.Version,
		Regions: data.Regions,
	}, 0)
}

func matchETag(clientETag, serverETag string) bool {
	clientETag = strings.TrimSpace(clientETag)
	serverETag = strings.TrimSpace(serverETag)
	if clientETag == "" || serverETag == "" {
		return false
	}
	if clientETag == serverETag {
		return true
	}
	cClean := strings.TrimPrefix(clientETag, "W/")
	sClean := strings.TrimPrefix(serverETag, "W/")
	return strings.Trim(cClean, `"`) == strings.Trim(sClean, `"`)
}
