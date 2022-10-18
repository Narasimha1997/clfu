package clfu

import (
	"container/list"
	"fmt"
	"sync"
)

type ValueType interface{}
type KeyType interface{}

// `KeyValueEntry` represents an item in the slice representation of LFU cache
type KeyValueEntry struct {
	Key       *KeyType   // pointer to key
	Value     *ValueType // pointer to value
	Frequency uint       // frequency of access
}

// `FrequencyNode` represents a node in the frequency linked list
type FrequencyNode struct {
	count      uint          // frequency count - never decreases
	valuesList *list.List    // valuesList contains pointer to the head of values linked list
	inner      *list.Element // actual content of the next element
}

// creates a new frequency list node with the given count
func newFrequencyNode(count uint) *FrequencyNode {
	return &FrequencyNode{
		count:      count,
		valuesList: list.New(),
		inner:      nil,
	}
}

// `KeyRefNode`` represents the value held on the LRU cache frequency list node
type KeyRefNode struct {
	inner          *list.Element // contains the actual value wrapped by a list element
	parentFreqNode *list.Element // contains reference to the frequency node element
	keyRef         *KeyType      // contains pointer to the key
	valueRef       *ValueType    // value
}

type LFULazyCounter struct {
	accessList []KeyType
	count      uint
	capacity   uint
}

// creates a new KeyRef node which is used to represent the value in linked list
func newKeyRefNode(keyRef *KeyType, valueRef *ValueType, parent *list.Element) *KeyRefNode {
	return &KeyRefNode{
		inner:          nil,
		parentFreqNode: parent,
		keyRef:         keyRef,
		valueRef:       valueRef,
	}
}

// `LFUCache` implements all the methods and data-structures required for LFU cache
type LFUCache struct {
	rwLock      sync.RWMutex            // rwLock is a read-write mutex which provides concurrent reads but exclusive writes
	lookupTable map[KeyType]*KeyRefNode // a hash table of <KeyType, *ValueType> for quick reference of values based on keys
	frequencies *list.List              // internal linked list that contains frequency mapping
	maxSize     uint                    // maxSize represents the maximum number of elements that can be in the cache before eviction
	isLazy      bool                    // if set to true, the frequency count update will happen lazily
	lazyCounter *LFULazyCounter         // contains pointer to lazy counter instance
}

// `MaxSize` returns the maximum size of the cache at that point in time
func (lfu *LFUCache) MaxSize() uint {
	lfu.rwLock.RLock()
	defer lfu.rwLock.RUnlock()

	return lfu.maxSize
}

// `CurrentSize` returns the number of elements in that cache
//
// Returns: `uint` representing the current size
func (lfu *LFUCache) CurrentSize() uint {
	lfu.rwLock.RLock()
	defer lfu.rwLock.RUnlock()

	return uint(len(lfu.lookupTable))
}

// `IsFull` checks if the LFU cache is full
//
// Returns (true/false), `true` if LFU cache is full, `false` if LFU cache is not full
func (lfu *LFUCache) IsFull() bool {
	lfu.rwLock.RLock()
	defer lfu.rwLock.RUnlock()

	return uint(len(lfu.lookupTable)) == lfu.maxSize
}

// `SetMaxSize` updates the max size of the LFU cache
//
// Parameters
//
// 1. size: `uint` value which specifies the new size of the LFU cache
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

