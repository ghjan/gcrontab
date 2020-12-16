package common

import "encoding/json"

//定时任务
type Job struct {
	Name     string `json:"name"`     //任务名
	Command  string `json:"command"`  //shell命令
	CronExpr string `json:"cronExpr"` //cron表达式
}

type Response struct {
	Errno int         `josn:"errno"`
	Msg   string      `josn:"msg"`
	Data  interface{} `josn:"data"`
}

func BuildResponse(errno int, msg string, data interface{}) (resp []byte, err error) {
	var (
		response Response
	)
	response.Errno = errno
	response.Msg = msg
	response.Data = data
	resp,err= json.Marshal(response)
	return
}
