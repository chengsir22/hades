package settings

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type AppConfig struct {
	Name       string `mapstructure:"name"`
	Mode       string `mapstructure:"mode"`
	Version    string `mapstructure:"version"`
	Bind       string `mapstructure:"bind"`
	Port       int    `mapstructure:"port"`
	MaxClients int    `mapstructure:"maxClients"`
	*DBConfig  `mapstructure:"db"`
	*LogConfig `mapstructure:"log"`
}

type DBConfig struct {
	DirPath      string      `mapstructure:"dirPath"`      // 数据库数据目录
	DataFileSize int64       `mapstructure:"dataFileSize"` // 数据文件大小
	SyncWrites   bool        `mapstructure:"syncWrites"`   // 是否同步写入
	IndexType    IndexerType `mapstructure:"indexType"`    // 索引类型
}

type IndexerType = int8

const (
	BTree IndexerType = iota + 1 // BTree 索引
	ART                          // ART Adpative Radix Tree 自适应基数树索引
)

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
