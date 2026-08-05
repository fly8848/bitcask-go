# bitcask-go

Bitcask 风格键值存储引擎，纯 Go、仅标准库、无第三方依赖。

## 核心思想

**追加写 + 内存索引**。所有写操作只往文件末尾追加日志记录（记录含 CRC 校验，value 长度 0 表示删除标记），内存维护 `key -> 磁盘位置(fileId, offset)` 的映射，因此读 O(1)、写顺序 IO、删除不搬数据。代价是磁盘有无效数据（被覆盖的旧记录、墓碑），由合并压缩阶段回收。

## 包结构

| 包 | 职责 |
|----|------|
| `record` | 日志记录编解码：磁盘格式 `keyLen(4B) \| valueLen(4B) \| crc(4B) \| key \| value` |
| `storage` | 数据文件与文件管理：按 id 命名、追加写、按需读、满阈值滚动 |
| `engine` | 对外 API：内存索引 + Put/Get/Delete/ListKeys/Close 主流程 |
| `merge` | 合并压缩：收集有效记录、重写新文件、清理旧文件 |

## 快速开始

```go
e, err := engine.New("./data")
if err != nil {
    log.Fatal(err)
}
defer e.Close()

e.Put([]byte("hello"), []byte("world"))
val, err := e.Get([]byte("hello")) // val == "world"
```

## 构建与测试

```sh
go build ./...
go test ./...
go vet ./...
```

## 学习文档

本项目同时是学习材料：`docs/README.md` 定义阅读顺序（论文与参考实现摘录、开发任务拆分与进度）。
