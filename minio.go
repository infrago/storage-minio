package storage_minio

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/bamgoo/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type (
	minioDriver struct{}

	minioConnection struct {
		instance *storage.Instance
		setting  minioSetting
		client   *minio.Client
	}

	minioSetting struct {
		Endpoint  string
		Region    string
		Bucket    string
		AccessKey string
		SecretKey string
		UseSSL    bool
	}
)

func (d *minioDriver) Connect(instance *storage.Instance) (storage.Connection, error) {
	setting := minioSetting{Endpoint: "127.0.0.1:9000", Bucket: "default"}
	if v, ok := instance.Setting["endpoint"].(string); ok && v != "" {
		setting.Endpoint = v
	}
	if v, ok := instance.Setting["region"].(string); ok && v != "" {
		setting.Region = v
	}
	if v, ok := instance.Setting["bucket"].(string); ok && v != "" {
		setting.Bucket = v
	}
	if v, ok := instance.Setting["access"].(string); ok && v != "" {
		setting.AccessKey = v
	}
	if v, ok := instance.Setting["accesskey"].(string); ok && v != "" {
		setting.AccessKey = v
	}
	if v, ok := instance.Setting["access_key"].(string); ok && v != "" {
		setting.AccessKey = v
	}
	if v, ok := instance.Setting["secret"].(string); ok && v != "" {
		setting.SecretKey = v
	}
	if v, ok := instance.Setting["secretkey"].(string); ok && v != "" {
		setting.SecretKey = v
	}
	if v, ok := instance.Setting["secret_key"].(string); ok && v != "" {
		setting.SecretKey = v
	}
	if v, ok := instance.Setting["use_ssl"].(bool); ok {
		setting.UseSSL = v
	}
	if v, ok := instance.Setting["ssl"].(bool); ok {
		setting.UseSSL = v
	}
	if setting.Bucket == "" {
		setting.Bucket = "default"
	}
	return &minioConnection{instance: instance, setting: setting}, nil
}

func (c *minioConnection) Open() error {
	client, err := minio.New(c.setting.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.setting.AccessKey, c.setting.SecretKey, ""),
		Secure: c.setting.UseSSL,
		Region: c.setting.Region,
	})
	if err != nil {
		return err
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, c.setting.Bucket)
	if err != nil {
		return err
	}
	if !exists {
		if err := client.MakeBucket(ctx, c.setting.Bucket, minio.MakeBucketOptions{Region: c.setting.Region}); err != nil {
			return err
		}
	}
	c.client = client
	return nil
}

func (c *minioConnection) Health() storage.Health {
	if c.client == nil {
		return storage.Health{Workload: 1}
	}
	return storage.Health{Workload: 0}
}

func (c *minioConnection) Close() error {
	c.client = nil
	return nil
}

func (c *minioConnection) Upload(original string, opt storage.UploadOption) (*storage.File, error) {
	if c.client == nil {
		return nil, errors.New("minio client not ready")
	}
	stat, err := os.Stat(original)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, errors.New("directory upload not supported")
	}

	ext := path.Ext(original)
	if len(ext) > 0 {
		ext = ext[1:]
	}
	if opt.Key == "" {
		return nil, errors.New("missing upload key")
	}

	file := c.instance.NewFile(opt.Prefix, opt.Key, ext, stat.Size())
	object := objectPath(file)

	metadata := toStringMap(opt.Metadata)
	tags := toStringMap(opt.Tags)

	_, err = c.client.FPutObject(context.Background(), c.setting.Bucket, object, original, minio.PutObjectOptions{
		ContentType:  opt.Mimetype,
		UserMetadata: metadata,
		UserTags:     tags,
		Expires:      opt.Expires,
	})
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (c *minioConnection) Fetch(file *storage.File, opt storage.FetchOption) (storage.Stream, error) {
	if c.client == nil {
		return nil, errors.New("minio client not ready")
	}
	object := objectPath(file)
	getOpts := minio.GetObjectOptions{}
	if opt.Start > 0 || opt.End > 0 {
		if err := getOpts.SetRange(opt.Start, opt.End); err != nil {
			return nil, err
		}
	}
	return c.client.GetObject(context.Background(), c.setting.Bucket, object, getOpts)
}

func (c *minioConnection) Download(file *storage.File, opt storage.DownloadOption) (string, error) {
	if c.client == nil {
		return "", errors.New("minio client not ready")
	}
	if opt.Target == "" {
		return "", errors.New("invalid target")
	}
	if st, err := os.Stat(opt.Target); err == nil && !st.IsDir() {
		return opt.Target, nil
	}
	if err := os.MkdirAll(path.Dir(opt.Target), 0o755); err != nil {
		return "", err
	}
	object := objectPath(file)
	if err := c.client.FGetObject(context.Background(), c.setting.Bucket, object, opt.Target, minio.GetObjectOptions{}); err != nil {
		return "", err
	}
	return opt.Target, nil
}

func (c *minioConnection) Remove(file *storage.File, _ storage.RemoveOption) error {
	if c.client == nil {
		return errors.New("minio client not ready")
	}
	return c.client.RemoveObject(context.Background(), c.setting.Bucket, objectPath(file), minio.RemoveObjectOptions{})
}

func (c *minioConnection) Browse(file *storage.File, opt storage.BrowseOption) (string, error) {
	if c.client == nil {
		return "", errors.New("minio client not ready")
	}
	exp := opt.Expires
	if exp <= 0 {
		exp = time.Hour
	}
	params := url.Values{}
	for k, v := range toStringMap(opt.Params) {
		params.Set(k, v)
	}
	u, err := c.client.PresignedGetObject(context.Background(), c.setting.Bucket, objectPath(file), exp, params)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func objectPath(file *storage.File) string {
	name := file.Key()
	if file.Type() != "" {
		name = fmt.Sprintf("%s.%s", file.Key(), file.Type())
	}
	return path.Join(file.Prefix(), name)
}

func toStringMap(m map[string]interface{}) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}
