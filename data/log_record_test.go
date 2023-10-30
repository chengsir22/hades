package data

import (
	"github.com/stretchr/testify/assert"
	"hash/crc32"
	"testing"
)

func TestEncodeLogRecord(t *testing.T) {
	// 正常情况
	rec1 := &LogRecord{
		Key:   []byte("hello"),
		Value: []byte("world"),
		Type:  LogRecordNormal,
	}
	res1, n1 := EncodeLogRecord(rec1)
	t.Log(res1)
	assert.NotNil(t, res1)
	assert.Greater(t, n1, int64(5))

	// value 为空的情况
	rec2 := &LogRecord{
		Key:  []byte("hello"),
		Type: LogRecordNormal,
	}
	res2, n2 := EncodeLogRecord(rec2)
	assert.NotNil(t, res2)
	assert.Greater(t, n2, int64(5))

	// 对 Deleted 情况的测试
	rec3 := &LogRecord{
		Key:   []byte("hello"),
		Value: []byte("world"),
		Type:  LogRecordDeleted,
	}
	res3, n3 := EncodeLogRecord(rec3)
	assert.NotNil(t, res3)
	assert.Greater(t, n3, int64(5))
}

func TestDecodeLogRecordHeader(t *testing.T) {
	headerBuf1 := []byte{188, 190, 141, 29, 0, 10, 10, 104, 101, 108, 108, 111, 119, 111, 114, 108, 100}
	h1, size1 := decodeLogRecordHeader(headerBuf1)
	assert.NotNil(t, h1)
	assert.Equal(t, int64(7), size1)
	assert.Equal(t, uint32(495828668), h1.crc)
	assert.Equal(t, LogRecordNormal, h1.recordType)
	assert.Equal(t, uint32(5), h1.keySize)
	assert.Equal(t, uint32(5), h1.valueSize)
}

func TestGetLogRecordCRC(t *testing.T) {
	rec1 := &LogRecord{
		Key:   []byte("hello"),
		Value: []byte("world"),
		Type:  LogRecordNormal,
	}
	headerBuf1 := []byte{188, 190, 141, 29, 0, 10, 10}
	crc1 := getLogRecordCRC(rec1, headerBuf1[crc32.Size:])
	assert.Equal(t, uint32(495828668), crc1)
}
