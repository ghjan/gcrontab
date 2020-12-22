package common

import (
	"context"
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"strings"
	"time"
)

//定时任务
type Job struct {
	Name     string `json:"name"`     //任务名
	Command  string `json:"command"`  //shell命令
	CronExpr string `json:"cronExpr"` //cron表达式
}

//任务事件
type JobEvent struct {
	EventType int //任务类型 SAVE,DELETE
	Job       *Job
}

//任务执行状态
type JobExecuteInfo struct {
	Job        *Job               //任务信息
	PlanTime   time.Time          //理论上的调度时间
	RealTime   time.Time          //实际的调度时间
	CancelCtx  context.Context    //任务command的上下文
	CancelFunc context.CancelFunc //用于取消command执行的cancel函数
}

//任务调度计划
type JobSchedulePlan struct {
	Job      *Job                 //要调度的任务信息
	Expr     *cronexpr.Expression //解析好的cronexpr表达式
	NextTime time.Time            //下次调度时间
}

type Response struct {
	Errno int         `josn:"errno"`
	Msg   string      `josn:"msg"`
	Data  interface{} `josn:"data"`
}

//任务执行结果
type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo //执行状态
	Output      []byte          //脚本输出
	Err         error           //脚本错误信息
	StartTime   time.Time       //启动时间
	EndTime     time.Time       //结束时间
}

//任务执行日志
type JobLog struct {
	JobName      string `json:"jobName";bson:"jobName"`           //任务名
	Command      string `json:"command";bson:"command"`           //shell命令
	Err          string `json:"err";bson:"err"`                   //错误原因
	Output       string `json:"output";bson:"output"`             //脚本执行输出
	PlanTime     int64  `json:"planTime";bson:"planTime"`         //计划开始时间
	ScheduleTime int64  `json:"scheduleTime";bson:"scheduleTime"` //实际调度时间
	StartTime    int64  `json:"startTime";bson:"startTime"`       //任务执行开始时间
	EndTime      int64  `json:"endTime";bson:"endTime"`           //任务执行结束时间
}

//日志批次
type LogBatch struct {
	Logs []interface{} //多条日志
}

//任务日志过滤条件
type JobLogFilter struct {
	JobName string `bson:"jobName"` //任务名
}

//任务日志排序规则
type SortLogByStartTime struct {
	SortOrder int64 `bson:"startTime"` //startTime:1正序
}

func BuildResponse(errno int, msg string, data interface{}) (resp []byte, err error) {
	var (
		response Response
	)
	response.Errno = errno
	response.Msg = msg
	response.Data = data
	resp, err = json.Marshal(response)
	return
}

//反序列化job
func UnpackJob(value []byte) (ret *Job, err error) {
	var (
		job Job
	)

	job = Job{}
	if err = json.Unmarshal(value, &job); err != nil {
		return
	}
	ret = &job
	return
}

//从etcd的key中提取任务名
func ExtractJobName(jobKey string) (string) {
	return strings.TrimPrefix(jobKey, JOB_SAVE_DIR)
}

//从etcd的key中提取杀死任务名
func ExtractKillerName(kilertKey string) (string) {
	return strings.TrimPrefix(kilertKey, JOB_KILLER_DIR)
}

func BuildJobEvent(eventType int, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}

func BuildJobSchedulerPlan(job *Job) (jobSchedulePlan *JobSchedulePlan, err error) {
	var (
		expr *cronexpr.Expression
	)
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return
	}

	//生成任务调度计划对象
	jobSchedulePlan = &JobSchedulePlan{
		Job:      job,
		Expr:     expr,
		NextTime: expr.Next(time.Now()),
	}
	return
}

func BuildJobExecuteInfo(jobSchedulePlan *JobSchedulePlan) (jobExecuteInfo *JobExecuteInfo) {
	jobExecuteInfo = &JobExecuteInfo{
		Job:      jobSchedulePlan.Job,
		PlanTime: jobSchedulePlan.NextTime, //计划调度时间
		RealTime: time.Now(),               //实际调度时间
	}
	jobExecuteInfo.CancelCtx, jobExecuteInfo.CancelFunc = context.WithCancel(context.TODO())

	return
}

//从etcd的key中提取worker的IP
func ExtractWorkerIP(key string) (string) {
	return strings.TrimPrefix(key, JOB_WORKERS_DIR)
}
