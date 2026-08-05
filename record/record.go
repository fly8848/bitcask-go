package record

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
)

// 磁盘布局：一条记录 = 头部 12B + body
//   keyLen(4B) | valueLen(4B) | crc(4B) | key | value
// valueLen == 0 表示墓碑（删除标记）
// crc 只算 body(key+value)，放头部是为了先读 12B 就能决定是否读 body
type LogRecord struct {
	Key   []byte
	Value []byte
}

func Encode(record *LogRecord) []byte {
	header := make([]byte, 12)
	binary.BigEndian.PutUint32(header[0:4], uint32(len(record.Key)))
	binary.BigEndian.PutUint32(header[4:8], uint32(len(record.Value)))

	body := make([]byte, 0)
	body = append(body, record.Key...)
	body = append(body, record.Value...)

	binary.BigEndian.PutUint32(header[8:12], crc32.ChecksumIEEE(body))

	data := make([]byte, 0)
	data = append(data, header...)
	data = append(data, record.Key...)
	data = append(data, record.Value...)

	return data
}

func Decode(data []byte) (*LogRecord, error) {
	keyLen := GetKeyLen(data)
	valueLen := GetValueLen(data)
	crcByte := binary.BigEndian.Uint32(data[8:12])

	if crcByte != crc32.ChecksumIEEE(data[12:]) {
		return nil, errors.New("crc对比不正确")
	}

	key := data[12 : 12+keyLen]
	value := data[12+keyLen : 12+keyLen+valueLen]

	return &LogRecord{
		Key:   key,
		Value: value,
	}, nil
}

func GetKeyLen(data []byte) uint32 {
	return binary.BigEndian.Uint32(data[0:4])
}

func GetValueLen(data []byte) uint32 {
	return binary.BigEndian.Uint32(data[4:8])
}
