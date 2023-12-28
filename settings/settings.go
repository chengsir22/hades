package settings

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type AppConfig struct {
	Name              string `mapstructure:"name"`
	Mode              string `mapstructure:"mode"`
	Version           string `mapstructure:"version"`
	Bind              string `mapstructure:"bind"`
	Port              int    `mapstructure:"port"`
	MaxClients        int    `mapstructure:"maxClients"`
	*DBConfig         `mapstructure:"db"`
	*IteratorConfig   `mapstructure:"iterator"`
	*WriteBatchConfig `mapstructure:"writeBatch"`
	*LogConfig        `mapstructure:"log"`
}

type DBConfig struct {
	DirPath            string      `mapstructure:"dirPath"`            // 数据库数据目录
	DataFileSize       int64       `mapstructure:"dataFileSize"`       // 数据文件大小
	SyncWrites         bool        `mapstructure:"syncWrites"`         // 是否同步写入
	BytesPerSync       uint        `mapstructure:"bytesPerSync"`       // 每写入多少字节进行一次 sync 操作
	IndexType          IndexerType `mapstructure:"indexType"`          // 索引类型
	MMapAtStartup      bool        `mapstructure:"mmapAtStartup"`      // 启动时是否使用 MMap 加载数据
	DataFileMergeRatio float32     `mapstructure:"dataFileMergeRatio"` //	数据文件合并的阈值
}

type IndexerType = int8

const (
	BTree     IndexerType = iota + 1 // BTree 索引
	ART                              // ART Adpative Radix Tree 自适应基数树索引
	BPlusTree                        // BPlusTree B+ 树索引，将索引存储到磁盘上
	Skiplist
)

// IteratorConfig 索引迭代器配置项
type IteratorConfig struct {
	// 遍历前缀为指定值的 Key，默认为空
	Prefix []byte
	// 是否反向遍历，默认 false 是正向
	Reverse bool `mapstructure:"reverse"`
}

// WriteBatchConfig 批量写配置项
type WriteBatchConfig struct {
	MaxBatchNum uint `mapstructure:"maxBatchNum"` // 一个批次当中最大的数据量10000
	SyncWrites  bool `mapstructure:"syncWrites"`  // 提交时是否 sync 持久化 true
}

// LogConfig  stores config for logger
type LogConfig struct {
	Path       string `mapstructure:"path"`
	Name       string `mapstructure:"name"`
	Ext        string `mapstructure:"ext"`
	TimeFormat string `mapstructure:"timeFormat"`
}

var Conf = new(AppConfig)

func Init(filepath string) (err error) {
	viper.SetConfigFile(filepath)
	//viper.SetConfigFile("hades.yaml")
	//viper.AddConfigPath("../")
	err = viper.ReadInConfig() // 读取配置信息
	if err != nil {            // 读取配置信息失败
		fmt.Printf("Fatal viper.ReadInConfig() failed, err: %s \n", err)
		return
	}

	// 把读取到的配置信息反序列化到Conf变量中
	if err = viper.Unmarshal(Conf); err != nil {
		fmt.Printf("viper.Unmarshal failed, err:%v\n", err)
		return
	}

	// 监控配置文件变化
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("配置文件修改了...")
		if err := viper.Unmarshal(Conf); err != nil {
			fmt.Printf("viper.Unmarshal failed, err:%v\n", err)
		}
	})
	return
}
