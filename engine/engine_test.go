package engine

import (
	"bytes"
	"errors"
	"testing"
)

// newTestEngine 开一个临时数据目录的引擎，返回引擎和关闭函数。
// 每个测试都用它，目录由 t.TempDir() 提供，测试结束自动清理。
func newTestEngine(t *testing.T) (*Engine, func()) {
	t.Helper()

	e, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	return e, func() {
		e.Close()
	}
}

// 场景：Put 之后 Get 能读回原值，key/value 与原数据逐字节一致。
func TestPutAndGet(t *testing.T) {
	e, cleanup := newTestEngine(t)
	t.Cleanup(cleanup)

	key := []byte("key")
	value := []byte("value")
	err := e.Put(key, value)
	if err != nil {
		t.Fatal(err)
	}

	got, err := e.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(value, got) {
		t.Errorf("got %q, want %q", got, value)
	}
}

// 场景：同 key 写两次，Get 必须返回第二次的值（索引指到最新记录）。
func TestOverwrite(t *testing.T) {
	e, cleanup := newTestEngine(t)
	t.Cleanup(cleanup)

	key := []byte("key")
	value := []byte("value")
	value1 := []byte("value1")

	err := e.Put(key, value)
	if err != nil {
		t.Fatal(err)
	}

	err = e.Put(key, value1)
	if err != nil {
		t.Fatal(err)
	}

	got, err := e.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(value1, got) {
		t.Errorf("got %q, want %q", got, value1)
	}
}

// 场景：没写过的 key 返回 ErrKeyNotFound，不能是其他 IO 错误。
func TestGetKeyNotFound(t *testing.T) {
	e, cleanup := newTestEngine(t)
	t.Cleanup(cleanup)

	key := []byte("key")
	_, err := e.Get(key)
	if !errors.Is(err, ErrKeyNotFound) {
		t.Error(err)
	}
}

// 场景：Delete 之后 Get 报 ErrKeyNotFound，ListKeys 里也不再有它。
func TestDelete(t *testing.T) {
	e, cleanup := newTestEngine(t)
	t.Cleanup(cleanup)

	key := []byte("key")
	value := []byte("value")

	err := e.Put(key, value)
	if err != nil {
		t.Fatal(err)
	}

	err = e.Delete(key)
	if err != nil {
		t.Fatal(err)
	}

	_, err = e.Get(key)
	if !errors.Is(err, ErrKeyNotFound) {
		t.Fatal(err)
	}

	isExists := false
	keys := e.ListKeys()
	for _, i := range keys {
		if bytes.Equal(key, i) {
			isExists = true
		}
	}

	if isExists {
		t.Fatal("key still present in ListKeys after Delete")
	}
}

// 场景：Put 多个 key 后 ListKeys 与索引一一对应，数量与内容都对得上。
func TestListKeys(t *testing.T) {
	e, cleanup := newTestEngine(t)
	t.Cleanup(cleanup)

	keys := map[string][]byte{
		"key":  []byte("key"),
		"key1": []byte("key1"),
		"key2": []byte("key2"),
	}
	value := []byte("value")

	for _, v := range keys {
		err := e.Put(v, value)
		if err != nil {
			t.Fatal(err)
		}
	}

	keys1 := e.ListKeys()
	if len(keys1) != len(keys) {
		t.Fatalf("ListKeys returned %d keys, want %d", len(keys1), len(keys))
	}
	for _, v := range keys1 {
		if !bytes.Equal(v, keys[string(v)]) {
			t.Errorf("unexpected key %q", v)
		}
	}
}

// 场景：Close 正常返回，之后句柄真正释放（删临时目录不报错）。
func TestClose(t *testing.T) {
	e, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	err = e.Close()
	if err != nil {
		t.Fatal(err)
	}
}
