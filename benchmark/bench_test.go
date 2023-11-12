package benchmark

import (
	"github.com/stretchr/testify/assert"
	"hades"
	"hades/lib/utils"
	"hades/settings"
	"math/rand"
	"testing"
	"time"
)

// go test -bench=.  -benchtime=5s

var db *hades.DB

func init() {
	// 初始化用于基准测试的存储引擎
	settings.Init("../hades.yaml")
	opts := settings.Conf.DBConfig
	//dir, _ := os.MkdirTemp("", "bitcask-go-bench")
	//opts.DirPath = dir

	var err error
	db, err = hades.Open(opts)
	if err != nil {
		panic(err)
	}
}

func Benchmark_Put(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := db.Put(utils.GetTestKey(i), utils.RandomValue(1024))
		assert.Nil(b, err)
	}
}

func Benchmark_Get(b *testing.B) {
	for i := 0; i < 10000; i++ {
		err := db.Put(utils.GetTestKey(i), utils.RandomValue(1024))
		assert.Nil(b, err)
	}

	rand.Seed(time.Now().UnixNano())
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := db.Get(utils.GetTestKey(rand.Int()))
		if err != nil && err != hades.ErrKeyNotFound {
			b.Fatal(err)
		}
	}
}

func Benchmark_Delete(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	rand.Seed(time.Now().UnixNano())
	for i := 0; i < b.N; i++ {
		err := db.Delete(utils.GetTestKey(rand.Int()))
		assert.Nil(b, err)
	}
}
