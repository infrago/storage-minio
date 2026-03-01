# storage-minio

`storage-minio` 是 `storage` 模块的 `minio` 驱动。

## 安装

```bash
go get github.com/infrago/storage@latest
go get github.com/infrago/storage-minio@latest
```

## 接入

```go
import (
    _ "github.com/infrago/storage"
    _ "github.com/infrago/storage-minio"
    "github.com/infrago/infra"
)

func main() {
    infra.Run()
}
```

## 配置示例

```toml
[storage]
driver = "minio"
```

## 公开 API（摘自源码）

- `func Driver() storage.Driver`
- `func (d *minioDriver) Connect(instance *storage.Instance) (storage.Connection, error)`
- `func (c *minioConnection) Open() error`
- `func (c *minioConnection) Health() storage.Health`
- `func (c *minioConnection) Close() error`
- `func (c *minioConnection) Upload(original string, opt storage.UploadOption) (*storage.File, error)`
- `func (c *minioConnection) Fetch(file *storage.File, opt storage.FetchOption) (storage.Stream, error)`
- `func (c *minioConnection) Download(file *storage.File, opt storage.DownloadOption) (string, error)`
- `func (c *minioConnection) Remove(file *storage.File, _ storage.RemoveOption) error`
- `func (c *minioConnection) Browse(file *storage.File, opt storage.BrowseOption) (string, error)`

## 排错

- driver 未生效：确认模块段 `driver` 值与驱动名一致
- 连接失败：检查 endpoint/host/port/鉴权配置
