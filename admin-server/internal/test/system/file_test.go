package system_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/crc64"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func fileTestService(t *testing.T) (*systemfile.Service, *gorm.DB, systemfile.StorageConfig) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&model.SystemFile{}, &model.SystemFileGroup{}, &model.SystemConfig{}); err != nil {
		t.Fatal(err)
	}
	service := systemfile.NewService(db)
	config := systemfile.DefaultConfig()
	config.Local.Path = t.TempDir()
	if err := service.SaveConfig(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	return service, db, config
}
func fileTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.Set(1, 1, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestImageLibraryLifecycleAndStableIDs(t *testing.T) {
	ctx := context.Background()
	s, db, config := fileTestService(t)
	if err := s.SaveGroup(ctx, systemfile.GroupSaveRequest{Name: "商品图片"}); err != nil {
		t.Fatal(err)
	}
	groups, err := s.Groups(ctx)
	if err != nil || len(groups) != 1 {
		t.Fatalf("groups=%v err=%v", groups, err)
	}
	data := fileTestPNG(t)
	row, err := s.Upload(ctx, "商品.png", groups[0].ID, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if row.ID <= 0 || row.Name != "商品.png" || !systemfile.ValidPath(row.Path) || row.Width != 3 || row.Height != 2 || row.Size != int64(len(data)) {
		t.Fatalf("unexpected upload: %+v", row)
	}
	stored, err := os.ReadFile(filepath.Join(config.Local.Path, row.Path))
	if err != nil || !bytes.Equal(stored, data) {
		t.Fatal("uploaded content changed")
	}
	if err := s.Rename(ctx, systemfile.RenameRequest{ID: row.ID, Name: "夏季商品"}); err != nil {
		t.Fatal(err)
	}
	rows, total, err := s.List(ctx, systemfile.ListRequest{Keyword: "夏季"})
	if err != nil || total != 1 || rows[0].ID != row.ID || rows[0].Path != row.Path || rows[0].Name != "夏季商品.png" {
		t.Fatalf("rename/search changed file identity or failed to preserve extension: %+v", rows[0])
	}
	if err := s.Rename(ctx, systemfile.RenameRequest{ID: row.ID, Name: "夏季热销.jpg"}); err != nil {
		t.Fatal(err)
	}
	rows, _, _ = s.List(ctx, systemfile.ListRequest{Keyword: "夏季热销"})
	if len(rows) != 1 || rows[0].Name != "夏季热销.png" {
		t.Fatalf("rename allowed tampering with extension: %+v", rows)
	}
	if err := s.Move(ctx, systemfile.MoveRequest{IDs: []int64{row.ID}, GroupID: 0}); err != nil {
		t.Fatal(err)
	}
	counts, err := s.GroupCounts(ctx)
	if err != nil || counts.Total != 1 || counts.Ungrouped != 1 || counts.TotalStorageSize != int64(len(data)) || counts.UngroupedStorageSize != int64(len(data)) || len(counts.Groups) != 1 || counts.Groups[0].Count != 0 || counts.Groups[0].StorageSize != 0 {
		t.Fatalf("unexpected counts after moving to ungrouped: %+v", counts)
	}
	if err := s.Move(ctx, systemfile.MoveRequest{IDs: []int64{row.ID}, GroupID: groups[0].ID}); err != nil {
		t.Fatal(err)
	}
	counts, err = s.GroupCounts(ctx)
	if err != nil || counts.Total != 1 || counts.Ungrouped != 0 || counts.TotalStorageSize != int64(len(data)) || counts.UngroupedStorageSize != 0 || len(counts.Groups) != 1 || counts.Groups[0].Count != 1 || counts.Groups[0].StorageSize != int64(len(data)) {
		t.Fatalf("unexpected counts after moving to group: %+v", counts)
	}
	if err := s.DeleteGroup(ctx, systemfile.GroupIDRequest{ID: groups[0].ID}); err != nil {
		t.Fatal(err)
	}
	groups, _ = s.Groups(ctx)
	if len(groups) != 0 {
		t.Fatal("group was not deleted")
	}
	rows, _, _ = s.List(ctx, systemfile.ListRequest{})
	if rows[0].GroupID != 0 {
		t.Fatal("group deletion did not ungroup image")
	}
	config.Local.Domain = "https://images.example.test/media"
	if err := s.SaveConfig(ctx, config); err != nil {
		t.Fatal(err)
	}
	config.DefaultStorage = "s3"
	config.S3 = systemfile.ObjectConfig{Bucket: "test-bucket", Region: "us-east-1", AccessKeyID: "test-key", AccessKeySecret: "test-secret"}
	if err := s.SaveConfig(ctx, config); err != nil {
		t.Fatal(err)
	}
	rows, _, _ = s.List(ctx, systemfile.ListRequest{})
	if rows[0].Storage != "local" || rows[0].URL != config.Local.Domain+"/"+row.Path {
		t.Fatal("default storage switch changed historical image")
	}
	if err := s.Delete(ctx, systemfile.IDsRequest{IDs: []int64{row.ID}}); err != nil {
		t.Fatal(err)
	}
	_, total, _ = s.List(ctx, systemfile.ListRequest{})
	if total != 0 {
		t.Fatal("deleted image still in library")
	}
	rows, err = s.Resolve(ctx, systemfile.IDsRequest{IDs: []int64{row.ID}})
	if err != nil || len(rows) != 1 {
		t.Fatal("business file_id no longer resolves after archive")
	}
	file, _, err := s.OpenLocal(ctx, row.Path)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()
	var record model.SystemFile
	db.Unscoped().First(&record, row.ID)
	if strings.Contains(record.Path, "://") || !record.DeletedAt.Valid {
		t.Fatal("URL stored or image not archived")
	}
}

func TestImageValidationAndUploadRollback(t *testing.T) {
	ctx := context.Background()
	s, _, config := fileTestService(t)
	pngData := fileTestPNG(t)
	for _, item := range []struct {
		name string
		data []byte
	}{
		{"fake.png", []byte("<script>alert(1)</script>")},
		{"wrong.jpg", pngData},
		{"unsafe.svg", pngData},
		{"truncated.png", pngData[:30]},
		{"large.png", append(pngData, make([]byte, systemfile.MaxUploadBytes)...)},
	} {
		if _, err := s.Upload(ctx, item.name, 0, bytes.NewReader(item.data)); err == nil {
			t.Errorf("accepted %s", item.name)
		}
	}
	if _, err := s.Upload(ctx, "valid.png", 999, bytes.NewReader(pngData)); err == nil {
		t.Fatal("accepted nonexistent group")
	}
	files := 0
	filepath.WalkDir(config.Local.Path, func(_ string, e os.DirEntry, err error) error {
		if err == nil && !e.IsDir() {
			files++
		}
		return err
	})
	if files != 0 {
		t.Fatal("failed database insert leaked uploaded object")
	}
	for _, key := range []string{"../.env", "/etc/passwd", "2026/09/../../secret", "2026/09/test.svg"} {
		if _, _, err := s.OpenLocal(ctx, key); err == nil {
			t.Fatalf("accepted invalid path %s", key)
		}
	}
}

func TestStorageSecretsRemainMaskedAndPreserved(t *testing.T) {
	ctx := context.Background()
	s, _, config := fileTestService(t)
	config.S3 = systemfile.ObjectConfig{Bucket: "test-bucket", Region: "us-east-1", AccessKeyID: "test-key", AccessKeySecret: "test-secret"}
	config.DefaultStorage = "s3"
	if err := s.SaveConfig(ctx, config); err != nil {
		t.Fatal(err)
	}
	masked, err := s.Config(ctx)
	if err != nil || masked.S3.AccessKeySecret != "" || !masked.S3.SecretSet {
		t.Fatal("secret leaked or not indicated")
	}
	masked.S3.Domain = "https://cdn.example.test"
	if err := s.SaveConfig(ctx, masked); err != nil {
		t.Fatal("empty secret did not preserve prior credential")
	}
	masked.S3.Domain = "javascript:alert(1)"
	if err := s.SaveConfig(ctx, masked); err == nil {
		t.Fatal("unsafe domain accepted")
	}
}

func TestObjectStorageAdapters(t *testing.T) {
	key := "2026/09/550e8400-e29b-41d4-a716-446655440000.png"
	for _, kind := range []string{"oss", "cos", "s3"} {
		t.Run(kind, func(t *testing.T) {
			puts, deletes := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.Path, key) {
					t.Errorf("unexpected object path %s", r.URL.Path)
				}
				if r.Header.Get("Authorization") == "" {
					t.Error("upload request is unsigned")
				}
				switch r.Method {
				case http.MethodPut:
					puts++
					body, _ := io.ReadAll(r.Body)
					if !bytes.Equal(body, []byte("test-image")) {
						t.Error("object body changed")
					}
				case http.MethodDelete:
					deletes++
				default:
					t.Errorf("unexpected method %s", r.Method)
				}
				w.Header().Set("ETag", `"test"`)
				w.Header().Set("x-cos-hash-crc64ecma", strconv.FormatUint(crc64.Checksum([]byte("test-image"), crc64.MakeTable(crc64.ECMA)), 10))
				if r.Method == http.MethodDelete {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			cfg := systemfile.DefaultConfig()
			p := systemfile.ObjectConfig{Endpoint: server.URL, Region: "us-east-1", Bucket: "test-bucket", AccessKeyID: "test-key", AccessKeySecret: "test-secret", UsePathStyle: true}
			cfg.OSS = p
			cfg.COS = p
			cfg.S3 = p
			driver, err := systemfile.NewStorage(cfg, kind)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			if err := driver.Put(ctx, key, bytes.NewReader([]byte("test-image")), "image/png"); err != nil {
				t.Fatal(err)
			}
			signed, err := driver.URL(ctx, key)
			if err != nil || !strings.Contains(signed, "?") {
				t.Fatal("private image URL not signed")
			}
			if err := driver.Delete(ctx, key); err != nil {
				t.Fatal(err)
			}
			if puts != 1 || deletes != 1 {
				t.Fatalf("puts=%d deletes=%d", puts, deletes)
			}
		})
	}
}

func TestResolveStoragePath(t *testing.T) {
	// 1. 绝对路径保持原样
	abs := filepath.Clean("/custom/absolute/uploads")
	if got := systemfile.ResolveStoragePath(abs); got != abs {
		t.Fatalf("expected %s, got %s", abs, got)
	}

	// 2. 环境变量 RUN_DIR 生效
	tempRunDir := t.TempDir()
	t.Setenv("RUN_DIR", tempRunDir)
	expectedRun := filepath.Join(tempRunDir, "uploads")
	if got := systemfile.ResolveStoragePath("uploads"); got != expectedRun {
		t.Fatalf("expected %s, got %s", expectedRun, got)
	}

	// 3. 环境变量 APP_ROOT 生效
	t.Setenv("RUN_DIR", "")
	tempAppRoot := t.TempDir()
	t.Setenv("APP_ROOT", tempAppRoot)
	expectedApp := filepath.Join(tempAppRoot, "my_uploads")
	if got := systemfile.ResolveStoragePath("my_uploads"); got != expectedApp {
		t.Fatalf("expected %s, got %s", expectedApp, got)
	}
}

func fileTestMP4() []byte {
	// Minimal valid MP4 header with ftyp box
	b := []byte("\x00\x00\x00\x18ftypmp42\x00\x00\x00\x00isommp42\x00\x00\x00\x08free")
	return b
}

func TestUnifiedFileSystemAndSecurityRules(t *testing.T) {
	ctx := context.Background()
	s, db, _ := fileTestService(t)
	pngData := fileTestPNG(t)
	mp4Data := fileTestMP4()

	// 1. Legacy file migration / backfill compatibility
	legacyFile := model.SystemFile{
		Storage: "local",
		Name:    "legacy.png",
		Path:    "2026/01/00000000-0000-0000-0000-000000000001.png",
		Ext:     "png",
		Size:    int64(len(pngData)),
		Width:   3,
		Height:  2,
	}
	if err := db.Create(&legacyFile).Error; err != nil {
		t.Fatal(err)
	}
	// Verify GORM default values / backfill
	var fetchedLegacy model.SystemFile
	if err := db.First(&fetchedLegacy, legacyFile.ID).Error; err != nil {
		t.Fatal(err)
	}
	if fetchedLegacy.Scene != model.SceneAdminMaterial || fetchedLegacy.Visibility != model.VisibilityPublic {
		t.Fatalf("unexpected legacy defaults: %+v", fetchedLegacy)
	}

	// 2. Admin uploads image: scene=admin_material, owner_type=admin, visibility=public
	adminImg, err := s.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:   "admin_banner.png",
		GroupID:    0,
		Scene:      model.SceneAdminMaterial,
		OwnerType:  model.OwnerTypeAdmin,
		OwnerID:    1,
		Visibility: model.VisibilityPublic,
		Reader:     bytes.NewReader(pngData),
	})
	if err != nil {
		t.Fatalf("admin upload image failed: %v", err)
	}
	if adminImg.Scene != model.SceneAdminMaterial || adminImg.OwnerType != model.OwnerTypeAdmin || adminImg.OwnerID != 1 || adminImg.Visibility != model.VisibilityPublic || adminImg.FileType != model.FileTypeImage {
		t.Fatalf("unexpected admin image: %+v", adminImg)
	}

	// 3. Admin uploads video: file_type=video, scene=admin_material
	adminVideo, err := s.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:   "intro.mp4",
		GroupID:    0,
		Scene:      model.SceneAdminMaterial,
		OwnerType:  model.OwnerTypeAdmin,
		OwnerID:    1,
		Visibility: model.VisibilityPublic,
		Reader:     bytes.NewReader(mp4Data),
	})
	if err != nil {
		t.Fatalf("admin upload video failed: %v", err)
	}
	if adminVideo.FileType != model.FileTypeVideo || adminVideo.Scene != model.SceneAdminMaterial {
		t.Fatalf("unexpected admin video: %+v", adminVideo)
	}

	// 4. Member aftersale uploads image: scene=after_sale_proof, visibility=private, owner_id bound
	memberID := int64(888)
	memberProofImg, err := s.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:   "broken_goods.png",
		GroupID:    0,
		Scene:      model.SceneAfterSaleProof,
		OwnerType:  model.OwnerTypeMember,
		OwnerID:    memberID,
		Visibility: model.VisibilityPrivate,
		Reader:     bytes.NewReader(pngData),
	})
	if err != nil {
		t.Fatalf("member upload proof image failed: %v", err)
	}
	if memberProofImg.Scene != model.SceneAfterSaleProof || memberProofImg.OwnerType != model.OwnerTypeMember || memberProofImg.OwnerID != memberID || memberProofImg.Visibility != model.VisibilityPrivate || memberProofImg.FileType != model.FileTypeImage {
		t.Fatalf("unexpected member proof image: %+v", memberProofImg)
	}

	// 5. Member aftersale uploads video: scene=after_sale_proof, file_type=video
	memberProofVideo, err := s.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:   "unpacking.mp4",
		GroupID:    0,
		Scene:      model.SceneAfterSaleProof,
		OwnerType:  model.OwnerTypeMember,
		OwnerID:    memberID,
		Visibility: model.VisibilityPrivate,
		Reader:     bytes.NewReader(mp4Data),
	})
	if err != nil {
		t.Fatalf("member upload proof video failed: %v", err)
	}
	if memberProofVideo.FileType != model.FileTypeVideo || memberProofVideo.Scene != model.SceneAfterSaleProof || memberProofVideo.Visibility != model.VisibilityPrivate {
		t.Fatalf("unexpected member proof video: %+v", memberProofVideo)
	}

	// 6. After-sale proof forbids non-media types (e.g. document pdf or txt)
	pdfDummy := []byte("%PDF-1.4 dummy content")
	_, err = s.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:   "invoice.pdf",
		Scene:      model.SceneAfterSaleProof,
		OwnerType:  model.OwnerTypeMember,
		OwnerID:    memberID,
		Visibility: model.VisibilityPrivate,
		Reader:     bytes.NewReader(pdfDummy),
	})
	if err == nil {
		t.Fatal("expected PDF upload to be rejected for after_sale_proof")
	}

	// 7. Dangerous extension rejection
	_, err = s.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:   "exploit.php",
		Scene:      model.SceneAdminMaterial,
		OwnerType:  model.OwnerTypeAdmin,
		OwnerID:    1,
		Visibility: model.VisibilityPublic,
		Reader:     bytes.NewReader([]byte("<?php echo 1; ?>")),
	})
	if err == nil {
		t.Fatal("expected dangerous php file to be rejected")
	}

	// 8. MIME & extension conflict rejection (executable masquerading as jpg)
	fakeJPG := []byte("MZ\x90\x00\x03\x00\x00\x00\x04\x00\x00\x00\xff\xff\x00\x00")
	_, err = s.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:   "fake.jpg",
		Scene:      model.SceneAdminMaterial,
		OwnerType:  model.OwnerTypeAdmin,
		OwnerID:    1,
		Visibility: model.VisibilityPublic,
		Reader:     bytes.NewReader(fakeJPG),
	})
	if err == nil {
		t.Fatal("expected disguised exe to be rejected on MIME mismatch")
	}

	// 9. Material Library isolation: List must NOT return member after-sale files
	matList, matTotal, err := s.List(ctx, systemfile.ListRequest{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range matList {
		if row.Scene == model.SceneAfterSaleProof {
			t.Fatalf("after_sale_proof file %d found in material library!", row.ID)
		}
	}
	// Filter by file_type = video
	videoList, _, err := s.List(ctx, systemfile.ListRequest{FileType: "video"})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range videoList {
		if v.FileType != model.FileTypeVideo {
			t.Fatalf("expected only video, got %s", v.FileType)
		}
	}

	// 10. Public vs Private resolution
	// Public resolve should NOT leak raw static links for private files
	pubResolved, err := s.Resolve(ctx, systemfile.IDsRequest{IDs: []int64{adminImg.ID, memberProofImg.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if len(pubResolved) != 2 {
		t.Fatalf("expected 2 resolved files, got %d", len(pubResolved))
	}
	if pubResolved[0].ID == adminImg.ID && pubResolved[0].URL == "" {
		t.Fatal("expected public file to have valid URL")
	}
	if pubResolved[1].ID == memberProofImg.ID && pubResolved[1].URL != "" {
		t.Fatal("expected private file to have empty URL in public resolve")
	}

	// Authorized ResolvePrivate should generate signed URLs
	privResolved, err := s.ResolvePrivate(ctx, []int64{memberProofImg.ID, memberProofVideo.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(privResolved) != 2 || privResolved[0].URL == "" || privResolved[1].URL == "" {
		t.Fatalf("expected signed URLs in ResolvePrivate: %+v", privResolved)
	}
	if !strings.Contains(privResolved[0].URL, "sign=") || !strings.Contains(privResolved[0].URL, "fid=") {
		t.Fatalf("expected HMAC sign params in private URL: %s", privResolved[0].URL)
	}

	// 11. Local file access security:
	// Public OpenLocal on private file must fail with ErrFileForbidden
	_, _, err = s.OpenLocal(ctx, memberProofImg.Path)
	if !errors.Is(err, systemfile.ErrFileForbidden) {
		t.Fatalf("expected ErrFileForbidden on public open of private file, got %v", err)
	}

	// OpenLocalPrivate with valid signature succeeds
	exp := time.Now().Add(10 * time.Minute).Unix()
	validSign := systemfile.SignLocalFile(memberProofImg.ID, memberProofImg.Path, exp)
	f, row, err := s.OpenLocalPrivate(ctx, memberProofImg.Path, memberProofImg.ID, exp, validSign)
	if err != nil {
		t.Fatalf("valid OpenLocalPrivate failed: %v", err)
	}
	_ = f.Close()
	if row.ID != memberProofImg.ID {
		t.Fatalf("mismatched file row: %+v", row)
	}

	// OpenLocalPrivate with tampered sign or expired timestamp fails
	_, _, err = s.OpenLocalPrivate(ctx, memberProofImg.Path, memberProofImg.ID, exp, "fake-signature")
	if !errors.Is(err, systemfile.ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
	expiredExp := time.Now().Add(-10 * time.Minute).Unix()
	expiredSign := systemfile.SignLocalFile(memberProofImg.ID, memberProofImg.Path, expiredExp)
	_, _, err = s.OpenLocalPrivate(ctx, memberProofImg.Path, memberProofImg.ID, expiredExp, expiredSign)
	if !errors.Is(err, systemfile.ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature for expired, got %v", err)
	}

	// 12. Deletion safety: Cannot delete non-admin_material files via material Delete
	err = s.Delete(ctx, systemfile.IDsRequest{IDs: []int64{memberProofVideo.ID}})
	if err == nil {
		t.Fatalf("expected error when deleting non-admin_material file, got nil")
	}

	// Cannot delete admin_material files referenced by after-sale
	matReferenced, err := s.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:     "mat_proof.png",
		Scene:        model.SceneAdminMaterial,
		OwnerType:    model.OwnerTypeAdmin,
		OwnerID:      1,
		Visibility:   model.VisibilityPublic,
		Reader:       bytes.NewReader(pngData),
		DeclaredSize: int64(len(pngData)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AfterSale{}); err != nil {
		t.Fatal(err)
	}
	proofJSON := fmt.Sprintf("[%d]", matReferenced.ID)
	asRecord := model.AfterSale{
		AfterSaleNo:  "AS_TEST_001",
		RequestID:    "req_test_001",
		OrderID:      1,
		MemberID:     memberID,
		Type:         1,
		Status:       10,
		ProofFileIDs: &proofJSON,
		AppliedAt:    time.Now(),
	}
	if err := db.Create(&asRecord).Error; err != nil {
		t.Fatal(err)
	}

	err = s.Delete(ctx, systemfile.IDsRequest{IDs: []int64{matReferenced.ID}})
	if !errors.Is(err, systemfile.ErrFileInUse) {
		t.Fatalf("expected ErrFileInUse when deleting referenced material, got %v", err)
	}

	_ = matTotal
}

func TestFileSecurityRules_SceneAllowlistAndMutations(t *testing.T) {
	ctx := context.Background()
	s, db, _ := fileTestService(t)
	pngData := fileTestPNG(t)

	// 1. Allowlists reject dangerous extensions: .xhtml, .html, .svg, .js
	badExts := []string{"test.xhtml", "test.html", "test.svg", "test.js", "test.xml", "test.php"}
	for _, ext := range badExts {
		_, err := s.UploadWithParams(ctx, systemfile.UploadParams{
			Filename:     ext,
			Scene:        model.SceneAdminMaterial,
			OwnerType:    model.OwnerTypeAdmin,
			OwnerID:      1,
			Visibility:   model.VisibilityPublic,
			Reader:       bytes.NewReader(pngData),
			DeclaredSize: int64(len(pngData)),
		})
		if err == nil {
			t.Fatalf("expected upload of %s to be rejected in admin_material", ext)
		}

		_, err = s.UploadWithParams(ctx, systemfile.UploadParams{
			Filename:     ext,
			Scene:        model.SceneAfterSaleProof,
			OwnerType:    model.OwnerTypeMember,
			OwnerID:      100,
			Visibility:   model.VisibilityPrivate,
			Reader:       bytes.NewReader(pngData),
			DeclaredSize: int64(len(pngData)),
		})
		if err == nil {
			t.Fatalf("expected upload of %s to be rejected in after_sale_proof", ext)
		}
	}

	// 2. Upload valid after_sale_proof file and admin_material file
	proofFile, err := s.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:     "proof.png",
		Scene:        model.SceneAfterSaleProof,
		OwnerType:    model.OwnerTypeMember,
		OwnerID:      100,
		Visibility:   model.VisibilityPrivate,
		Reader:       bytes.NewReader(pngData),
		DeclaredSize: int64(len(pngData)),
	})
	if err != nil {
		t.Fatalf("upload valid proof failed: %v", err)
	}

	matFile, err := s.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:     "material.png",
		Scene:        model.SceneAdminMaterial,
		OwnerType:    model.OwnerTypeAdmin,
		OwnerID:      1,
		Visibility:   model.VisibilityPublic,
		Reader:       bytes.NewReader(pngData),
		DeclaredSize: int64(len(pngData)),
	})
	if err != nil {
		t.Fatalf("upload valid material failed: %v", err)
	}

	// 3. Rename must reject non-admin_material file
	err = s.Rename(ctx, systemfile.RenameRequest{ID: proofFile.ID, Name: "renamed.png"})
	if err == nil {
		t.Fatalf("expected error when renaming non-material file, got nil")
	}
	// Rename admin_material succeeds
	err = s.Rename(ctx, systemfile.RenameRequest{ID: matFile.ID, Name: "renamed.png"})
	if err != nil {
		t.Fatalf("renaming material failed: %v", err)
	}

	// 4. Move must reject non-admin_material file
	err = s.SaveGroup(ctx, systemfile.GroupSaveRequest{Name: "测试分组"})
	if err != nil {
		t.Fatal(err)
	}
	var grp model.SystemFileGroup
	_ = db.First(&grp)

	err = s.Move(ctx, systemfile.MoveRequest{IDs: []int64{proofFile.ID}, GroupID: grp.ID})
	if err == nil {
		t.Fatalf("expected error when moving non-material file, got nil")
	}
	// Move admin_material succeeds
	err = s.Move(ctx, systemfile.MoveRequest{IDs: []int64{matFile.ID}, GroupID: grp.ID})
	if err != nil {
		t.Fatalf("moving material failed: %v", err)
	}

	// 5. Delete must reject non-admin_material file
	err = s.Delete(ctx, systemfile.IDsRequest{IDs: []int64{proofFile.ID}})
	if err == nil {
		t.Fatalf("expected error when deleting non-material file, got nil")
	}
	// Delete admin_material succeeds
	err = s.Delete(ctx, systemfile.IDsRequest{IDs: []int64{matFile.ID}})
	if err != nil {
		t.Fatalf("deleting material failed: %v", err)
	}
}

func TestFile_TXT_SniffingUTF8Boundary(t *testing.T) {
	ctx := context.Background()
	s, _, _ := fileTestService(t)

	// 1. Valid ASCII text
	asciiData := []byte("Hello, this is a plain text file for testing upload.")
	row1, err := s.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:     "ascii.txt",
		Scene:        model.SceneAdminMaterial,
		OwnerType:    model.OwnerTypeAdmin,
		OwnerID:      1,
		Visibility:   model.VisibilityPublic,
		Reader:       bytes.NewReader(asciiData),
		DeclaredSize: int64(len(asciiData)),
	})
	if err != nil {
		t.Fatalf("upload ascii txt failed: %v", err)
	}
	if row1.FileType != "document" || row1.Ext != "txt" {
		t.Fatalf("unexpected row1: %+v", row1)
	}

	// 2. Valid Chinese UTF-8 text
	chineseData := []byte("这是纯正的中文文本内容，用于测试系统文件上传接口的UTF-8编码检测。")
	row2, err := s.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:     "chinese.txt",
		Scene:        model.SceneAdminMaterial,
		OwnerType:    model.OwnerTypeAdmin,
		OwnerID:      1,
		Visibility:   model.VisibilityPublic,
		Reader:       bytes.NewReader(chineseData),
		DeclaredSize: int64(len(chineseData)),
	})
	if err != nil {
		t.Fatalf("upload chinese txt failed: %v", err)
	}
	if row2.FileType != "document" || row2.Ext != "txt" {
		t.Fatalf("unexpected row2: %+v", row2)
	}

	// 3. 512-byte sample truncated in middle of a 3-byte Chinese rune:
	// "中" in UTF-8 is 3 bytes: 0xE4 0xB8 0xAD.
	// We craft a 600-byte file: first 511 bytes are ASCII 'a', 512th byte is 0xE4 (1st byte of "中"),
	// followed by 0xB8, 0xAD and more text.
	// The 512-byte sniff buffer will end on 0xE4, which is an incomplete UTF-8 rune at the boundary!
	var truncatedBuf bytes.Buffer
	truncatedBuf.Write(bytes.Repeat([]byte("a"), 511))
	truncatedBuf.WriteString("中文测试后缀内容")
	if truncatedBuf.Len() < 512 {
		t.Fatal("sample length should be >= 512")
	}
	dataTruncated := truncatedBuf.Bytes()

	row3, err := s.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:     "truncated_boundary.txt",
		Scene:        model.SceneAdminMaterial,
		OwnerType:    model.OwnerTypeAdmin,
		OwnerID:      1,
		Visibility:   model.VisibilityPublic,
		Reader:       bytes.NewReader(dataTruncated),
		DeclaredSize: int64(len(dataTruncated)),
	})
	if err != nil {
		t.Fatalf("upload txt with truncated UTF-8 rune at 512-byte boundary should succeed, but got error: %v", err)
	}
	if row3.FileType != "document" {
		t.Fatalf("unexpected row3: %+v", row3)
	}

	// 4. Truly corrupted UTF-8 in the middle of buffer must be rejected
	badData := make([]byte, 200)
	copy(badData, []byte("Hello world, this contains invalid byte "))
	badData[10] = 0xFF // Invalid UTF-8 byte in the middle
	badData[11] = 0xFE
	_, err = s.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:     "corrupted.txt",
		Scene:        model.SceneAdminMaterial,
		OwnerType:    model.OwnerTypeAdmin,
		OwnerID:      1,
		Visibility:   model.VisibilityPublic,
		Reader:       bytes.NewReader(badData),
		DeclaredSize: int64(len(badData)),
	})
	if err == nil {
		t.Fatal("expected error for corrupted UTF-8 txt file, got nil")
	}
}
