package benchmark

import (
	bitcaskgo "bitcask-go"
	"bitcask-go/utils"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var db *bitcaskgo.DB

func init() {
	var err error
	options := bitcaskgo.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-bench")
	options.DirPath = dir
	options.IndexType = bitcaskgo.BTree
	options.MMapIsOpen = false
	db, err = bitcaskgo.Open(options)
	if err != nil {
		return
	}
}
func Benchmark_Put(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := db.Put(utils.GetTestKey(i), utils.RandomValue(128))
		assert.Nil(b, err)
	}
}
func Benchmark_Get(b *testing.B) {
	for i := 0; i < 100000; i++ {
		err := db.Put(utils.GetTestKey(i), utils.RandomValue(128))
		assert.Nil(b, err)
	}
	rand.New(rand.NewSource(time.Now().UnixNano()))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := db.Get(utils.GetTestKey(rand.Int()))
		if err != nil && err != bitcaskgo.ErrKeyIsUnfound {
			b.Fatal(err)
		}
	}
}
func Benchmark_Delete(b *testing.B) {
	for i := 0; i < 100000; i++ {
		err := db.Put(utils.GetTestKey(i), utils.RandomValue(128))
		assert.Nil(b, err)
	}
	rand.New(rand.NewSource(time.Now().UnixNano()))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := db.Delete(utils.GetTestKey(rand.Int()))
		if err != nil && err != bitcaskgo.ErrKeyIsUnfound {
			b.Fatal(err)
		}
	}
}

// 压测 10 秒，PUT 和 GET 并行
func Benchmark_ConcurrentPutGet(b *testing.B) {
	duration := 10 * time.Second
	numWorkers := 8 // 4 PUT + 4 GET

	var wg sync.WaitGroup
	stopCh := make(chan struct{})
	var ops int64 // 原子计数总操作数
	// PUT workers
	wg.Add(numWorkers / 2)
	for w := 0; w < numWorkers/2; w++ {
		go func(workerID int) {
			defer wg.Done()
			i := workerID * 100000
			for {
				select {
				case <-stopCh:
					return
				default:
					key := utils.GetTestKey(i)
					value := utils.RandomValue(128)
					if err := db.Put(key, value); err != nil {
						b.Fatal(err)
					}
					atomic.AddInt64(&ops, 1)
					i++
				}
			}
		}(w)
	}

	// GET workers
	wg.Add(numWorkers / 2)
	for w := 0; w < numWorkers/2; w++ {
		go func(workerID int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))
			for {
				select {
				case <-stopCh:
					return
				default:
					key := utils.GetTestKey(r.Intn(1000000))
					_, err1 := db.Get(key)
					if err1 != nil && err1 != bitcaskgo.ErrKeyIsUnfound {
						b.Fatal(err1)
					}
					atomic.AddInt64(&ops, 1)
				}
			}
		}(w)
	}

	start := time.Now()
	time.Sleep(duration)
	close(stopCh)
	wg.Wait()

	elapsed := time.Since(start)
	fmt.Printf("Total operations: %d\n", ops)
	fmt.Printf("Elapsed time: %v\n", elapsed)
	fmt.Printf("Throughput: %.2f ops/sec\n", float64(ops)/elapsed.Seconds())
}
