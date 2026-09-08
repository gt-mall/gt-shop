package system_test

import (
	"bytes"
	"context"
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

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
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
	file, err := s.OpenLocal(ctx, row.Path)
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
		if _, err := s.OpenLocal(ctx, key); err == nil {
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
			if err := driver.Put(ctx, key, []byte("test-image"), "image/png"); err != nil {
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
