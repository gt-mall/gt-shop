package file

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	cos "github.com/tencentyun/cos-go-sdk-v5"
)

var relativePathPattern = regexp.MustCompile(`^\d{4}/(0[1-9]|1[0-2])/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.[a-z0-9]{1,10}$`)

func ValidPath(key string) bool { return relativePathPattern.MatchString(key) }

var (
	signingMu     sync.RWMutex
	signingSecret = "mall_system_file_signing_secret_key"
)

// SetSigningSecret configures the HMAC secret used for local private file URL signing.
func SetSigningSecret(secret string) {
	if strings.TrimSpace(secret) == "" {
		return
	}
	signingMu.Lock()
	defer signingMu.Unlock()
	signingSecret = strings.TrimSpace(secret)
}

func getSigningSecret() string {
	signingMu.RLock()
	defer signingMu.RUnlock()
	return signingSecret
}

// SignLocalFile generates an HMAC-SHA256 signature for a local private file access.
func SignLocalFile(id int64, key string, exp int64) string {
	h := hmac.New(sha256.New, []byte(getSigningSecret()))
	_, _ = fmt.Fprintf(h, "%d:%s:%d", id, key, exp)
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyLocalFileSignature verifies whether the signature for the local private file is valid and unexpired.
func VerifyLocalFileSignature(id int64, key string, exp int64, sign string) bool {
	if exp <= 0 || time.Now().Unix() > exp {
		return false
	}
	expected := SignLocalFile(id, key, exp)
	return hmac.Equal([]byte(strings.ToLower(sign)), []byte(strings.ToLower(expected)))
}

// Storage is deliberately small: adding a provider only requires this adapter.
type Storage interface {
	Put(context.Context, string, io.Reader, string) error
	Delete(context.Context, string) error
	URL(context.Context, string) (string, error)
	PrivateURL(context.Context, string, int64, time.Duration) (string, error)
}

type localStorage struct{ config LocalConfig }

func (s localStorage) Put(ctx context.Context, key string, r io.Reader, _ string) error {
	if !ValidPath(key) {
		return ErrStorage
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(s.config.Path, 0755); err != nil {
		return err
	}
	root, err := os.OpenRoot(s.config.Path)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.MkdirAll(path.Dir(key), 0755); err != nil {
		return err
	}
	out, err := root.OpenFile(key, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, writeErr := io.Copy(out, r)
	closeErr := out.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		_ = root.Remove(key)
		return err
	}
	return nil
}

func (s localStorage) Delete(_ context.Context, key string) error {
	if !ValidPath(key) {
		return ErrStorage
	}
	root, err := os.OpenRoot(s.config.Path)
	if err != nil {
		return err
	}
	defer root.Close()
	return root.Remove(key)
}

func (s localStorage) URL(_ context.Context, key string) (string, error) {
	if !ValidPath(key) {
		return "", ErrStorage
	}
	domain := strings.TrimRight(s.config.Domain, "/")
	if domain == "" {
		domain = "/media"
	}
	return domain + "/" + key, nil
}

func (s localStorage) PrivateURL(_ context.Context, key string, id int64, duration time.Duration) (string, error) {
	if !ValidPath(key) {
		return "", ErrStorage
	}
	if duration <= 0 {
		duration = 30 * time.Minute
	}
	exp := time.Now().Add(duration).Unix()
	sig := SignLocalFile(id, key, exp)
	domain := strings.TrimRight(s.config.Domain, "/")
	if domain == "" {
		domain = "/media"
	}
	return fmt.Sprintf("%s/private/%s?fid=%d&exp=%d&sign=%s", domain, key, id, exp, sig), nil
}

type objectStorage struct {
	config       ObjectConfig
	put          func(context.Context, string, io.Reader, string) error
	delete       func(context.Context, string) error
	sign         func(context.Context, string) (string, error)
	signDuration func(context.Context, string, time.Duration) (string, error)
}

func (s objectStorage) Put(ctx context.Context, key string, r io.Reader, mime string) error {
	if !ValidPath(key) {
		return ErrStorage
	}
	return s.put(ctx, key, r, mime)
}

func (s objectStorage) Delete(ctx context.Context, key string) error {
	if !ValidPath(key) {
		return ErrStorage
	}
	return s.delete(ctx, key)
}

func (s objectStorage) URL(ctx context.Context, key string) (string, error) {
	if !ValidPath(key) {
		return "", ErrStorage
	}
	if s.config.Domain != "" {
		return strings.TrimRight(s.config.Domain, "/") + "/" + key, nil
	}
	return s.sign(ctx, key)
}

func (s objectStorage) PrivateURL(ctx context.Context, key string, _ int64, duration time.Duration) (string, error) {
	if !ValidPath(key) {
		return "", ErrStorage
	}
	if duration <= 0 {
		duration = 30 * time.Minute
	}
	if s.signDuration != nil {
		return s.signDuration(ctx, key, duration)
	}
	return s.sign(ctx, key)
}

// ResolveStoragePath 将本地存储目录解析为绝对路径。
func ResolveStoragePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		p = "uploads"
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}

	// 1. 环境变量配置
	if root := strings.TrimSpace(os.Getenv("RUN_DIR")); root != "" {
		return filepath.Clean(filepath.Join(root, p))
	}
	if root := strings.TrimSpace(os.Getenv("APP_ROOT")); root != "" {
		return filepath.Clean(filepath.Join(root, p))
	}

	// 2. 程序二进制自身所在目录
	if execPath, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(execPath); err == nil {
			execPath = resolved
		}
		if !isTempBuildPath(execPath) {
			execDir := filepath.Dir(execPath)
			return filepath.Clean(filepath.Join(execDir, p))
		}
	}

	// 3. 开发环境与回退处理
	if cwd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(cwd, "release", "run")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return filepath.Clean(filepath.Join(candidate, p))
		}
		candidate = filepath.Join(cwd, "..", "release", "run")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return filepath.Clean(filepath.Join(candidate, p))
		}
		return filepath.Clean(filepath.Join(cwd, p))
	}

	return filepath.Clean(p)
}

