package main

import (
	"flag"
	"fmt"
	"github.com/ghjan/gcrontab/master/api"
	"github.com/ghjan/gcrontab/master/config"
	"github.com/ghjan/gcrontab/master/etcd"
	"github.com/ghjan/gcrontab/pkg/utils"
	"path/filepath"
	"runtime"
	"time"
)

var (
	confFile string //配置文件路径
)

//解析命令行参数
func initArgs() {
	// master -config ./master.json
	//master -h

	flag.StringVar(&confFile, "config", "master/config/master.json", "指定master.json")
	if confFile == "" {
		RootPath := utils.GetRootPath()
		confFile = filepath.Join(RootPath, "master/config/master.json")
	}

	flag.Parse()
}
func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	var (
		err error
	)
	//解析命令行参数
	initArgs()

	//初始化线程
	initEnv()

	//读取配置文件
	if err = config.InitConfig(confFile); err != nil {
		goto ERR
		return
	}

	//初始化任务管理器
	if err = etcd.InitJobMgr(); err != nil {
		goto ERR
	}
	//启动Api HTTP服务
	if err = api.InitApiServer(); err != nil {
		goto ERR
	}

	//正常退出
	for {
		time.Sleep(1 * time.Second)
	}
	return
	//异常退出
ERR:
	fmt.Println("master main error:", err)

}
