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
	infra.Register("object", drv)
	infra.Register("s3", drv)
	infra.Register("minio", drv)
	infra.Register("oss", drv)
	infra.Register("cos", drv)
}
