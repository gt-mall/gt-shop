package file

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"mall/admin_core/internal/model"

	"github.com/google/uuid"
	_ "golang.org/x/image/webp"
	"gorm.io/gorm"
)

type Service struct{ dao *DAO }

func NewService(db *gorm.DB) *Service { return &Service{dao: &DAO{db: db}} }

func isDangerousExt(ext string) bool {
	dangerous := map[string]bool{
		"exe": true, "bat": true, "cmd": true, "sh": true,
		"php": true, "html": true, "htm": true, "js": true,
		"jsp": true, "asp": true, "aspx": true, "dll": true,
		"apk": true, "com": true, "scr": true, "vbs": true,
		"wsf": true, "cgi": true, "svg": true,
	}
	return dangerous[strings.ToLower(ext)]
}

func isAllowedProofImageExt(ext string) bool {
	allowed := map[string]bool{
		"jpg": true, "jpeg": true, "png": true, "webp": true,
	}
	return allowed[strings.ToLower(ext)]
}

func isAllowedProofVideoExt(ext string) bool {
	allowed := map[string]bool{
		"mp4": true, "mov": true,
	}
	return allowed[strings.ToLower(ext)]
}

func GetUploadLimit(scene, ext string) int64 {
	switch scene {
	case model.SceneAfterSaleProof:
		if ext == "mp4" || ext == "mov" {
			return 100 << 20 // 100 MB for after-sale video
		}
		return 10 << 20 // 10 MB for after-sale image
	case model.SceneMemberAvatar:
		return 5 << 20 // 5 MB for avatar
	case model.SceneAdminMaterial:
		if ext == "mp4" || ext == "mov" || ext == "webm" {
			return 500 << 20 // 500 MB for video
		}
		if ext == "pdf" || ext == "doc" || ext == "docx" || ext == "xls" || ext == "xlsx" || ext == "ppt" || ext == "pptx" || ext == "txt" {
			return 100 << 20 // 100 MB for document
		}
		return 20 << 20 // 20 MB for image / other
	default:
		return 20 << 20
	}
}

func DetectFileTypeAndMIME(ext string, sniffBuf []byte, scene string) (string, string, error) {
	ext = strings.ToLower(ext)
	if isDangerousExt(ext) {
		return "", "", ErrFileTypeInvalid
	}

	sniffed := http.DetectContentType(sniffBuf)
	if strings.Contains(sniffed, "text/html") ||
		strings.Contains(sniffed, "application/x-dosexec") ||
		strings.Contains(sniffed, "application/javascript") {
		return "", "", ErrFileTypeInvalid
	}

	switch ext {
	// Images
	case "jpg", "jpeg":
		if sniffed != "image/jpeg" {
			return "", "", ErrFileTypeMismatch
		}
		return model.FileTypeImage, "image/jpeg", nil
	case "png":
		if sniffed != "image/png" {
			return "", "", ErrFileTypeMismatch
		}
		return model.FileTypeImage, "image/png", nil
	case "gif":
		if sniffed != "image/gif" {
			return "", "", ErrFileTypeMismatch
		}
		return model.FileTypeImage, "image/gif", nil
	case "webp":
		isWebp := sniffed == "image/webp" ||
			(len(sniffBuf) >= 12 && string(sniffBuf[0:4]) == "RIFF" && string(sniffBuf[8:12]) == "WEBP")
		if !isWebp {
			return "", "", ErrFileTypeMismatch
		}
		return model.FileTypeImage, "image/webp", nil
	case "bmp":
		if sniffed != "image/bmp" {
			return "", "", ErrFileTypeMismatch
		}
		return model.FileTypeImage, "image/bmp", nil

	// Videos
	case "mp4":
		isMP4 := strings.HasPrefix(sniffed, "video/") ||
			(len(sniffBuf) >= 8 && string(sniffBuf[4:8]) == "ftyp")
		if !isMP4 {
			return "", "", ErrFileTypeMismatch
		}
		return model.FileTypeVideo, "video/mp4", nil
	case "mov":
		isMOV := strings.HasPrefix(sniffed, "video/") ||
			(len(sniffBuf) >= 8 && (string(sniffBuf[4:8]) == "ftyp" || string(sniffBuf[4:8]) == "moov"))
		if !isMOV {
			return "", "", ErrFileTypeMismatch
		}
		return model.FileTypeVideo, "video/quicktime", nil
	case "webm":
		return model.FileTypeVideo, "video/webm", nil

	// Audios
	case "mp3":
		return model.FileTypeAudio, "audio/mpeg", nil
	case "wav":
		return model.FileTypeAudio, "audio/wav", nil
	case "aac":
		return model.FileTypeAudio, "audio/aac", nil
	case "m4a":
		return model.FileTypeAudio, "audio/mp4", nil

	// Documents
	case "pdf":
		if !strings.HasPrefix(string(sniffBuf), "%PDF-") && sniffed != "application/pdf" {
			return "", "", ErrFileTypeMismatch
		}
		return model.FileTypeDocument, "application/pdf", nil
	case "doc":
		return model.FileTypeDocument, "application/msword", nil
	case "docx":
		return model.FileTypeDocument, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", nil
	case "xls":
		return model.FileTypeDocument, "application/vnd.ms-excel", nil
	case "xlsx":
		return model.FileTypeDocument, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
	case "ppt":
		return model.FileTypeDocument, "application/vnd.ms-powerpoint", nil
	case "pptx":
		return model.FileTypeDocument, "application/vnd.openxmlformats-officedocument.presentationml.presentation", nil
	case "txt":
		return model.FileTypeDocument, "text/plain", nil

	// Archives
	case "zip":
		return model.FileTypeArchive, "application/zip", nil
	case "rar":
		return model.FileTypeArchive, "application/x-rar-compressed", nil
	case "7z":
		return model.FileTypeArchive, "application/x-7z-compressed", nil
	case "tar":
		return model.FileTypeArchive, "application/x-tar", nil
	case "gz":
		return model.FileTypeArchive, "application/gzip", nil

	default:
		return model.FileTypeOther, "application/octet-stream", nil
	}
}

