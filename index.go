package storage_minio

import (
	"github.com/infrago/infra"
	"github.com/infrago/storage"
)

func Driver() storage.Driver {
	return &minioDriver{}
}

func init() {
	infra.Register("minio", Driver())
}