// `Put` method inserts a `<KeyType, ValueType>` to the LFU cache and updates internal
// data structures to keep track of access frequencies, if the cache is full, it evicts the
// least frequently used value from the cache.
//
// Parameters:
//
// 1. key: Key is of `KeyType` (or simply an `interface{}`) which represents the key, note that the key must be hashable type.
//
// 2. value: Value is of `ValueType` (or simply an `interface{}`) which represents the value
//
// 3. replace: replace is a `bool`, if set to `true`, the `value` of the given `key` will be overwritten if exists, if set to
// `false`, an `error` is thrown if `key` already exists.
//
// Returns: `error` if there are any errors during insertions
func (lfu *LFUCache) Put(key KeyType, value ValueType, replace bool) error {
	// get write lock
	lfu.rwLock.Lock()
	defer lfu.rwLock.Unlock()

	if lfu.isLazy && lfu.lazyCounter.count > 0 {
		lfu.unsafeFlushLazyCounter()
	}

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

	valueNode := newKeyRefNode(&key, &value, nil)

	head := lfu.frequencies.Front()
	if head == nil {
		// fresh linked list
		freqNode := newFrequencyNode(1)
		head = lfu.frequencies.PushFront(freqNode)
		freqNode.inner = head

	} else {
		node := head.Value.(*FrequencyNode)
		if node.count != 1 {
			freqNode := newFrequencyNode(1)
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

// `Evict` can be called to manually perform eviction
//
// Returns: `error` if there are any errors during eviction
func (lfu *LFUCache) Evict() error {
	lfu.rwLock.Lock()
	defer lfu.rwLock.Unlock()

	if lfu.isLazy && lfu.lazyCounter.count > 0 {
		lfu.unsafeFlushLazyCounter()
	}

	return lfu.unsafeEvict()
}

func (lfu *LFUCache) unsafeUpdateFrequency(valueNode *KeyRefNode) {
	parentFreqNode := valueNode.parentFreqNode
	currentNode := parentFreqNode.Value.(*FrequencyNode)
	nextParentFreqNode := parentFreqNode.Next()

	var newParent *list.Element = nil

	if nextParentFreqNode == nil {
		// this is the last node
		// create a new node with frequency + 1
		newFreqNode := newFrequencyNode(currentNode.count + 1)
		lfu.frequencies.PushBack(newFreqNode)
		newParent = parentFreqNode.Next()

	} else {
		nextNode := nextParentFreqNode.Value.(*FrequencyNode)
		if nextNode.count == (currentNode.count + 1) {
			newParent = nextParentFreqNode
		} else {
			// insert a node in between
			newFreqNode := newFrequencyNode(currentNode.count + 1)

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
}

// unsafeFlushLazyCounter flushes the updates in lazy counter without locking
func (lfu *LFUCache) unsafeFlushLazyCounter() error {
	// WARNING: calling this function directly is not recommended, because
	// this function assumes caller has a RWLock over the LFU cache.

	for i := 0; i < int(lfu.lazyCounter.count); i++ {
		key := lfu.lazyCounter.accessList[i]
		valueNode, found := lfu.lookupTable[key]
		if !found {
			return fmt.Errorf("key %v not found", key)
		}

		lfu.unsafeUpdateFrequency(valueNode)
	}

	lfu.lazyCounter.count = 0

	return nil
}

// FlushLazyCounter updates the state LFU cache with pending frequency updates in lazy counter
//
// Returns: error if lazy update fails
func (lfu *LFUCache) FlushLazyCounter() error {
	lfu.rwLock.Lock()
	defer lfu.rwLock.Unlock()

	return lfu.unsafeFlushLazyCounter()
}

// `Get` can be called to obtain the value for given key
//
// Parameters:
//
// key: key: Key is of `KeyType` (or simply an `interface{}`) which represents the key, note that the key must be hashable type
//
// Returns: `(*ValueType, bool)` - returns a pointer to the value in LFU cache if `key` exists, else it will be `nil` with `error` non-nil.
func (lfu *LFUCache) Get(key KeyType) (*ValueType, bool) {
	if !lfu.isLazy {
		lfu.rwLock.Lock()
		defer lfu.rwLock.Unlock()

		// check if data is in the map
		valueNode, found := lfu.lookupTable[key]
		if !found {
			return nil, false
		}

		lfu.unsafeUpdateFrequency(valueNode)

		return valueNode.valueRef, true
	} else {
		lfu.rwLock.Lock()
		defer lfu.rwLock.Unlock()

		// is lazy update list full?
		if lfu.lazyCounter.count >= lfu.lazyCounter.capacity {
			err := lfu.unsafeFlushLazyCounter()
			if err != nil {
				return nil, false
			}
		}

		// perform get
		valueNode, found := lfu.lookupTable[key]
		if !found {
			return nil, false
		}

		// update the lazy counter
		lfu.lazyCounter.accessList[lfu.lazyCounter.count] = key
		lfu.lazyCounter.count += 1

		return valueNode.valueRef, true
	}
}

// `Delete` removes the specified entry from LFU cache
//
// Parameters:
//
// key: key is of type `KeyType` (or simply `interface{}`) which represents the key to be deleted
//
// Returns: `error` which is nil if `key` is deleted, non-nil if there are some errors while deletion
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

// `AsSlice` returns the list of all elements in the key lfu cache and their frequencies
//
// Returns: a pointer to the slice of `KeyValueEntry` type, which contains the list of elements (key, value and frequency) in the current
// state of LFU cache.
func (lfu *LFUCache) AsSlice() *[]KeyValueEntry {
	lfu.rwLock.RLock()
	defer lfu.rwLock.RUnlock()

	valuesList := make([]KeyValueEntry, 0)

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

// `GetTopFrequencyItems` returns the list of all elements in the key lfu cache and their frequencies
//
// Returns: a pointer to the slice of `KeyValueEntry` type, which contains the list of elements (key, value and frequency) having
// highest frequency value in the current state of the LFU cache.
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

// `GetLeastFrequencyItems` returns the list of all elements in the key lfu cache and their frequencies
//
// Returns: a pointer to the slice of `KeyValueEntry` type, which contains the list of elements (key, value and frequency) having
// least frequency value in the current state of the LFU cache.
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

// `NewLFUCache` returns the new instance of LFU cache with specified size.
//
// Parameters:
//
// 1. maxSize: `uint` representing the max size of LFU cache.
//
// Returns: (*LFUCache) a pointer to LFU cache instance.
func NewLFUCache(maxSize uint) *LFUCache {
	return &LFUCache{
		rwLock:      sync.RWMutex{},
		lookupTable: make(map[KeyType]*KeyRefNode),
		maxSize:     maxSize,
		frequencies: list.New(),
	}
}

// `NewLFUCache` returns the new instance of LFU cache with specified size and lazy mode enabled.
//
// Parameters:
//
// 1. maxSize: `uint` representing the max size of LFU cache.
//
// 2. lazyCounterSize: size of lazy counter to use, frequencies will be updated in batch or when some write happens or manually FlushLazyCounter is called.
// Returns: (*LFUCache) a pointer to LFU cache instance.
func NewLazyLFUCache(maxSize uint, lazyCounterSize uint) *LFUCache {
	lazyCounter := LFULazyCounter{
		count:      0,
		capacity:   lazyCounterSize,
		accessList: make([]KeyType, lazyCounterSize),
	}

	lfuCache := &LFUCache{
		rwLock:      sync.RWMutex{},
		lookupTable: make(map[KeyType]*KeyRefNode),
		maxSize:     maxSize,
		frequencies: list.New(),
		isLazy:      true,
		lazyCounter: &lazyCounter,
	}

	return lfuCache
}