func probeVideoMetadata(data []byte, ext string) (width, height int, durationMS int64) {
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return 0, 0, 0
	}
	tmpFile, err := os.CreateTemp("", "probe_*."+ext)
	if err != nil {
		return 0, 0, 0
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return 0, 0, 0
	}
	_ = tmpFile.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,duration",
		"-of", "csv=p=0:s=,",
		tmpFile.Name(),
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, 0
	}
	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) >= 2 {
		width, _ = strconv.Atoi(parts[0])
		height, _ = strconv.Atoi(parts[1])
	}
	if len(parts) >= 3 {
		if sec, err := strconv.ParseFloat(parts[2], 64); err == nil {
			durationMS = int64(sec * 1000)
		}
	}
	return width, height, durationMS
}

func (s *Service) UploadWithParams(ctx context.Context, params UploadParams) (FileResponse, error) {
	if s.dao.db == nil {
		return FileResponse{}, ErrUnavailable
	}

	name := path.Base(strings.ReplaceAll(params.Filename, "\\", "/"))
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 255 {
		return FileResponse{}, ErrName
	}
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(name), "."))
	if ext == "" || isDangerousExt(ext) {
		return FileResponse{}, ErrFileTypeInvalid
	}

	scene := strings.TrimSpace(params.Scene)
	if scene == "" {
		scene = model.SceneAdminMaterial
	}
	ownerType := strings.TrimSpace(params.OwnerType)
	if ownerType == "" {
		ownerType = model.OwnerTypeAdmin
	}
	ownerID := params.OwnerID
	visibility := params.Visibility
	if visibility == 0 {
		visibility = model.VisibilityPublic
	}
	groupID := params.GroupID
	if scene != model.SceneAdminMaterial {
		groupID = 0 // customer files do not use material groups
	} else if groupID < 0 {
		return FileResponse{}, ErrName
	}

	// Size limit check
	maxBytes := GetUploadLimit(scene, ext)
	if params.DeclaredSize > maxBytes {
		return FileResponse{}, ErrFileSizeExceeded
	}

	data, err := io.ReadAll(io.LimitReader(params.Reader, maxBytes+1))
	if err != nil {
		return FileResponse{}, err
	}
	if int64(len(data)) > maxBytes {
		return FileResponse{}, ErrFileSizeExceeded
	}
	if len(data) == 0 {
		return FileResponse{}, errors.New("上传文件不能为空")
	}

	sniffBuf := data
	if len(sniffBuf) > 512 {
		sniffBuf = sniffBuf[:512]
	}
	fileType, mimeType, err := DetectFileTypeAndMIME(ext, sniffBuf, scene)
	if err != nil {
		return FileResponse{}, err
	}

	// Scene specific restrictions
	switch scene {
	case model.SceneAfterSaleProof:
		if fileType != model.FileTypeImage && fileType != model.FileTypeVideo {
			return FileResponse{}, ErrFileTypeInvalid
		}
		if fileType == model.FileTypeImage && !isAllowedProofImageExt(ext) {
			return FileResponse{}, ErrFileTypeInvalid
		}
		if fileType == model.FileTypeVideo && !isAllowedProofVideoExt(ext) {
			return FileResponse{}, ErrFileTypeInvalid
		}
	case model.SceneMemberAvatar:
		if fileType != model.FileTypeImage {
			return FileResponse{}, ErrFileTypeInvalid
		}
	}

	var width, height int
	var durationMS int64
	if fileType == model.FileTypeImage {
		meta, format, err := image.DecodeConfig(bytes.NewReader(data))
		formats := map[string]string{"jpg": "jpeg", "jpeg": "jpeg", "png": "png", "gif": "gif", "webp": "webp", "bmp": "bmp"}
		if err != nil || meta.Width <= 0 || meta.Height <= 0 || (meta.Height > 0 && meta.Width > 25000000/meta.Height) || (formats[ext] != "" && formats[ext] != format) {
			return FileResponse{}, ErrImage
		}
		width = meta.Width
		height = meta.Height
	} else if fileType == model.FileTypeVideo {
		width, height, durationMS = probeVideoMetadata(data, ext)
	}

	config, err := s.loadConfig(ctx)
	if err != nil {
		return FileResponse{}, err
	}
	driver, err := NewStorage(config, config.DefaultStorage)
	if err != nil {
		return FileResponse{}, ErrConfig
	}

	key := time.Now().Format("2006/01") + "/" + uuid.NewString() + "." + ext
	if err := driver.Put(ctx, key, data, mimeType); err != nil {
		return FileResponse{}, ErrStorage
	}

	row := model.SystemFile{
		GroupID:      groupID,
		Storage:      config.DefaultStorage,
		Name:         name,
		OriginalName: name,
		Path:         key,
		Ext:          ext,
		MIMEType:     mimeType,
		FileType:     fileType,
		Scene:        scene,
		OwnerType:    ownerType,
		OwnerID:      ownerID,
		Visibility:   visibility,
		Size:         int64(len(data)),
		Width:        width,
		Height:       height,
		DurationMS:   durationMS,
		CoverFileID:  0,
	}
	if err := s.dao.Create(ctx, &row); err != nil {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = driver.Delete(cleanup, key)
		return FileResponse{}, err
	}
	return s.toResponse(ctx, row, config), nil
}

