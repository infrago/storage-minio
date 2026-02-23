module github.com/bamgoo/storage-minio

go 1.25.3

require (
	github.com/bamgoo/bamgoo v0.0.0
	github.com/bamgoo/storage v0.0.0
	github.com/minio/minio-go/v7 v7.0.76
)

replace github.com/bamgoo/bamgoo => ../bamgoo
replace github.com/bamgoo/storage => ../storage
