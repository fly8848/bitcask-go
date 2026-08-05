package record

import (
	"bytes"
	"testing"
)

func TestConsistency(t *testing.T) {
	want := &LogRecord{
		Key:   []byte("test"),
		Value: []byte("test123"),
	}

	got, err := Decode(Encode(want))
	if err != nil {
		t.Fatal(err)
	}

	assertRecordEqual(t, got, want)
}

func TestEmptyValue(t *testing.T) {
	want := &LogRecord{
		Key:   []byte("test"),
		Value: nil,
	}

	got, err := Decode(Encode(want))
	if err != nil {
		t.Fatal(err)
	}

	assertRecordEqual(t, got, want)
}

func TestLongKey(t *testing.T) {
	want := &LogRecord{
		Key:   bytes.Repeat([]byte("k"), 1024),
		Value: []byte("value"),
	}

	got, err := Decode(Encode(want))
	if err != nil {
		t.Fatal(err)
	}

	assertRecordEqual(t, got, want)
}

func TestCrcError(t *testing.T) {
	rec := &LogRecord{
		Key:   []byte("test"),
		Value: []byte("test123"),
	}
	data := Encode(rec)
	data[len(data)-1] ^= 0xFF

	if _, err := Decode(data); err == nil {
		t.Error("篡改字节后应返回错误")
	}
}

func assertRecordEqual(t *testing.T, got, want *LogRecord) {
	t.Helper()
	if !bytes.Equal(got.Key, want.Key) {
		t.Errorf("key 不符: got %q, want %q", got.Key, want.Key)
	}
	if !bytes.Equal(got.Value, want.Value) {
		t.Errorf("value 不符: got %q, want %q", got.Value, want.Value)
	}
}
