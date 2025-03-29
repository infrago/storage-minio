package storage_minio

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"

	"github.com/infrago/infra"
	"github.com/infrago/storage"
	"github.com/infrago/util"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

//-------------------- minioBase begin -------------------------

var (
	errBrowseNotSupported = errors.New("Store browse not supported.")
)

type (
	minioDriver  struct{}
	minioConnect struct {
		mutex  sync.RWMutex
		health storage.Health

		instance *storage.Instance
		setting  minioSetting

		client *minio.Client
	}
	minioSetting struct {
		Endpoint string
		Region   string
		Bucket   string

		AccessKey string
		SecretKey string

		UseSSL bool
	}
)

// 连接
func (driver *minioDriver) Connect(instance *storage.Instance) (storage.Connect, error) {
	setting := minioSetting{
		Bucket: infra.Name(), Endpoint: "127.0.0.1:9000",
	}

	if vv, ok := instance.Setting["bucket"].(string); ok {
		setting.Bucket = vv
	}
	if vv, ok := instance.Setting["endpoint"].(string); ok {
		setting.Endpoint = vv
	}
	if vv, ok := instance.Setting["region"].(string); ok {
		setting.Region = vv
	}
	if vv, ok := instance.Setting["access"].(string); ok {
		setting.AccessKey = vv
	}
	if vv, ok := instance.Setting["accesskey"].(string); ok {
		setting.AccessKey = vv
	}
	if vv, ok := instance.Setting["access_key"].(string); ok {
		setting.AccessKey = vv
	}

	if vv, ok := instance.Setting["secret"].(string); ok {
		setting.SecretKey = vv
	}
	if vv, ok := instance.Setting["secretkey"].(string); ok {
		setting.SecretKey = vv
	}
	if vv, ok := instance.Setting["secret_key"].(string); ok {
		setting.SecretKey = vv
	}

	if vv, ok := instance.Setting["use_ssl"].(bool); ok {
		setting.UseSSL = vv
	}

	return &minioConnect{
		instance: instance, setting: setting,
	}, nil

}

// 打开连接
func (this *minioConnect) Open() error {
	ctx := context.Background()
	setting := this.setting

	mc, err := minio.New(setting.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(setting.AccessKey, setting.SecretKey, ""),
		Secure: setting.UseSSL,
	})
	if err != nil {
		return err
	}

	//判断存储桶是否存在
	bucketExists, err := mc.BucketExists(ctx, setting.Bucket)
	if err != nil {
		return err
	}
	if !bucketExists {
		err = mc.MakeBucket(ctx, setting.Bucket, minio.MakeBucketOptions{Region: setting.Region})
		if err != nil {
			return err
		}
	}

	this.client = mc

	return nil
}

func (this *minioConnect) Health() storage.Health {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.health
}

// 关闭连接
func (this *minioConnect) Close() error {
	if this.client != nil {
		this.client = nil
	}
	return nil
}

func (this *minioConnect) Upload(orginal string, opt storage.UploadOption) (*storage.File, error) {
	stat, err := os.Stat(orginal)
	if err != nil {
		return nil, err
	}

	//250327不再支持目录上传
	if stat.IsDir() {
		return nil, errors.New("directory upload not supported")
	}

	ext := util.Extension(orginal)

	if opt.Key == "" {
		//如果没有指定key，使用文件的hash
		//使用hash的前4位，生成2级目录
		hash, hex := this.filehash(orginal)
		if opt.Prefix == "" {
			opt.Prefix = path.Join(hex[0:2], hex[2:4])
		} else {
			opt.Prefix = path.Join(opt.Prefix, hex[0:2], hex[2:4])
		}
		opt.Key = hash
	}

	file := this.instance.File(opt.Prefix, opt.Key, ext, stat.Size())
	if file == nil {
		return nil, errors.New("create file error")
	}

	//保存文件
	_, tfile, err := this.filepath(file)
	if err != nil { //文件路径错误
		return nil, err
	}

	metadata := map[string]string{}
	for k, v := range opt.Metadata {
		metadata[k] = fmt.Sprintf("%v", v)
	}
	tags := map[string]string{}
	for k, v := range opt.Tags {
		tags[k] = fmt.Sprintf("%v", v)
	}

	bucket := this.setting.Bucket

	ctx := context.Background()
	_, putErr := this.client.FPutObject(ctx, bucket, tfile, orginal, minio.PutObjectOptions{
		ContentType:  opt.Mimetype,
		UserMetadata: metadata, UserTags: tags,
		Expires: opt.Expires,
	})
	if putErr != nil {
		return nil, putErr
	}

	return file, nil
}

func (this *minioConnect) Fetch(file *storage.File, opt storage.FetchOption) (storage.Stream, error) {
	_, sFile, err := this.filepath(file)
	if err != nil {
		return nil, err
	}

	bucketName := this.setting.Bucket

	getOpts := minio.GetObjectOptions{}
	if opt.Start > 0 || opt.End > 0 {
		getOpts.SetRange(opt.Start, opt.End)
	}

	ctx := context.Background()
	return this.client.GetObject(ctx, bucketName, sFile, getOpts)
}

func (this *minioConnect) Download(file *storage.File, opt storage.DownloadOption) (string, error) {
	_, sFile, err := this.filepath(file)
	if err != nil {
		return "", err
	}

	if opt.Target == "" {
		return "", errors.New("invalid target")
	}

	_, err = os.Stat(opt.Target)
	if err == nil {
		//无错误，文件已经存在，直接返回
		return opt.Target, nil
	}

	bucketName := this.setting.Bucket
	minioName := sFile

	ctx := context.Background()
	getErr := this.client.FGetObject(ctx, bucketName, minioName, opt.Target, minio.GetObjectOptions{})
	if getErr != nil {
		return "", getErr
	}

	return opt.Target, nil
}

func (this *minioConnect) Remove(file *storage.File, opt storage.RemoveOption) error {
	_, sFile, err := this.filepath(file)
	if err != nil {
		return err
	}

	bucketName := this.setting.Bucket
	minioName := sFile

	ctx := context.Background()
	rmErr := this.client.RemoveObject(ctx, bucketName, minioName, minio.RemoveObjectOptions{})
	if rmErr != nil {
		return rmErr
	}

	return nil
}

func (this *minioConnect) Browse(file *storage.File, opt storage.BrowseOption) (string, error) {
	return "", errBrowseNotSupported
}

//-------------------- minioBase end -------------------------

// storaging 生成存储路径
func (this *minioConnect) filepath(file *storage.File) (string, string, error) {
	name := file.Key()
	if file.Type() != "" {
		name = fmt.Sprintf("%s.%s", file.Key(), file.Type())
	}

	sfile := path.Join(file.Prefix(), name)
	spath := path.Dir(sfile)

	return spath, sfile, nil
}

// 算文件的hash
func (this *minioConnect) filehash(file string) (string, string) {
	if f, e := os.Open(file); e == nil {
		defer f.Close()
		h := sha1.New()
		if _, e := io.Copy(h, f); e == nil {
			hex := fmt.Sprintf("%x", h.Sum(nil))
			hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
			return hash, hex
		}
	}
	return "", ""
}
