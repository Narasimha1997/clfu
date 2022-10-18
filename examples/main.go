package main

import (
	"clfu"
	"fmt"
	"runtime"
	"sync"
	"time"
)

func normal() {
	lfuCache := clfu.NewLFUCache(1000)
	for i := 0; i < 1000; i++ {
		lfuCache.Put(i, i, false)
	}

	// get the data 4M times and compute the latency

	routine := func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < 4*1000000; i++ {
			lfuCache.Get(i % 1000)
		}
	}

	wg := sync.WaitGroup{}
	st := time.Now()
	// run a goroutine for each CPU
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go routine(&wg)
	}

	wg.Wait()

	et := time.Since(st)
	fmt.Printf("Normal (lazy not enabled): n_goroutines=%d, n_access=%d, time_taken=%v\n", runtime.NumCPU(), runtime.NumCPU()*4*1000000, et)
}

func lazy() {
	lfuCache := clfu.NewLazyLFUCache(1000, 1000)
	for i := 0; i < 1000; i++ {
		lfuCache.Put(i, i, false)
	}

	// get the data 4M times and compute the latency

	routine := func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < 4*1000000; i++ {
			lfuCache.Get(i % 1000)
		}
	}

	wg := sync.WaitGroup{}
	st := time.Now()
	// run a goroutine for each CPU
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go routine(&wg)
	}

	wg.Wait()

	et := time.Since(st)
	fmt.Printf("Lazy (with size 1000 as cache size): n_goroutines=%d, n_access=%d, time_taken=%v\n", runtime.NumCPU(), runtime.NumCPU()*4*1000000, et)
}

func timeTaken() {
	// this function is very compute intensive

	fmt.Println("Checking lazy vs normal execution speeds")

	normal()

	lazy()

}

func averageAccessTimeNormal() {
	lfuCache := clfu.NewLFUCache(1000)
	for i := 0; i < 1000; i++ {
		lfuCache.Put(i, i, false)
	}

	// get the data 4M times and compute the latency
	totalTime := 0

	routine := func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < 4*1000000; i++ {
			st := time.Now()
			lfuCache.Get(i % 1000)
			et := time.Since(st)
			totalTime += int(et.Nanoseconds())
		}
	}

	wg := sync.WaitGroup{}
	// run a goroutine for each CPU
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go routine(&wg)
	}

	wg.Wait()

	averageTotalTime := totalTime / (4 * 1000000 * runtime.NumCPU())
	fmt.Printf("Normal: n_goroutines=%d, n_access=%d, average_time_per_access=%v\n", runtime.NumCPU(), runtime.NumCPU()*4*1000000, averageTotalTime)
}

func averageAccessTimeLazy() {
	lfuCache := clfu.NewLazyLFUCache(1000, 1000)
	for i := 0; i < 1000; i++ {
		lfuCache.Put(i, i, false)
	}

	// get the data 4M times and compute the latency
	totalTime := 0

	routine := func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < 4*1000000; i++ {
			st := time.Now()
			lfuCache.Get(i % 1000)
			et := time.Since(st)
			totalTime += int(et.Nanoseconds())
		}
	}

	wg := sync.WaitGroup{}
	// run a goroutine for each CPU
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go routine(&wg)
	}

	wg.Wait()

	averageTotalTime := totalTime / (4 * 1000000 * runtime.NumCPU())
	fmt.Printf("Lazy (with size 1000 as cache size): n_goroutines=%d, n_access=%d, average_time_per_access=%v\n", runtime.NumCPU(), runtime.NumCPU()*4*1000000, averageTotalTime)
}

func averageAccessTime() {
	fmt.Println("Checking average access time - lazy vs normal")

	averageAccessTimeNormal()

	averageAccessTimeLazy()
}

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

	// these functions are provided for benchmark purposes
	timeTaken()
	averageAccessTime()
}
