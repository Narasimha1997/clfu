package clfu_test

import (
	"clfu"
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
