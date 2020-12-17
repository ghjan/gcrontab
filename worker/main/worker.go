package main

import (
	"flag"
	"fmt"
	"github.com/ghjan/gcrontab/pkg/utils"
	"github.com/ghjan/gcrontab/worker"
	"github.com/ghjan/gcrontab/worker/config"
	"path/filepath"
	"runtime"
	"time"
)

var (
	confFile string //配置文件路径
)

//解析命令行参数
func initArgs() {
	// worker -config ./worker.json
	//worker -h

	flag.StringVar(&confFile, "config", "worker/config/worker.json", "指定worker.json")
	if confFile == "" {
		RootPath := utils.GetRootPath()
		confFile = filepath.Join(RootPath, "worker/config/worker.json")
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
	//加载执行器
	if err = worker.InitExecutor(); err != nil {
		goto ERR
	}

	//启动调度器
	if err = worker.InitScheduler(); err != nil {
		goto ERR
	}
	//初始化任务管理器
	if err = worker.InitJobMgr(); err != nil {
		goto ERR
	}

	//正常退出
	for {
		time.Sleep(1 * time.Second)
	}
	return
	//异常退出
ERR:
	fmt.Println("worker main error:", err)

}
