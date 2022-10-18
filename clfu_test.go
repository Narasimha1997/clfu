package clfu_test

import (
	"clfu"
	"sync"
	"testing"
)

func TestPut(t *testing.T) {

	// create a new LFU cache with size=10
	lfu := clfu.NewLFUCache(10)

	// insert 1000 elements with replace=false
	for i := 1; i <= 1000; i++ {
		err := lfu.Put(i, i, false)
		if err != nil {
			t.Fatalf("error while inserting key value paris to LFU cache, error=%s", err.Error())
		}
	}

	// verify the elements inserted
	if lfu.CurrentSize() != 10 {
		t.Fatalf("expected size of LFU cache was 10, but got %d", lfu.CurrentSize())
	}

	allElements := lfu.AsSlice()
	for i := 0; i < 10; i++ {
		value := (*(*allElements)[i].Value).(int)
		if value != (i + 991) {
			t.Fatalf("invalid value in the cache, expected %d, but got %d", value, i+991)
		}
	}
}

func TestPutWithReplace(t *testing.T) {
	lfu := clfu.NewLFUCache(1)

	// insert an element
	err := lfu.Put(1, 1, false)
	if err != nil {
		t.Fatalf("error while inserting key value paris to LFU cache, error=%s", err.Error())
	}

	// insert with replace
	err = lfu.Put(1, 1000, true)
	if err != nil {
		t.Fatalf("error while inserting key value paris to LFU cache, error=%s", err.Error())
	}

	// get and check the value
	valueRaw, found := lfu.Get(1)
	if !found {
		t.Fatalf("key '1' not found")
	}

	value := (*valueRaw).(int)
	if value != 1000 {
		t.Fatalf("expected value of replacing the key with insert was 1000 but got %d", value)
	}
}

func TestComplexStructPutAndGet(t *testing.T) {

	type SampleStructValue struct {
		Name     string
		Value    string
		Age      int
		Elements []int
	}

	// create a new LFU cache with size=10
	lfu := clfu.NewLFUCache(10)

	sampleStructValue := SampleStructValue{
		Name:     "test",
		Value:    "test-xxxxx",
		Age:      100000,
		Elements: []int{10, 20, 30, 40},
	}

	err := lfu.Put("my-test-sample-key", sampleStructValue, false)
	if err != nil {
		t.Fatalf("error while inserting key value paris to LFU cache, error=%s", err.Error())
	}

	valueRaw, found := lfu.Get("my-test-sample-key")
	if !found {
		t.Fatalf("key 'my-test-sample-key' not found")
	}

	value := (*valueRaw).(SampleStructValue)
	allGood := value.Name == "test" && value.Value == "test-xxxxx" && value.Age == 100000 && len(value.Elements) == 4
	if !allGood {
		t.Fatalf("improper value read from the cache")
	}
}

func TestManualEvict(t *testing.T) {
	// create a new LFU cache with size=10
	lfu := clfu.NewLFUCache(10)

	// insert 1000 elements with replace=false
	for i := 1; i <= 1000; i++ {
		err := lfu.Put(i, i, false)
		if err != nil {
			t.Fatalf("error while inserting key value paris to LFU cache, error=%s", err.Error())
		}
	}

	// verify the elements inserted
	if lfu.CurrentSize() != 10 {
		t.Fatalf("expected size of LFU cache was 10, but got %d", lfu.CurrentSize())
	}

	// increase the frequency of last 5 elements
	for i := 991; i <= 995; i++ {
		lfu.Get(i)
	}

	// now evict times
	for i := 0; i < 5; i++ {
		lfu.Evict()
	}

	// verify the elements inserted
	if lfu.CurrentSize() != 5 {
		t.Fatalf("expected size of LFU cache was 5, but got %d", lfu.CurrentSize())
	}

	// now the remaining elements from 991 to 994
	allElements := lfu.AsSlice()
	for i := 0; i < 5; i++ {
		value := (*(*allElements)[i].Value).(int)
		if value != (i + 991) {
			t.Fatalf("invalid value in the cache, expected %d, but got %d", i+991, value)
		}
	}
}

func TestDelete(t *testing.T) {
	// create a new LFU cache with size=10
	lfu := clfu.NewLFUCache(10)

	// insert 1000 elements with replace=false
	for i := 1; i <= 1000; i++ {
		err := lfu.Put(i, i, false)
		if err != nil {
			t.Fatalf("error while inserting key value paris to LFU cache, error=%s", err.Error())
		}
	}

	// verify the elements inserted
	if lfu.CurrentSize() != 10 {
		t.Fatalf("expected size of LFU cache was 10, but got %d", lfu.CurrentSize())
	}

	// delete the odd  elements
	for i := 1; i <= 10; i++ {
		if i&1 == 1 {
			err := lfu.Delete(990 + i)
			if err != nil {
				t.Fatalf("error while deleting value from the cache, error=%s", err.Error())
			}
		}
	}

	// verify the presence of even elements
	for i := 1; i <= 10; i++ {
		if i&1 == 0 {
			_, found := lfu.Get(i + 990)
			if !found {
				t.Fatalf("expected key %d to be present in the cache, but it is not found", i+990)
			}
		}
	}
}

