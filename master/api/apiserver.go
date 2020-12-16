package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ghjan/gcrontab/common"
	"github.com/ghjan/gcrontab/master/config"
	"github.com/ghjan/gcrontab/master/etcd"
	"net"
	"net/http"
	"strconv"
	"time"
)

//任务的HTTP接口
type ApiServer struct {
	httpServer *http.Server
}

var (
	//单例对象
	G_apiServer *ApiServer
)

//初始化服务
func InitApiServer() (err error) {
	var (
		mux           *http.ServeMux
		listener      net.Listener
		httpServer    *http.Server
		staticDir     http.Dir
		staticHandler http.Handler
	)
	//配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)
	mux.HandleFunc("/job/delete", handleJobDelete)
	mux.HandleFunc("/job/list", handleJobList)
	mux.HandleFunc("/job/kill", handleJobKill)

	//静态文件目录
	staticDir = http.Dir(config.G_config.Webroot)
	staticHandler = http.FileServer(staticDir)
	mux.Handle("/", http.StripPrefix("/", staticHandler)) //./webroot/index.html

	//启动TCP监听
	address := ":" + strconv.Itoa(config.G_config.ApiPort)
	if listener, err = net.Listen("tcp", address); err != nil {
		return
	}

	fmt.Printf("server listening in %s\n", address)

	//创建一个HTTP服务
	httpServer = &http.Server{
		ReadTimeout:  time.Duration(config.G_config.ApiReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(config.G_config.ApiWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}

	G_apiServer = &ApiServer{
		httpServer: httpServer,
	}

	//启动了服务端
	go httpServer.Serve(listener)

	return
}

//保存任务接口
// POST job={"name":"job1","command":"echo hello","cronExpr":"*/5 * * * *"}
func handleJobSave(resp http.ResponseWriter, req *http.Request) {
	var (
		err     error
		postJob string
		job     common.Job
		oldJob  *common.Job
		bytes   []byte
	)
	//1.解析POST表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	//2.取表单中的job字段
	postJob = req.PostForm.Get("job")
	//3.反序列化job
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		goto ERR
	}

	//4.任务保存到ETCD中
	if oldJob, err = etcd.G_jobMgr.SaveJob(&job); err != nil {
		goto ERR
	}

	//5.返回正常应答（{“errorno":0,"msg"L"","data":{...}})
	if bytes, err = common.BuildResponse(0, "success", oldJob); err != nil {
		goto ERR
	}
	resp.Write(bytes)

	//正常退出
	return

ERR:
	//6.异常返回
	fmt.Println("error:", err)
	if bytes, err = common.BuildResponse(-1, err.Error(), oldJob); err == nil {
		resp.Write(bytes)
	}
	return
}

//删除任务接口
// POST /job/delete name=job1
func handleJobDelete(resp http.ResponseWriter, req *http.Request) {
	var (
		err    error
		name   string
		oldJob *common.Job
		bytes  []byte
	)
	//1.解析POST表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	name = req.PostForm.Get("name")
	if name == "" {
		err = errors.New("parameter name is blank")
		goto ERR
	}
	//去删除任务
	if oldJob, err = etcd.G_jobMgr.DeleteJob(name); err != nil {
		goto ERR
	}

	//正常应答
	if bytes, err = common.BuildResponse(0, "success", oldJob); err == nil {
		resp.Write(bytes)
	}

	return
ERR:
	//6.异常返回
	fmt.Println("error:", err)
	if bytes, err = common.BuildResponse(-1, err.Error(), oldJob); err == nil {
		resp.Write(bytes)
	}
	return
}

//任务列表接口
//
func handleJobList(resp http.ResponseWriter, req *http.Request) {
	var (
		jobList []*common.Job
		err     error
		bytes   []byte
	)
	if jobList, err = etcd.G_jobMgr.ListJobs(); err != nil {
		goto ERR
	}
	if bytes, err = common.BuildResponse(0, "success", jobList); err == nil {
		resp.Write(bytes)
	}

	return
ERR:
	//6.异常返回
	fmt.Println("error:", err)
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
	return
}

//中断/杀死任务接口
// POST /job/kill name=job1
func handleJobKill(resp http.ResponseWriter, req *http.Request) {
	var (
		name  string
		err   error
		bytes []byte
	)
	//1.解析POST表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	name = req.PostForm.Get("name")
	if name == "" {
		err = errors.New("parameter name is blank")
		goto ERR
	}
	if err = etcd.G_jobMgr.KillJob(name); err != nil {
		goto ERR
	}
	if bytes, err = common.BuildResponse(0, "success", nil); err == nil {
		resp.Write(bytes)
	}

	return
ERR:
	//6.异常返回
	fmt.Println("error:", err)
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
	return
}
