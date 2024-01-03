package index

import (
	"hades/data"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNewSkipList_Put(t *testing.T) {
	sl := NewSkipList()
	res1 := sl.Put([]byte("key1"), &data.LogRecordPos{
		Fid:    1,
		Offset: 100,
		Size:   10,
	})
	assert.Nil(t, res1)

	res2 := sl.Put([]byte("key1"), &data.LogRecordPos{
		Fid:    2,
		Offset: 200,
		Size:   20,
	})
	assert.Equal(t, uint32(1), res2.Fid)
	assert.Equal(t, int64(100), res2.Offset)
	assert.Equal(t, uint32(10), res2.Size)
}

func TestSkipList_Get(t *testing.T) {
	sl := NewSkipList()
	res1 := sl.Put([]byte("key1"), &data.LogRecordPos{
		Fid:    1,
		Offset: 100,
		Size:   10,
	})
	assert.Nil(t, res1)

	pos1 := sl.Get([]byte("key1"))
	assert.Equal(t, uint32(1), pos1.Fid)
	assert.Equal(t, int64(100), pos1.Offset)
	assert.Equal(t, uint32(10), pos1.Size)

	pos2 := sl.Get([]byte("key2"))
	assert.Nil(t, pos2)
}

func TestSkipList_Delete(t *testing.T) {
	sl := NewSkipList()
	res1 := sl.Put([]byte("key1"), &data.LogRecordPos{
		Fid:    1,
		Offset: 100,
		Size:   10,
	})
	assert.Nil(t, res1)

	res2, ok := sl.Delete([]byte("key1"))
	assert.True(t, ok)
	assert.Equal(t, uint32(1), res2.Fid)
	assert.Equal(t, int64(100), res2.Offset)
	assert.Equal(t, uint32(10), res2.Size)

	res3, ok := sl.Delete([]byte("key2"))
	assert.False(t, ok)
	assert.Nil(t, res3)
}

func TestSkipList_Size(t *testing.T) {
	sl := NewSkipList()
	assert.Equal(t, 0, sl.Size())

	sl.Put([]byte("key1"), &data.LogRecordPos{
		Fid:    1,
		Offset: 100,
		Size:   10,
	})
	assert.Equal(t, 1, sl.Size())

	sl.Put([]byte("key2"), &data.LogRecordPos{
		Fid:    2,
		Offset: 200,
		Size:   20,
	})
	assert.Equal(t, 2, sl.Size())

	sl.Delete([]byte("key1"))
	assert.Equal(t, 1, sl.Size())

	sl.Delete([]byte("key2"))
	assert.Equal(t, 0, sl.Size())
}