func TestLeastAndFrequentItemsGetter(t *testing.T) {
	lfu := clfu.NewLFUCache(10)

	// insert 1000 elements with replace=false
	for i := 1; i <= 1000; i++ {
		err := lfu.Put(i, i, false)
		if err != nil {
			t.Fatalf("error while inserting key value paris to LFU cache, error=%s", err.Error())
		}
	}

	// verify the elements inserted
	if lfu.CurrentSize() != 10 {
		t.Fatalf("expected size of LFU cache was 10, but got %d", lfu.CurrentSize())
	}

	// increase the frequency of first 5 elements
	for i := 991; i <= 995; i++ {
		lfu.Get(i)
	}

	// least frequency items - 996 to 1000
	allElements := lfu.GetLeastFrequencyItems()
	for i := 0; i < 5; i++ {
		value := (*(*allElements)[i].Value).(int)
		if value != (i + 996) {
			t.Fatalf("invalid value in the cache, expected %d, but got %d", i+996, value)
		}
	}

	// top frequency items - 991 to 995
	allElements = lfu.GetTopFrequencyItems()
	for i := 0; i < 5; i++ {
		value := (*(*allElements)[i].Value).(int)
		if value != (i + 991) {
			t.Fatalf("invalid value in the cache, expected %d, but got %d", i+991, value)
		}
	}
}

func TestMaxSizeResize(t *testing.T) {
	lfu := clfu.NewLFUCache(10)

	// insert 1000 elements with replace=false
	for i := 1; i <= 1000; i++ {
		err := lfu.Put(i, i, false)
		if err != nil {
			t.Fatalf("error while inserting key value paris to LFU cache, error=%s", err.Error())
		}
	}

	lfu.SetMaxSize(30)
	// insert 1000 elements with replace=false
	for i := 1; i <= 1000; i++ {
		err := lfu.Put(i, i, false)
		if err != nil {
			t.Fatalf("error while inserting key value paris to LFU cache, error=%s", err.Error())
		}
	}

	allGood := lfu.MaxSize() == lfu.CurrentSize()
	if !allGood {
		t.Fatalf("expected the size of cache be 30, but got %d", lfu.CurrentSize())
	}
}

func TestConcurrentPut(t *testing.T) {

	// create a cache with small size
	lfu := clfu.NewLFUCache(1000)

	// 4 goroutines will be inserting values 1M each
	insertOp := func(wg *sync.WaitGroup, from int, to int) {
		defer wg.Done()
		for i := from; i < to; i++ {
			lfu.Put(i, i, false)
		}
	}

	wg := sync.WaitGroup{}

	for i := 0; i < 4; i++ {
		wg.Add(1)
		start := i * 1000000
		go insertOp(&wg, start, start+1000000)
	}

	wg.Wait()

	if lfu.CurrentSize() != 1000 {
		t.Fatalf("expected the size of cache be 1000, but got %d", lfu.CurrentSize())
	}
}

func TestConcurrentGet(t *testing.T) {

	lfu := clfu.NewLFUCache(100000)

	for i := 0; i < 10000; i++ {
		lfu.Put(i, i, false)
	}

	// 4 goroutines will be getting values 1M each
	getOp := func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			for j := 0; j < 10000; j++ {
				lfu.Get(j)
			}
		}
	}

	wg := sync.WaitGroup{}

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go getOp(&wg)
	}

	wg.Wait()

	topElements := lfu.GetTopFrequencyItems()

	// all elements must be accessed 400 times
	allGood := len(*topElements) == 10000 && ((*topElements)[0].Frequency == 400)
	if allGood {
		t.Fatalf("expected all the elements to be accessed 400 times")
	}
}

func BenchmarkPut(b *testing.B) {
	lfu := clfu.NewLFUCache(1000)
	// insert 10M elements
	for i := 0; i < b.N; i++ {
		lfu.Put(i, i, false)
	}
}

func BenchmarkConcurrentPut(b *testing.B) {
	lfu := clfu.NewLFUCache(1000)

	i := 0

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {

			lfu.Put(i, i, false)
			i++
		}
	})
}

func BenchmarkGetOperation(b *testing.B) {
	lfu := clfu.NewLFUCache(100)
	for i := 0; i < 100; i++ {
		lfu.Put(i, i, false)
	}

	for i := 0; i < b.N; i++ {
		lfu.Get(i % 100)
	}
}

func BenchmarkConcurrentGet(b *testing.B) {
	lfu := clfu.NewLFUCache(100)
	for i := 0; i < 100; i++ {
		lfu.Put(i, i, false)
	}

	i := 0

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {

			lfu.Get(i % 100)
			i++
		}
	})
}

func BenchmarkLazyLFUGet(b *testing.B) {
	lfu := clfu.NewLazyLFUCache(100, 19)
	for i := 0; i < 100; i++ {
		lfu.Put(i, i, false)
	}

	for i := 0; i < b.N; i++ {
		lfu.Get(i % 100)
	}
}

func BenchmarkConcurrentLazyLFUGet(b *testing.B) {
	lfu := clfu.NewLazyLFUCache(100, 100)
	for i := 0; i < 100; i++ {
		lfu.Put(i, i, false)
	}

	i := 0

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {

			lfu.Get(i % 100)
			i++
		}
	})
}
