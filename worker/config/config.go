package config

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	EtcdEndpoints         []string `json:"etcdEndpoints"`
	EtcdDialTimeeout      int      `json:"etcdDialTimeeout"`
	MongodbUri            string   `json:"mongodbUri"`
	MongodbConnectTimeout int      `json:"mongodbConnectTimeout"`
	JobLogBatchSize       int      `json:"jobLogBatchSize"`
	JobLogCommitTimeout   int      `json:"jobLogCommitTimeout"`
}

var (
	//单例
	G_config Config
)

func InitConfig(filename string) (err error) {
	var (
		content []byte
		cfg     Config
	)
	//1.把配置读进来
	if content, err = ioutil.ReadFile(filename); err != nil {
		return
	}
	//2.json反序列化
	if err = json.Unmarshal(content, &cfg); err != nil {
		return
	}
	G_config = cfg
	return
}
