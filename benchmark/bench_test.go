package benchmark

import (
	bitcaskgo "bitcask-go"
	"bitcask-go/utils"
	"math/rand"
	"os"
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
	options.IndexType = bitcaskgo.ART
	options.MMapIsOpen = true
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
