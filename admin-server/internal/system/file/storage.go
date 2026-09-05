package file

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	cos "github.com/tencentyun/cos-go-sdk-v5"
)

var relativePathPattern = regexp.MustCompile(`^\d{4}/(0[1-9]|1[0-2])/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.(jpg|jpeg|png|gif|webp)$`)

func ValidPath(key string) bool { return relativePathPattern.MatchString(key) }

// Storage is deliberately small: adding a provider only requires this adapter.
type Storage interface {
	Put(context.Context, string, []byte, string) error
	Delete(context.Context, string) error
	URL(context.Context, string) (string, error)
}

type localStorage struct{ config LocalConfig }

func (s localStorage) Put(ctx context.Context, key string, data []byte, _ string) error {
	if !ValidPath(key) {
		return ErrImage
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
	_, writeErr := io.Copy(out, bytes.NewReader(data))
	closeErr := out.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		_ = root.Remove(key)
		return err
	}
	return nil
}
func (s localStorage) Delete(_ context.Context, key string) error {
	if !ValidPath(key) {
		return ErrImage
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
		return "", ErrImage
	}
	domain := strings.TrimRight(s.config.Domain, "/")
	if domain == "" {
		domain = "/media"
	}
	return domain + "/" + key, nil
}

type objectStorage struct {
	config ObjectConfig
	put    func(context.Context, string, []byte, string) error
	delete func(context.Context, string) error
	sign   func(context.Context, string) (string, error)
}

func (s objectStorage) Put(ctx context.Context, key string, data []byte, mime string) error {
	if !ValidPath(key) {
		return ErrImage
	}
	return s.put(ctx, key, data, mime)
}
func (s objectStorage) Delete(ctx context.Context, key string) error {
	if !ValidPath(key) {
		return ErrImage
	}
	return s.delete(ctx, key)
}
func (s objectStorage) URL(ctx context.Context, key string) (string, error) {
	if !ValidPath(key) {
		return "", ErrImage
	}
	if s.config.Domain != "" {
		return strings.TrimRight(s.config.Domain, "/") + "/" + key, nil
	}
	return s.sign(ctx, key)
}

func NewStorage(config StorageConfig, kind string) (Storage, error) {
	if kind == "local" {
		if strings.TrimSpace(config.Local.Path) == "" {
			return nil, ErrConfig
		}
		return localStorage{config.Local}, nil
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
		result.put = func(ctx context.Context, key string, data []byte, mime string) error {
			return bucket.PutObject(key, bytes.NewReader(data), oss.ContentType(mime), oss.WithContext(ctx))
		}
		result.delete = func(ctx context.Context, key string) error { return bucket.DeleteObject(key, oss.WithContext(ctx)) }
		result.sign = func(_ context.Context, key string) (string, error) { return bucket.SignURL(key, oss.HTTPGet, 3600) }
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
		result.put = func(ctx context.Context, key string, data []byte, mime string) error {
			_, err := client.Object.Put(ctx, key, bytes.NewReader(data), &cos.ObjectPutOptions{ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{ContentType: mime}})
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
	case "s3":
		if p.Region == "" {
			return nil, ErrConfig
		}
		options := s3.Options{Region: p.Region, Credentials: credentials.NewStaticCredentialsProvider(p.AccessKeyID, p.AccessKeySecret, ""), HTTPClient: httpClient, UsePathStyle: p.UsePathStyle, RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired}
		if p.Endpoint != "" {
			options.BaseEndpoint = aws.String(p.Endpoint)
		}
		client := s3.New(options)
		result.put = func(ctx context.Context, key string, data []byte, mime string) error {
			_, err := client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(p.Bucket), Key: aws.String(key), Body: bytes.NewReader(data), ContentType: aws.String(mime)})
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
	}
	return result, nil
}