// Upload preserves backward-compatibility for legacy image uploads.
func (s *Service) Upload(ctx context.Context, name string, groupID int64, reader io.Reader) (FileResponse, error) {
	return s.UploadWithParams(ctx, UploadParams{
		Filename:   name,
		GroupID:    groupID,
		Scene:      model.SceneAdminMaterial,
		OwnerType:  model.OwnerTypeAdmin,
		OwnerID:    0,
		Visibility: model.VisibilityPublic,
		Reader:     reader,
	})
}

func (s *Service) toResponse(ctx context.Context, row model.SystemFile, config StorageConfig) FileResponse {
	var fileURL string
	driver, err := NewStorage(config, row.Storage)
	if err == nil {
		if row.Visibility == model.VisibilityPrivate {
			fileURL, _ = driver.PrivateURL(ctx, row.Path, row.ID, 30*time.Minute)
		} else {
			fileURL, _ = driver.URL(ctx, row.Path)
		}
	}

	origName := row.OriginalName
	if origName == "" {
		origName = row.Name
	}
	return FileResponse{
		ID:           row.ID,
		GroupID:      row.GroupID,
		Storage:      row.Storage,
		Name:         row.Name,
		OriginalName: origName,
		Path:         row.Path,
		Ext:          row.Ext,
		MIMEType:     row.MIMEType,
		FileType:     row.FileType,
		Scene:        row.Scene,
		OwnerType:    row.OwnerType,
		OwnerID:      row.OwnerID,
		Visibility:   row.Visibility,
		Size:         row.Size,
		Width:        row.Width,
		Height:       row.Height,
		DurationMS:   row.DurationMS,
		CoverFileID:  row.CoverFileID,
		URL:          fileURL,
		CreatedAt:    row.CreatedAt,
	}
}

func (s *Service) List(ctx context.Context, req ListRequest) ([]FileResponse, int64, error) {
	config, err := s.loadConfig(ctx)
	if err != nil {
		return nil, 0, err
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 25
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	rows, total, err := s.dao.List(ctx, req)
	if err != nil {
		return nil, 0, err
	}
	result := make([]FileResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, s.toResponse(ctx, row, config))
	}
	return result, total, nil
}

// Resolve returns FileResponse for public files (or masked for private files).
func (s *Service) Resolve(ctx context.Context, req IDsRequest) ([]FileResponse, error) {
	config, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.dao.Resolve(ctx, req.IDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]model.SystemFile, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	result := make([]FileResponse, 0, len(rows))
	for _, id := range req.IDs {
		if row, ok := byID[id]; ok {
			// Public resolve: if private, do not leak raw static link
			resp := s.toResponse(ctx, row, config)
			if row.Visibility == model.VisibilityPrivate {
				resp.URL = "" // Do not provide public URL without authorization
			}
			result = append(result, resp)
		}
	}
	return result, nil
}

