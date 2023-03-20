package store_object

import (
	"github.com/infrago/store"
)

func Driver() store.Driver {
	return &objectDriver{}
}

func init() {
	drv := Driver()
	store.Register("object", drv)
	store.Register("s3", drv)
	store.Register("minio", drv)
	store.Register("oss", drv)
	store.Register("cos", drv)
}
