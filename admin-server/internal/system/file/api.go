package file

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/model"
	"mall/admin_core/pkg/response"
)

type API struct{ service *Service }

func NewAPI(service *Service) *API { return &API{service: service} }

func writeError(c *gin.Context, err error) {
	var mysqlErr *mysql.MySQLError
	switch {
	case errors.Is(err, ErrUnavailable), errors.Is(err, ErrImage), errors.Is(err, ErrConfig),
		errors.Is(err, ErrGroupNotFound), errors.Is(err, ErrName), errors.Is(err, ErrStorage),
		errors.Is(err, ErrFileSizeExceeded), errors.Is(err, ErrFileTypeInvalid),
		errors.Is(err, ErrFileTypeMismatch), errors.Is(err, ErrFileInUse),
		errors.Is(err, ErrFileForbidden), errors.Is(err, ErrInvalidSignature):
		response.Error(c, err.Error())
	case errors.As(err, &mysqlErr) && mysqlErr.Number == 1062:
		response.Error(c, "名称已存在")
	default:
		response.Error(c, "素材操作失败，请检查数据库和存储服务")
	}
}

func bindJSON(c *gin.Context, request any) bool {
	if err := c.ShouldBindJSON(request); err != nil {
		response.Error(c, "请求参数不正确")
		return false
	}
	return true
}

// Upload godoc
// @Summary Upload material file (image, video, document); call for each file
// @ID uploadSystemFile
// @Tags system-file
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "File resource"
// @Param group_id formData integer false "Group ID; zero means ungrouped"
// @Success 200 {object} response.Response[FileResponse]
// @Router /api/system/files/upload [post]
func (a *API) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(1<<20))
	defer func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	}()
	file, err := c.FormFile("file")
	if err != nil {
		writeError(c, ErrFileTypeInvalid)
		return
	}
	groupID, err := strconv.ParseInt(c.DefaultPostForm("group_id", "0"), 10, 64)
	if err != nil || groupID < 0 {
		response.Error(c, "分组参数不正确")
		return
	}
	stream, err := file.Open()
	if err != nil {
		writeError(c, ErrFileTypeInvalid)
		return
	}
	defer stream.Close()

	var adminID int64
	if user, ok := auth.CurrentUser(c); ok {
		adminID = user.ID
	}

	result, err := a.service.UploadWithParams(c.Request.Context(), UploadParams{
		Filename:     file.Filename,
		GroupID:      groupID,
		Scene:        model.SceneAdminMaterial,
		OwnerType:    model.OwnerTypeAdmin,
		OwnerID:      adminID,
		Visibility:   model.VisibilityPublic,
		Reader:       stream,
		DeclaredSize: file.Size,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// ServeLocal serves public files or verified private files through an os.Root boundary.
func (a *API) ServeLocal(c *gin.Context) {
	rawPath := strings.TrimPrefix(c.Param("path"), "/")

	// Case 1: Private local file request with signature
	if strings.HasPrefix(rawPath, "private/") {
		key := strings.TrimPrefix(rawPath, "private/")
		fid, _ := strconv.ParseInt(c.Query("fid"), 10, 64)
		exp, _ := strconv.ParseInt(c.Query("exp"), 10, 64)
		sign := c.Query("sign")

		file, row, err := a.service.OpenLocalPrivate(c.Request.Context(), key, fid, exp, sign)
		if err != nil {
			c.String(http.StatusForbidden, "403 Forbidden: private file access denied")
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil || info.IsDir() {
			c.Status(http.StatusNotFound)
			return
		}

		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Content-Disposition", "inline")
		c.Header("Cache-Control", "private, no-cache")
		if row.MIMEType != "" {
			c.Header("Content-Type", row.MIMEType)
		}
		http.ServeContent(c.Writer, c.Request, key, info.ModTime(), file)
		return
	}

	// Case 2: Public local file request
	file, err := a.service.OpenLocal(c.Request.Context(), rawPath)
	if err != nil {
		if errors.Is(err, ErrFileForbidden) {
			c.String(http.StatusForbidden, "403 Forbidden: private file requires authorization")
			return
		}
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "public, max-age=86400")
	http.ServeContent(c.Writer, c.Request, rawPath, info.ModTime(), file)
}

// List godoc
// @Summary List and search admin material files
// @ID listSystemFiles
// @Tags system-file
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ListRequest true "Request"
// @Success 200 {object} response.Response[[]FileResponse]
// @Router /api/system/files/list [post]
func (a *API) List(c *gin.Context) {
	var request ListRequest
	if !bindJSON(c, &request) {
		return
	}
	result, total, err := a.service.List(c.Request.Context(), request)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, int(total))
}

// Resolve godoc
// @Summary Resolve business file IDs into current URLs (including archived images)
// @ID resolveSystemFiles
// @Tags system-file
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body IDsRequest true "Request"
// @Success 200 {object} response.Response[[]FileResponse]
// @Router /api/system/files/resolve [post]
func (a *API) Resolve(c *gin.Context) {
	var request IDsRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := a.service.Resolve(c.Request.Context(), request)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// Rename godoc
// @Summary Rename a file display name
// @ID renameSystemFile
// @Tags system-file
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RenameRequest true "Request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/files/rename [post]
func (a *API) Rename(c *gin.Context) {
	var request RenameRequest
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.Rename(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// Move godoc
// @Summary Move files to a group
// @ID moveSystemFiles
// @Tags system-file
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MoveRequest true "Request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/files/move [post]
func (a *API) Move(c *gin.Context) {
	var request MoveRequest
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.Move(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// Delete godoc
// @Summary Archive files without breaking existing business file IDs
// @ID deleteSystemFiles
// @Tags system-file
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body IDsRequest true "Request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/files/delete [post]
func (a *API) Delete(c *gin.Context) {
	var request IDsRequest
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.Delete(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// Groups godoc
// @Summary List material groups and counts
// @ID listSystemFileGroups
// @Tags system-file
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[[]GroupResponse]
// @Router /api/system/files/groups [get]
func (a *API) Groups(c *gin.Context) {
	result, err := a.service.Groups(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// GroupCounts godoc
// @Summary Get material counts for all, ungrouped, and each group
// @ID getSystemFileGroupCounts
// @Tags system-file
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[GroupCountsResponse]
// @Router /api/system/files/groups/counts [get]
func (a *API) GroupCounts(c *gin.Context) {
	result, err := a.service.GroupCounts(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// SaveGroup godoc
// @Summary Create or rename a material group
// @ID saveSystemFileGroup
// @Tags system-file
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body GroupSaveRequest true "Request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/files/groups/save [post]
func (a *API) SaveGroup(c *gin.Context) {
	var request GroupSaveRequest
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.SaveGroup(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// DeleteGroup godoc
// @Summary Delete a group and move its files to ungrouped
// @ID deleteSystemFileGroup
// @Tags system-file
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body GroupIDRequest true "Request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/files/groups/delete [post]
func (a *API) DeleteGroup(c *gin.Context) {
	var request GroupIDRequest
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.DeleteGroup(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// Config godoc
// @Summary Get storage settings with secrets masked
// @ID getFileStorageConfig
// @Tags system-file
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[StorageConfig]
// @Router /api/system/files/config [get]
func (a *API) Config(c *gin.Context) {
	result, err := a.service.Config(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// SaveConfig godoc
// @Summary Save storage settings; empty secrets retain existing values
// @ID saveFileStorageConfig
// @Tags system-file
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body StorageConfig true "Request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/files/config/save [post]
func (a *API) SaveConfig(c *gin.Context) {
	var request StorageConfig
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.SaveConfig(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}