// ResolvePrivate generates short-term signed access URLs for authorized contexts (Admin / Member).
func (s *Service) ResolvePrivate(ctx context.Context, ids []int64) ([]FileResponse, error) {
	if len(ids) == 0 {
		return []FileResponse{}, nil
	}
	config, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.dao.Resolve(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]model.SystemFile, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	result := make([]FileResponse, 0, len(ids))
	for _, id := range ids {
		if row, ok := byID[id]; ok {
			result = append(result, s.toResponse(ctx, row, config))
		}
	}
	return result, nil
}

func (s *Service) Groups(ctx context.Context) ([]GroupResponse, error) {
	if s.dao.db == nil {
		return nil, ErrUnavailable
	}
	return s.dao.Groups(ctx)
}

func (s *Service) GroupCounts(ctx context.Context) (GroupCountsResponse, error) {
	if s.dao.db == nil {
		return GroupCountsResponse{}, ErrUnavailable
	}
	return s.dao.GroupCounts(ctx)
}

func (s *Service) SaveGroup(ctx context.Context, req GroupSaveRequest) error {
	if s.dao.db == nil {
		return ErrUnavailable
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || utf8.RuneCountInString(req.Name) > 64 {
		return ErrName
	}
	return s.dao.SaveGroup(ctx, req)
}

func (s *Service) DeleteGroup(ctx context.Context, req GroupIDRequest) error {
	if s.dao.db == nil {
		return ErrUnavailable
	}
	return s.dao.DeleteGroup(ctx, req.ID)
}

func (s *Service) Rename(ctx context.Context, req RenameRequest) error {
	if s.dao.db == nil {
		return ErrUnavailable
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || utf8.RuneCountInString(req.Name) > 255 {
		return ErrName
	}
	var file model.SystemFile
	if err := s.dao.query(ctx).Select("id, ext").First(&file, req.ID).Error; err != nil {
		return err
	}
	if file.Ext != "" {
		suffix := "." + strings.ToLower(file.Ext)
		baseName := req.Name
		if ext := path.Ext(baseName); ext != "" {
			baseName = strings.TrimSuffix(baseName, ext)
		}
		baseName = strings.TrimSpace(baseName)
		if baseName == "" {
			return ErrName
		}
		req.Name = baseName + suffix
	}
	return s.dao.Rename(ctx, req)
}

func (s *Service) Move(ctx context.Context, req MoveRequest) error {
	if s.dao.db == nil {
		return ErrUnavailable
	}
	return s.dao.Move(ctx, req)
}

func (s *Service) Delete(ctx context.Context, req IDsRequest) error {
	if s.dao.db == nil {
		return ErrUnavailable
	}
	return s.dao.Delete(ctx, req.IDs)
}

// OpenLocal serves public files only. Private files return ErrFileForbidden.
func (s *Service) OpenLocal(ctx context.Context, key string) (*os.File, error) {
	if !ValidPath(key) {
		return nil, os.ErrNotExist
	}
	config, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	var row model.SystemFile
	if err := s.dao.query(ctx).Unscoped().Where("storage = ? AND path = ?", "local", key).First(&row).Error; err != nil {
		return nil, err
	}
	if row.Visibility == model.VisibilityPrivate {
		return nil, ErrFileForbidden
	}
	root, err := os.OpenRoot(ResolveStoragePath(config.Local.Path))
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.Open(key)
}

// OpenLocalPrivate serves private files verified with HMAC signature and unexpired timestamp.
func (s *Service) OpenLocalPrivate(ctx context.Context, key string, id int64, exp int64, sign string) (*os.File, *model.SystemFile, error) {
	if !ValidPath(key) {
		return nil, nil, os.ErrNotExist
	}
	if !VerifyLocalFileSignature(id, key, exp, sign) {
		return nil, nil, ErrInvalidSignature
	}
	config, err := s.loadConfig(ctx)
	if err != nil {
		return nil, nil, err
	}
	var row model.SystemFile
	if err := s.dao.query(ctx).Unscoped().Where("storage = ? AND path = ? AND id = ?", "local", key, id).First(&row).Error; err != nil {
		return nil, nil, err
	}
	root, err := os.OpenRoot(ResolveStoragePath(config.Local.Path))
	if err != nil {
		return nil, nil, err
	}
	defer root.Close()
	f, err := root.Open(key)
	if err != nil {
		return nil, nil, err
	}
	return f, &row, nil
}
