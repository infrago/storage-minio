package storage_minio

import (
	"github.com/bamgoo/bamgoo"
	"github.com/bamgoo/storage"
)

func Driver() storage.Driver {
	return &minioDriver{}
}

func init() {
	bamgoo.Register("minio", Driver())
}
