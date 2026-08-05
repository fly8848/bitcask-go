package storage

import (
	"bitcask-go/record"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Storage struct {
	DataDir      string   // 数据目录路径
	ActiveFileId uint32   // 当前活跃文件的 id，Write 的返回值要用
	ActiveFile   *os.File // 当前正在写的文件，一直开着
	FileIds      []uint32 // 目录里所有文件的 id，按序排
	MaxFileSize  int64    // 活跃文件写满此值自动滚动新文件
}

func FileName(fileId uint32) string {
	return fmt.Sprintf("%09d.data", fileId)
}

func ParseFileId(name string) (uint32, error) {
	nameNumber, err := strconv.ParseUint(strings.TrimSuffix(name, ".data"), 10, 32)
	return uint32(nameNumber), err
}

func NewStorage(dir string) (*Storage, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	ids := buildFileIds(dirEntries)

	activeFileId := ids[len(ids)-1]
	f, err := openFile(dir, activeFileId)
	if err != nil {
		return nil, err
	}
	f.Seek(0, io.SeekEnd)

	return &Storage{
		DataDir:      dir,
		ActiveFile:   f,
		ActiveFileId: activeFileId,
		FileIds:      ids,
		MaxFileSize:  4 * 1024 * 1024,
	}, nil
}

func buildFileIds(dirEntries []os.DirEntry) []uint32 {
	ids := make([]uint32, 0)
	for _, i := range dirEntries {
		id, err := ParseFileId(i.Name())
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		ids = append(ids, 1)
	}
	return ids
}

func (s *Storage) Write(rec *record.LogRecord) (uint32, int64, error) {
	data := record.Encode(rec)

	size, err := s.ActiveFile.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, 0, err
	}

	if (len(data) + int(size)) > int(s.MaxFileSize) {
		if err := s.rollFile(); err != nil {
			return 0, 0, err
		}
	}

	// 重新 Seek 到末尾，拿到这条记录的 offset
	size, err = s.ActiveFile.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, 0, err
	}

	_, err = s.ActiveFile.Write(data)
	if err != nil {
		return 0, 0, err
	}

	return s.ActiveFileId, size, nil
}

// rollFile 关闭当前活跃文件，开一个新 id 的文件作为活跃文件
func (s *Storage) rollFile() error {
	if err := s.ActiveFile.Close(); err != nil {
		return err
	}

	newId := s.ActiveFileId + 1
	f, err := openFile(s.DataDir, newId)
	if err != nil {
		return err
	}

	s.FileIds = append(s.FileIds, newId)
	s.ActiveFile = f
	s.ActiveFileId = newId
	return nil
}

func (s *Storage) Read(fileId uint32, offset int64) (*record.LogRecord, error) {
	// 1. 文件选择：fileId == s.ActiveFileId 直接用 s.ActiveFile
	//    否则 os.OpenFile 按文件名开，读完 Close

	f := s.ActiveFile
	if s.ActiveFileId != fileId {
		file, err := os.OpenFile(s.DataDir+"/"+FileName(fileId), os.O_RDONLY, 0644)
		if err != nil {
			return nil, err
		}
		f = file
		defer file.Close()
	}

	// 读头部 12 字节，拿到这条记录的长度元信息
	header, keyLen, valueLen, err := readHeader(f, offset)
	if err != nil {
		return nil, err
	}

	body := make([]byte, keyLen+valueLen)
	_, err = f.ReadAt(body, offset+12)
	if err != nil {
		return nil, err
	}

	return record.Decode(append(header, body...))
}

func (s *Storage) Close() error {
	return s.ActiveFile.Close()
}

func openFile(dir string, fileid uint32) (*os.File, error) {
	return os.OpenFile(dir+"/"+FileName(fileid), os.O_CREATE|os.O_RDWR, 0644)
}

// readHeader 从文件 offset 处读头部 12 字节，返回头部本身和解析出的长度
func readHeader(f *os.File, offset int64) ([]byte, uint32, uint32, error) {
	header := make([]byte, 12)
	_, err := f.ReadAt(header, offset)
	if err != nil {
		return nil, 0, 0, err
	}
	return header, record.GetKeyLen(header), record.GetValueLen(header), nil
}
