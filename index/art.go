package index

import (
	"bytes"
	"github.com/plar/go-adaptive-radix-tree"
	"hades/data"
	"sort"
	"sync"
)

// AdaptiveRadixTree 自适应基数树索引
type AdaptiveRadixTree struct {
	tree art.Tree
	lock *sync.RWMutex
}

func NewART() *AdaptiveRadixTree {
	return &AdaptiveRadixTree{
		tree: art.New(),
		lock: new(sync.RWMutex),
	}
}

func (art *AdaptiveRadixTree) Put(key []byte, pos *data.LogRecordPos) *data.LogRecordPos {
	art.lock.Lock()
	oldValue, _ := art.tree.Insert(key, pos)
	art.lock.Unlock()
	if oldValue == nil {
		return nil
	}
	return oldValue.(*data.LogRecordPos)
}

func (art *AdaptiveRadixTree) Get(key []byte) *data.LogRecordPos {
	art.lock.RLock()
	defer art.lock.RUnlock()
	value, found := art.tree.Search(key)
	if !found {
		return nil
	}
	return value.(*data.LogRecordPos)
}

func (art *AdaptiveRadixTree) Delete(key []byte) (*data.LogRecordPos, bool) {
	art.lock.Lock()
	oldValue, deleted := art.tree.Delete(key)
	art.lock.Unlock()
	if oldValue == nil {
		return nil, false
	}
	return oldValue.(*data.LogRecordPos), deleted
}

func (art *AdaptiveRadixTree) Size() int {
	art.lock.RLock()
	size := art.tree.Size()
	art.lock.RUnlock()
	return size
}

func (art *AdaptiveRadixTree) Iterator(reverse bool) Iterator {
	art.lock.RLock()
	defer art.lock.RUnlock()
	return newARTIterator(art.tree, reverse)
}

func (art *AdaptiveRadixTree) Close() error {
	return nil
}

// Art 索引迭代器
type artIterator struct {
	currIndex int     // 当前遍历的下标位置
	reverse   bool    //	是否反向遍历
	values    []*Item //key + 位置索引信息
}

// 这里相当于创建一个快照
func newARTIterator(tree art.Tree, reverse bool) *artIterator {
	var idx int

	if reverse {
		idx = tree.Size() - 1
	}

	values := make([]*Item, tree.Size())

	saveValues := func(node art.Node) bool {
		item := &Item{
			key: node.Key(),
			pos: node.Value().(*data.LogRecordPos),
		}
		values[idx] = item
		if reverse {
			idx--
		} else {
			idx++
		}
		return true
	}

	tree.ForEach(saveValues)

	return &artIterator{
		currIndex: 0,
		reverse:   reverse,
		values:    values,
	}
}

func (a *artIterator) Next() {
	a.currIndex += 1
}

func (a *artIterator) Rewind() {
	a.currIndex = 0
}

func (a *artIterator) Seek(key []byte) {
	if a.reverse {
		a.currIndex = sort.Search(len(a.values), func(i int) bool {
			return bytes.Compare(a.values[i].key, key) <= 0
		})
	} else {
		a.currIndex = sort.Search(len(a.values), func(i int) bool {
			return bytes.Compare(a.values[i].key, key) >= 0
		})
	}
}

func (a *artIterator) Valid() bool {
	return a.currIndex < len(a.values)
}

func (a *artIterator) Key() []byte {
	return a.values[a.currIndex].key
}

func (a *artIterator) Value() *data.LogRecordPos {
	return a.values[a.currIndex].pos
}

func (a *artIterator) Close() {
	a.values = nil
}
