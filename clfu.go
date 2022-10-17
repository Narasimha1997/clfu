package main

import (
	"container/list"
	"fmt"
	"sync"
)

type ValueType interface{}
type KeyType interface{}

type KeyValueEntry struct {
	Key       *KeyType
	Value     *ValueType
	Frequency uint
}

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

	if headFreqInner.valuesList.Len() == 0 && headFreqInner.count > 1 {
		// this node can be removed from the frequency list
		freqList := lfu.frequencies
		freqList.Remove(headFreq)
		lfu.frequencies = freqList
	}

	// remove the key from lookup table
	key := removeResult.keyRef
	delete(lfu.lookupTable, *key)
	return nil
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

		return fmt.Errorf("key %v already found in the cache", key)
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

// Evict can be called to manually perform eviction
func (lfu *LFUCache) Evict() error {
	lfu.rwLock.Lock()
	defer lfu.rwLock.Unlock()

	return lfu.unsafeEvict()
}

// Get can be called to obtain the value for given key
func (lfu *LFUCache) Get(key KeyType) (*ValueType, bool) {
	lfu.rwLock.Lock()
	defer lfu.rwLock.Unlock()

	// check if data is in the map
	valueNode, found := lfu.lookupTable[key]
	if !found {
		return nil, false
	}

	parentFreqNode := valueNode.parentFreqNode
	currentNode := parentFreqNode.Value.(*FrequencyNode)
	nextParentFreqNode := parentFreqNode.Next()

	var newParent *list.Element = nil

	if nextParentFreqNode == nil {
		// this is the last node
		// create a new node with frequency + 1
		newFreqNode := NewFrequencyNode(currentNode.count + 1)
		lfu.frequencies.PushBack(newFreqNode)
		newParent = parentFreqNode.Next()

	} else {
		nextNode := nextParentFreqNode.Value.(*FrequencyNode)
		if nextNode.count == (currentNode.count + 1) {
			newParent = nextParentFreqNode
		} else {
			// insert a node in between
			newFreqNode := NewFrequencyNode(currentNode.count + 1)

			lfu.frequencies.InsertAfter(newFreqNode, parentFreqNode)
			newParent = parentFreqNode.Next()
		}
	}

	// remove from the existing list
	currentNode.valuesList.Remove(valueNode.inner)

	newParentNode := newParent.Value.(*FrequencyNode)
	valueNode.parentFreqNode = newParent
	newValueNode := newParentNode.valuesList.PushBack(valueNode)
	valueNode.inner = newValueNode

	// check if the current node is empty
	if currentNode.valuesList.Len() == 0 {
		// remove the current node
		lfu.frequencies.Remove(parentFreqNode)
	}

	return valueNode.valueRef, true
}

func (lfu *LFUCache) Delete(key KeyType) error {
	lfu.rwLock.Lock()
	defer lfu.rwLock.Unlock()

	// check if the key is in the map
	valueNode, found := lfu.lookupTable[key]
	if !found {
		return fmt.Errorf("key %v not found", key)
	}

	parentFreqNode := valueNode.parentFreqNode

	currentNode := (parentFreqNode.Value).(*FrequencyNode)
	currentNode.valuesList.Remove(valueNode.inner)

	if currentNode.valuesList.Len() == 0 {
		lfu.frequencies.Remove(parentFreqNode)
	}

	delete(lfu.lookupTable, key)
	return nil
}

// obtain the list of all elements in the key lfu cache and their frequencies
func (lfu *LFUCache) AsSlice() *[]KeyValueEntry {
	lfu.rwLock.RLock()
	defer lfu.rwLock.RUnlock()

	valuesList := make([]KeyValueEntry, 0)
	// remove the current node

	for current := lfu.frequencies.Front(); current != nil; current = current.Next() {
		currentNode := current.Value.(*FrequencyNode)
		count := currentNode.count
		for value := currentNode.valuesList.Front(); value != nil; value = value.Next() {
			valueNode := (value.Value).(*KeyRefNode)
			valuesList = append(valuesList, KeyValueEntry{
				Key:       valueNode.keyRef,
				Value:     valueNode.valueRef,
				Frequency: count,
			})
		}
	}

	return &valuesList
}

// obtain the list of key-value pairs that have highest access frequency
func (lfu *LFUCache) GetTopFrequencyItems() *[]KeyValueEntry {
	lfu.rwLock.RLock()
	defer lfu.rwLock.RUnlock()

	valuesList := make([]KeyValueEntry, 0)

	current := lfu.frequencies.Back()
	if current == nil {
		return &valuesList
	}

	currentNode := current.Value.(*FrequencyNode)
	count := currentNode.count
	for value := currentNode.valuesList.Front(); value != nil; value = value.Next() {
		valueNode := (value.Value).(*KeyRefNode)
		valuesList = append(valuesList, KeyValueEntry{
			Key:       valueNode.keyRef,
			Value:     valueNode.valueRef,
			Frequency: count,
		})
	}

	return &valuesList
}

// obtain the list of key-value pairs that have lowest access frequency
func (lfu *LFUCache) GetLeastFrequencyItems() *[]KeyValueEntry {
	lfu.rwLock.RLock()
	defer lfu.rwLock.RUnlock()

	valuesList := make([]KeyValueEntry, 0)

	current := lfu.frequencies.Front()
	if current == nil {
		return &valuesList
	}

	currentNode := current.Value.(*FrequencyNode)
	count := currentNode.count
	for value := currentNode.valuesList.Front(); value != nil; value = value.Next() {
		valueNode := (value.Value).(*KeyRefNode)
		valuesList = append(valuesList, KeyValueEntry{
			Key:       valueNode.keyRef,
			Value:     valueNode.valueRef,
			Frequency: count,
		})
	}

	return &valuesList
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
