package main

import (
	"clfu"
	"fmt"
)

func main() {

	// create a new instance of LFU cache with a max size
	lfuCache := clfu.NewLFUCache(3)

	// insert values, any interface{} can be used as key, value
	lfuCache.Put("u939801", 123, false)
	lfuCache.Put("u939802", 411, false)
	lfuCache.Put("u939803", 234, false)

	// insert with replace=true, will replace the value of 'u939802'
	lfuCache.Put("u939802", 512, true)

	// get the current size (should return '3')
	fmt.Printf("current_size=%d\n", lfuCache.CurrentSize())

	// get the max size (should return '3')
	fmt.Printf("max_size=%d\n", lfuCache.MaxSize())

	// check if the cache if full
	fmt.Printf("is_full=%v\n", lfuCache.IsFull())

	// get values (this will increase the frequency of given key 1)
	rawValue, found := lfuCache.Get("u939802")
	if found {
		fmt.Printf("Value of 'u939802' is %d\n", (*rawValue).(int))
	}

	rawValue, found = lfuCache.Get("u939803")
	if found {
		fmt.Printf("Value of 'u939803' is %d\n", (*rawValue).(int))
	}

	// insert new entry, should evict `u939801` now  because it is the least used element
	lfuCache.Put("u939804", 1000, false)

	// delete the entry from cache
	err := lfuCache.Delete("u939804")
	if err != nil {
		fmt.Printf("failed to delete, no key 'u939804'")
	}
}
