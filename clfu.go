package main

import (
	"container/list"
	"fmt"
	"sync"
)

type ValueType interface{}
type KeyType interface{}

// FrequencyNode represents a node in the frequency linked list
type FrequencyNode struct {
	// frequency count - never decreases
	count uint
	// valuesList contains pointer to the head of values linked list
	valuesList *list.List
	// actual content of the next element
	inner *list.Element
}

// creates a new frequency list node with the given count
func NewFrequencyNode(count uint) *FrequencyNode {
	return &FrequencyNode{
		count:      count,
		valuesList: list.New(),
		inner:      nil,
	}
}

// KeyRefNode represents the value held on the LRU cache frequency list node
type KeyRefNode struct {
	// contains the actual value wrapped by a list element
	inner *list.Element
	// contains reference to the frequency node element
	parentFreqNode *list.Element
	// contains pointer to the key
	keyRef *KeyType
	// value
	valueRef *ValueType
}

func NewKeyRefNode(keyRef *KeyType, valueRef *ValueType, parent *list.Element) *KeyRefNode {
	return &KeyRefNode{
		inner:          nil,
		parentFreqNode: parent,
		keyRef:         keyRef,
		valueRef:       valueRef,
	}
}

// LFUCache implements all the methods and data-structures required for LFU cache
type LFUCache struct {
	// rwLock is a read-write mutex which provides concurrent reads but exclusive writes
	rwLock sync.RWMutex
	// a hash table of <KeyType, *ValueType> for quick reference of values based on keys
	lookupTable map[KeyType]*KeyRefNode
	// internal linked list that contains frequency mapping
	frequencies *list.List
	// maxSize represents the maximum number of elements that can be in the cache before eviction
	maxSize uint
}

// MaxSize returns the maximum size of the cache at that point in time
func (lfu *LFUCache) MaxSize() uint {
	lfu.rwLock.RLock()
	defer lfu.rwLock.RUnlock()

	return lfu.maxSize
}

// CurrentSize returns the number of elements in that cache
func (lfu *LFUCache) CurrentSize() uint {
	lfu.rwLock.RLock()
	defer lfu.rwLock.RUnlock()

	return uint(len(lfu.lookupTable))
}

func (lfu *LFUCache) IsFull() bool {
	lfu.rwLock.RLock()
	defer lfu.rwLock.RUnlock()

	return uint(len(lfu.lookupTable)) == lfu.maxSize
}

// customize the max size of the cache
func (lfu *LFUCache) SetMaxSize(size uint) {
	lfu.rwLock.Lock()
	defer lfu.rwLock.Unlock()

	lfu.maxSize = size
}

func (lfu *LFUCache) Put(key KeyType, value ValueType, replace bool) error {
	// get write lock
	lfu.rwLock.Lock()
	defer lfu.rwLock.Unlock()

	if _, ok := lfu.lookupTable[key]; ok {
		if replace {
			// update the cache value
			lfu.lookupTable[key].valueRef = &value
			return nil
		}

		return fmt.Errorf("inserting a new entry to the cache exceeds the max size %d", lfu.maxSize)
	}

	if lfu.maxSize == uint(len(lfu.lookupTable)) {
		lfu.unsafeEvict()
	}

	valueNode := NewKeyRefNode(&key, &value, nil)

	head := lfu.frequencies.Front()
	if head == nil {
		// fresh linked list
		freqNode := NewFrequencyNode(1)
		head = lfu.frequencies.PushFront(freqNode)
		freqNode.inner = head

	} else {
		node := head.Value.(*FrequencyNode)
		if node.count != 1 {
			freqNode := NewFrequencyNode(1)
			head = lfu.frequencies.PushFront(freqNode)
			freqNode.inner = head
		}
	}

	valueNode.parentFreqNode = head
	node := head.Value.(*FrequencyNode)
	head = node.valuesList.PushBack(valueNode)
	valueNode.inner = head

	lfu.lookupTable[key] = valueNode
	return nil
}

// evict the least recently used element from the cache, this function is unsafe to be called externally
// because it doesn't provide locking mechanism.
func (lfu *LFUCache) unsafeEvict() error {
	// WARNING: This function assumes that a write lock has been held by the caller already
	// get the head node of the list
	headFreq := lfu.frequencies.Front()
	if headFreq == nil {
		// list is empty, this is a very unusual condition
		return fmt.Errorf("internal error: failed to evict, empty frequency list")
	}

	headFreqInner := (headFreq.Value).(*FrequencyNode)

	if headFreqInner.valuesList.Len() == 0 {
		// again this is a very unusual condition
		return fmt.Errorf("internal error: failed to evict, empty values list")
	}

	headValuesList := headFreqInner.valuesList
	// pop the head of this this values list
	headValueNode := headValuesList.Front()
	removeResult := headValuesList.Remove(headValueNode).(*KeyRefNode)

	// update the values list
	headFreqInner.valuesList = headValuesList

	// remove the key from lookup table
	key := removeResult.keyRef
	delete(lfu.lookupTable, *key)
	return nil
}

// create a new instance of LFU cache
func NewLFUCache(maxSize uint) *LFUCache {
	return &LFUCache{
		rwLock:      sync.RWMutex{},
		lookupTable: make(map[KeyType]*KeyRefNode),
		maxSize:     maxSize,
		frequencies: list.New(),
	}
}
