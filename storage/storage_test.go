package storage

import (
	"bitcask-go/record"
	"bytes"
	"testing"
)

func TestRollFile(t *testing.T) {
	st, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	st.MaxFileSize = 60

	rec := &record.LogRecord{
		Key:   []byte("testKey"),
		Value: []byte("testValue"),
	}

	fileId, offset, err := st.Write(rec)
	if err != nil {
		t.Fatal(err)
	}
	if fileId != 1 || offset != 0 {
		t.Errorf("首条记录位置不符: got (%d, %d), want (1, 0)", fileId, offset)
	}

	dataLen := int64(len(record.Encode(rec)))
	fileId, offset, err = st.Write(rec)
	if err != nil {
		t.Fatal(err)
	}
	if fileId != 1 || offset != dataLen {
		t.Errorf("未触发滚动: got (%d, %d), want (1, %d)", fileId, offset, dataLen)
	}

	fileId, offset, err = st.Write(rec)
	if err != nil {
		t.Fatal(err)
	}
	if fileId != 2 || offset != 0 {
		t.Errorf("应触发滚动: got (%d, %d), want (2, 0)", fileId, offset)
	}
}

func TestReadAcrossFiles(t *testing.T) {
	st, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	st.MaxFileSize = 60

	recs := []*record.LogRecord{
		{Key: []byte("key1"), Value: []byte("value1")},
		{Key: []byte("key2"), Value: []byte("value2")},
		{Key: []byte("key3"), Value: []byte("value3")},
	}

	for _, want := range recs {
		fileId, offset, err := st.Write(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := st.Read(fileId, offset)
		if err != nil {
			t.Fatal(err)
		}
		assertRecordEqual(t, got, want)
	}
}

func assertRecordEqual(t *testing.T, got, want *record.LogRecord) {
	t.Helper()
	if !bytes.Equal(got.Key, want.Key) {
		t.Errorf("key 不符: got %q, want %q", got.Key, want.Key)
	}
	if !bytes.Equal(got.Value, want.Value) {
		t.Errorf("value 不符: got %q, want %q", got.Value, want.Value)
	}
}
