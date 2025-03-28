package storage_object

import (
	"github.com/infrago/infra"
	"github.com/infrago/storage"
)

func Driver() storage.Driver {
	return &objectDriver{}
}

func init() {
	drv := Driver()
	infra.Register("minio", drv)
}
