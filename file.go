package store_object

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"time"

	. "github.com/infrago/base"
	"github.com/infrago/store"
	"github.com/infrago/util"
)

//-------------------- objectBase begin -------------------------

var (
	errBrowseNotSupported = errors.New("Store browse not supported.")
)

type (
	objectDriver  struct{}
	objectConnect struct {
		mutex  sync.RWMutex
		health store.Health

		instance *store.Instance

		setting objectSetting
	}
	objectSetting struct {
		Storage string
	}
)

// 连接
func (driver *objectDriver) Connect(instance *store.Instance) (store.Connect, error) {
	setting := objectSetting{
		Storage: "store/storage",
	}

	if vv, ok := instance.Setting["storage"].(string); ok {
		setting.Storage = vv
	}

	return &objectConnect{
		instance: instance, setting: setting,
	}, nil

}

// 打开连接
func (this *objectConnect) Open() error {
	return nil
}

func (this *objectConnect) Health() store.Health {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.health
}

// 关闭连接
func (this *objectConnect) Close() error {
	return nil
}

func (this *objectConnect) Upload(target string, metadata Map) (store.File, store.Files, error) {
	stat, err := os.Stat(target)
	if err != nil {
		return nil, nil, err
	}

	//是目录
	if stat.IsDir() {

		dirs, err := ioutil.ReadDir(target)
		if err != nil {
			return nil, nil, err
		}

		files := store.Files{}
		for _, file := range dirs {
			if !file.IsDir() {

				source := path.Join(target, file.Name())
				hash := this.instance.Hash(source)
				if hash == "" {
					return nil, nil, errors.New("hash error")
				}

				info := this.instance.File(hash, source, file.Size())

				err := this.storage(source, info)
				if err != nil {
					return nil, nil, err
				}

				files = append(files, info)
			}
		}

		return nil, files, nil

	} else {

		hash := this.instance.Hash(target)
		if hash == "" {
			return nil, nil, errors.New("hash error")
		}

		file := this.instance.File(hash, target, stat.Size())

		err := this.storage(target, file)
		if err != nil {
			return nil, nil, err
		}

		return file, nil, nil
	}
}

func (this *objectConnect) Download(file store.File) (string, error) {
	///直接返回本地文件存储
	_, sFile, err := this.storaging(file)
	if err != nil {
		return "", err
	}
	return sFile, nil
}

func (this *objectConnect) Remove(file store.File) error {
	_, sFile, err := this.storaging(file)
	if err != nil {
		return err
	}

	return os.Remove(sFile)
}

func (this *objectConnect) Browse(file store.File, query Map, expirs time.Duration) (string, error) {
	return "", errBrowseNotSupported
}

//-------------------- objectBase end -------------------------

func (this *objectConnect) storage(source string, coding store.File) error {
	_, sFile, err := this.storaging(coding)
	if err != nil {
		return err
	}

	//如果文件已经存在，直接返回
	if _, err := os.Stat(sFile); err == nil {
		return nil
	}

	//打开原始文件
	fff, err := os.Open(source)
	if err != nil {
		return err
	}
	defer fff.Close()

	//创建文件
	save, err := os.OpenFile(sFile, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		return err
	}
	defer save.Close()

	//复制文件
	_, err = io.Copy(save, fff)
	if err != nil {
		return err
	}

	return nil
}

func (this *objectConnect) storaging(file store.File) (string, string, error) {
	//使用hash的hex hash 的前4位，生成2级目录
	//共256*256个目录
	hash := util.Sha256(file.Hash())
	hashPath := path.Join(hash[0:2], hash[2:4])

	full := file.Hash()
	if file.Type() != "" {
		full = fmt.Sprintf("%s.%s", file.Hash(), file.Type())
	}

	spath := path.Join(this.setting.Storage, hashPath)
	sfile := path.Join(spath, full)

	// //创建目录
	err := os.MkdirAll(spath, 0777)
	if err != nil {
		return "", "", errors.New("生成目录失败")
	}

	return spath, sfile, nil
}
