# clfu
![Tests](https://github.com/Narasimha1997/clfu/actions/workflows/test.yml/badge.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/Narasimha1997/clfu.svg)](https://pkg.go.dev/github.com/Narasimha1997/clfu)

Implementation of Constant Time LFU (least frequently used) cache in Go with concurrency safety. This implementation is based on the paper [An O(1) algorithm for implementing the LFU
cache eviction scheme](http://dhruvbird.com/lfu.pdf). As opposed to priority heap based LFU cache, this algorithm provides almost O(1) insertion, retrieval and eviction operations by using Linked lists and hash-maps instead of frequency based min-heap data-structure. This algorithm trade-offs memory to improve performance.

## Using the module
The codebase can be imported and used as a go-module. To add `clfu` as a go module dependency to your project, run:
```
go get github.com/Narasimha1997/clfu
```

## Example
The example below shows some of the basic operations provided by `clfu`:
```go
import (
	"clfu"
	"fmt"
)

func main() {

	// create a new instance of LFU cache with a max size, you can also use NewLazyLFUCache(size uint, lazyCacheSize uint) to use LFU cache
    // in lazy mode.
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
```

### Lazy mode
As per the algorithm specification, whenever we call a `Get` on given key, the linked list structure has to be modified to update the frequency of the given key by 1, in normal mode (i.e when lazy mode is disabled) the structural update to linked list is performed on every `Get`, this can be avoided using lazy mode, in lazy mode the frequency updates are not made immediately, instead a slice is used to keep track of the keys whose frequencies needs to be updated, once this slice reaches it's maximum capacity, the linked list is updated in bulk. The bulk update is also triggered when `Put` or `Evict` methods are called to make sure writes always happen on the correct state of the linked list, however the bulk update can also be triggered manually by calling `FlushLazyCounter`. This is how LFU cache can be instantiated in lazy mode:
```go
// here 1000 is the size of the LFU cache, 10000 is the size of lazy update slice,
// i.e bulk update on the linked list will be triggered once after every 10000 gets.
lfuCache := clfu.NewLFUCache(1000, 10000)

// you can also manually trigger lazy update
lfuCache.FlushLazyCounter()
```
Lazy mode is best suitable when `Put` operations are not so frequent and `Get` operations occur in very high volumes.

### Obtaining the cache entries as slice
The module provides `AsSlice()` method which can be used to obtain all the elements in LFU cache at that given point in time as a slice, the entries in the returned slice will be in the increasing order of their access frequency.
```go
lfuCache := clfu.NewLFUCache(3)

// insert some values
lfuCache.Put("u939801", 123, false)
lfuCache.Put("u939802", 411, false)
lfuCache.Put("u939803", 234, false)

// obtain them as slice
entries := lfuCache.AsSlice()
for _, entry := range *entries {
	fmt.Printf("Frequency=%d\n", entry.Frequency)
	fmt.Printf("Key=%s\n", (*entry.Key).(string))
	fmt.Printf("Value=%d\n", (*entry.Value).(int))
}
```

Similarly we can also use `GetTopFrequencyItems()` to obtain the list entries having highest frequency value and `GetLeastFrequencyItems()` to get the list of entries having least frequency value. 

### Testing
If you want to make modifications or validate the functions locally run the following command from project root:
```
go test -v
```

This will execute the testing suite against the module:
```
=== RUN   TestPut
--- PASS: TestPut (0.00s)
=== RUN   TestPutWithReplace
--- PASS: TestPutWithReplace (0.00s)
=== RUN   TestComplexStructPutAndGet
--- PASS: TestComplexStructPutAndGet (0.00s)
=== RUN   TestManualEvict
--- PASS: TestManualEvict (0.00s)
=== RUN   TestDelete
--- PASS: TestDelete (0.00s)
=== RUN   TestLeastAndFrequentItemsGetter
--- PASS: TestLeastAndFrequentItemsGetter (0.00s)
=== RUN   TestMaxSizeResize
--- PASS: TestMaxSizeResize (0.00s)
=== RUN   TestConcurrentPut
--- PASS: TestConcurrentPut (2.84s)
=== RUN   TestConcurrentGet
--- PASS: TestConcurrentGet (1.42s)
=== RUN   TestConcurrentLazyGet
--- PASS: TestConcurrentLazyGet (1.54s)
PASS
ok      clfu    5.800s
```

### Benchmarks
To run the benchmark suite, run the following command from the project root:
```
go test -bench=. -benchmem
```
This will run the benchmark suite against the module:
```
goos: linux
goarch: amd64
pkg: clfu
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkPut-12                          3022968               405.9 ns/op           129 B/op          6 allocs/op
BenchmarkConcurrentPut-12                2232406               522.9 ns/op           129 B/op          6 allocs/op
BenchmarkGetOperation-12                 7356638               136.7 ns/op            49 B/op          1 allocs/op
BenchmarkConcurrentGet-12                5105284               212.7 ns/op            51 B/op          1 allocs/op
BenchmarkLazyLFUGet-12                   7642183               153.8 ns/op            49 B/op          1 allocs/op
BenchmarkConcurrentLazyLFUGet-12         4828609               217.3 ns/op            51 B/op          1 allocs/op
PASS
ok      clfu    15.317s
```
Going with the above numbers we can achieve at max `2.46M PUTs` and `7.3M GETs` per second on a single core, in lazy mode we can get max `6.9M GETs`. Note that the LFU cache is suitable for linear operation and not concurrent operations because all the operations need a write lock to be imposed to avoid race conditions. From the benchmarks it is evident that the performance will decrease as we perform concurrent `PUT` and `GET` because of lock-contention. 

From `examples/main.go` we can also notice that on average, it takes `11.8s` to perform `48M GETs` using concurrency of 12. (tested on 12vCPU machine) and `13.4s` to do the same using lazy mode.

```
Normal (lazy not enabled): n_goroutines=12, n_access=48000000, time_taken=11.846151472s
Lazy (with size 1000 as cache size): n_goroutines=12, n_access=48000000, time_taken=13.357687219s
```

### Contributing
Feel free to raise issues, submit PRs or suggest improvements.