func isTempBuildPath(p string) bool {
	if strings.Contains(p, "go-build") {
		return true
	}
	tmp := os.TempDir()
	if tmp != "" && strings.HasPrefix(p, tmp) {
		return true
	}
	if strings.HasPrefix(p, "/private/var/folders") || strings.HasPrefix(p, "/var/folders") {
		return true
	}
	return false
}

func NewStorage(config StorageConfig, kind string) (Storage, error) {
	if kind == "local" {
		if strings.TrimSpace(config.Local.Path) == "" {
			return nil, ErrConfig
		}
		cfg := config.Local
		cfg.Path = ResolveStoragePath(cfg.Path)
		return localStorage{cfg}, nil
	}
	var p ObjectConfig
	switch kind {
	case "oss":
		p = config.OSS
	case "cos":
		p = config.COS
	case "s3":
		p = config.S3
	default:
		return nil, ErrConfig
	}
	if p.Bucket == "" || p.AccessKeyID == "" || p.AccessKeySecret == "" {
		return nil, ErrConfig
	}
	if p.Domain != "" && !validURL(p.Domain) {
		return nil, ErrConfig
	}
	if p.Endpoint != "" && !validURL(p.Endpoint) {
		return nil, ErrConfig
	}
	result := objectStorage{config: p}
	httpClient := &http.Client{Timeout: 60 * time.Second}
	switch kind {
	case "oss":
		if p.Endpoint == "" {
			return nil, ErrConfig
		}
		client, err := oss.New(p.Endpoint, p.AccessKeyID, p.AccessKeySecret, oss.HTTPClient(httpClient))
		if err != nil {
			return nil, err
		}
		bucket, err := client.Bucket(p.Bucket)
		if err != nil {
			return nil, err
		}
		result.put = func(ctx context.Context, key string, r io.Reader, mime string) error {
			return bucket.PutObject(key, r, oss.ContentType(mime), oss.WithContext(ctx))
		}
		result.delete = func(ctx context.Context, key string) error { return bucket.DeleteObject(key, oss.WithContext(ctx)) }
		result.sign = func(_ context.Context, key string) (string, error) { return bucket.SignURL(key, oss.HTTPGet, 3600) }
		result.signDuration = func(_ context.Context, key string, d time.Duration) (string, error) {
			sec := int64(d.Seconds())
			if sec <= 0 {
				sec = 1800
			}
			return bucket.SignURL(key, oss.HTTPGet, sec)
		}
	case "cos":
		endpoint := p.Endpoint
		if endpoint == "" {
			if p.Region == "" {
				return nil, ErrConfig
			}
			endpoint = "https://" + p.Bucket + ".cos." + p.Region + ".myqcloud.com"
		}
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, err
		}
		httpClient.Transport = &cos.AuthorizationTransport{SecretID: p.AccessKeyID, SecretKey: p.AccessKeySecret}
		client := cos.NewClient(&cos.BaseURL{BucketURL: u}, httpClient)
		result.put = func(ctx context.Context, key string, r io.Reader, mime string) error {
			_, err := client.Object.Put(ctx, key, r, &cos.ObjectPutOptions{ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{ContentType: mime}})
			return err
		}
		result.delete = func(ctx context.Context, key string) error { _, err := client.Object.Delete(ctx, key); return err }
		result.sign = func(ctx context.Context, key string) (string, error) {
			u, err := client.Object.GetPresignedURL(ctx, http.MethodGet, key, p.AccessKeyID, p.AccessKeySecret, time.Hour, nil)
			if err != nil {
				return "", err
			}
			return u.String(), nil
		}
		result.signDuration = func(ctx context.Context, key string, d time.Duration) (string, error) {
			if d <= 0 {
				d = 30 * time.Minute
			}
			u, err := client.Object.GetPresignedURL(ctx, http.MethodGet, key, p.AccessKeyID, p.AccessKeySecret, d, nil)
			if err != nil {
				return "", err
			}
			return u.String(), nil
		}
	case "s3":
		if p.Region == "" {
			return nil, ErrConfig
		}
		options := s3.Options{Region: p.Region, Credentials: credentials.NewStaticCredentialsProvider(p.AccessKeyID, p.AccessKeySecret, ""), HTTPClient: httpClient, UsePathStyle: p.UsePathStyle, RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired}
		if p.Endpoint != "" {
			options.BaseEndpoint = aws.String(p.Endpoint)
		}
		client := s3.New(options)
		result.put = func(ctx context.Context, key string, r io.Reader, mime string) error {
			_, err := client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(p.Bucket), Key: aws.String(key), Body: r, ContentType: aws.String(mime)})
			return err
		}
		result.delete = func(ctx context.Context, key string) error {
			_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(p.Bucket), Key: aws.String(key)})
			return err
		}
		result.sign = func(ctx context.Context, key string) (string, error) {
			signed, err := s3.NewPresignClient(client).PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(p.Bucket), Key: aws.String(key)}, s3.WithPresignExpires(time.Hour))
			if err != nil {
				return "", err
			}
			return signed.URL, nil
		}
		result.signDuration = func(ctx context.Context, key string, d time.Duration) (string, error) {
			if d <= 0 {
				d = 30 * time.Minute
			}
			signed, err := s3.NewPresignClient(client).PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(p.Bucket), Key: aws.String(key)}, s3.WithPresignExpires(d))
			if err != nil {
				return "", err
			}
			return signed.URL, nil
		}
	}
	return result, nil
}
