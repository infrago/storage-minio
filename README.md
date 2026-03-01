# storage-minio

`storage-minio` 是 `github.com/infrago/storage` 的**minio 驱动**。

## 包定位

- 类型：驱动
- 作用：把 `storage` 模块的统一接口落到 `minio` 后端实现

## 快速接入

```go
import (
    _ "github.com/infrago/storage"
    _ "github.com/infrago/storage-minio"
)
```

```toml
[storage]
driver = "minio"
```

## `setting` 专用配置项

配置位置：`[storage].setting`

- `endpoint`
- `region`
- `bucket`
- `access`
- `accesskey`
- `access_key`
- `secret`
- `secretkey`
- `secret_key`
- `use_ssl`
- `ssl`

## 说明

- `setting` 仅对当前驱动生效，不同驱动键名可能不同
- 连接失败时优先核对 `setting` 中 host/port/认证/超时等参数
