package engine

import (
	"errors"
	"os"

	"bitcask-go/record"
	"bitcask-go/storage"
)

// 索引位置：一个 key 在磁盘上的具体位置，由 storage.Write 返回
type LogPos struct {
	FileId uint32
	Offset int64
}

// 哨兵错误：key 不存在时 Get 返回它
var ErrKeyNotFound = errors.New("key not found")

// Engine 是引擎主体：内存索引 + 数据文件管理
type Engine struct {
	store *storage.Storage
	index map[string]LogPos
}

func New(dataDir string) (*Engine, error) {
	// 思路：引擎 = 底层存储 + 空索引，组装好即可。
	// 关键不变量：数据目录不存在时也要能正常打开（想想该由谁兜底建目录）；
	//           所有错误原样传回，不能吞。
	// 自查：全新目录能建出来；已有目录能复用；目录路径非法时报错。

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	st, err := storage.NewStorage(dataDir)
	if err != nil {
		return nil, err
	}

	i := make(map[string]LogPos, 0)

	e := &Engine{
		store: st,
		index: i,
	}

	if err := e.loadIndex(); err != nil {
		return nil, err
	}

	return e, nil
}

// loadIndex 打开已有数据目录时重建索引。
// 思路：按文件 id 从小到大、文件内按 offset 从 0 开始循环读，
//
//	把每个 key 的最新位置写入索引；墓碑（value 为空）跳过；
//	读到损坏记录（CRC 失败）停止该文件，继续下一个。
//
// 关键不变量：索引必须指到磁盘上该 key 的最新一条有效记录；
//
//	正常扫完（EOF 在文件尾）与提前遇坏记录的处理不能混淆。
//
// 自查：写完重开数据齐全；截断文件尾部重开不报错且只丢坏记录。
func (e *Engine) loadIndex() error {
	offset := 0
	for _, fileId := range e.store.FileIds {
		rec, err := e.store.Read(fileId, int64(offset))
		if err != nil || len(rec.Value) == 0 {
			continue
		}

		e.index[string(rec.Key)] = LogPos{FileId: fileId, Offset: int64(offset)}
		offset = rec.Size()
	}
	return nil
}

func (e *Engine) Put(key, value []byte) error {
	// 思路：先落盘，拿到这条记录的位置，再告诉索引"这个 key 的最新位置在哪"。
	// 关键不变量：索引里的位置必须和磁盘上该 key 最新记录一致；
	//           顺序错了（先改索引后落盘），崩溃后重启会读到旧值。
	// 自查：同 key 写两次，Get 返回第二次的值。

	fileId, offset, err := e.store.Write(&record.LogRecord{
		Key:   key,
		Value: value,
	})

	if err != nil {
		return err
	}

	e.index[string(key)] = LogPos{
		FileId: fileId,
		Offset: offset,
	}

	return nil
}

func (e *Engine) Get(key []byte) ([]byte, error) {
	// 思路：先找位置，再读盘。
	// 关键不变量：索引里没有、或读回来是墓碑记录，两种情况都必须
	//           返回 ErrKeyNotFound，否则删除过的 key 会读到旧值。
	// 自查：没写过的 key 报错；Put 过的能读到原值；Delete 后读不到旧值。

	logPos, ok := e.index[string(key)]
	if !ok {
		return nil, ErrKeyNotFound
	}

	rec, err := e.store.Read(logPos.FileId, logPos.Offset)
	if err != nil {
		return nil, err
	}
	if len(rec.Value) == 0 {
		return nil, ErrKeyNotFound
	}

	return rec.Value, nil
}

func (e *Engine) Delete(key []byte) error {
	// 思路：删除不是抹掉旧数据，而是追加一条标记记录。
	//       想想为什么必须这样做——这直接关系到任务 4 的崩溃恢复。
	// 关键不变量：写盘失败时绝不能留下"索引删了但磁盘还在"的半删状态。
	// 自查：删除后 Get 报 ErrKeyNotFound；重启前这些记录也要能恢复出正确状态。

	return e.Put(key, nil)
}

func (e *Engine) ListKeys() [][]byte {
	// 思路：索引里有什么就返回什么。
	// 自查：和 Put 过的 key 一一对应；注意返回类型是 [][]byte。
	res := make([][]byte, 0)
	for i := range e.index {
		key := []byte(i)
		val, err := e.Get(key)
		if err != nil || len(val) == 0 {
			continue
		}
		res = append(res, key)
	}
	return res
}

func (e *Engine) Close() error {
	// 思路：资源归属在底层，这里只做透传，不重复造轮子。
	// 自查：Close 后文件句柄真正释放（Windows 下能删掉数据目录验证）。
	return e.store.Close()
}
