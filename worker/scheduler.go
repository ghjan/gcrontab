package worker

import (
	"fmt"
	"github.com/ghjan/gcrontab/common"
	"time"
)

type Scheduler struct {
	jobEventChan      chan *common.JobEvent              //etcd任务事件队列
	jobPlanTable      map[string]*common.JobSchedulePlan //任务调度计划表
	jobExecutionTable map[string]*common.JobExecuteInfo  //任务执行表
	JobResultChan     chan *common.JobExecuteResult
}

var (
	G_scheduler *Scheduler
)

//尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobPlan *common.JobSchedulePlan) {
	//调度和执行是两件事情
	var (
		jobExecuteInfo *common.JobExecuteInfo
		jobExecuting   bool
	)

	//执行的任务可能运行很久，1分钟会调度60次，但是只能执行1次 防止并发

	//如果任务正在执行，跳过本次调度
	if jobExecuteInfo, jobExecuting = scheduler.jobExecutionTable[jobPlan.Job.Name]; jobExecuting {
		fmt.Println("尚未退出，跳过执行:", jobPlan.Job.Name)
		return
	}
	//构建执行状态信息
	jobExecuteInfo = common.BuildJobExecuteInfo(jobPlan)

	//保存执行状态
	scheduler.jobExecutionTable[jobPlan.Job.Name] = jobExecuteInfo

	//执行任务
	fmt.Println("执行任务:", jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
	G_executor.ExecuteJob(jobExecuteInfo)
	return
}

//重新计算任务调度状态
func (scheduler *Scheduler) TrySchedule() (scheduleAfter time.Duration) {
	var (
		jobPlan  *common.JobSchedulePlan
		now      time.Time
		nearTime *time.Time
	)

	//如果任务表为空，睡眠一会儿
	if len(scheduler.jobPlanTable) == 0 {
		scheduleAfter = 1 * time.Second
		return
	}
	now = time.Now()

	//1.遍历所有任务
	for _, jobPlan = range scheduler.jobPlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			//过期的任务立即执行
			fmt.Printf("过期的任务立即执行, 执行任务：%s\n", jobPlan.Job.Name)
			scheduler.TryStartJob(jobPlan)
			jobPlan.NextTime = jobPlan.Expr.Next(now) //更新下次执行时间
		}
		//统计最近一个要过期的任务时间
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}

	//下次调度时间
	scheduleAfter = (*nearTime).Sub(now)

	//3.统计最近的要过期的任务的事件(N秒后过期 ==scheduleAfter)

	return
}

//调度协程
func (scheduler Scheduler) scheduleLoop() {
	var (
		jobEvent      *common.JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		jobResult     *common.JobExecuteResult
	)
	//初始化一次（第一次是1秒)
	scheduleAfter = scheduler.TrySchedule()

	//调度的延迟定时器
	scheduleTimer = time.NewTimer(scheduleAfter)

	//定时任务 common.Job
	for {
		select {
		case jobEvent = <-scheduler.jobEventChan: //监听任务变化事件
			//对内存中维护的任务列表进行增删改查
			scheduler.handleJobEvent(jobEvent)
		case <-scheduleTimer.C: //最近的任务到期了
		case jobResult = <-scheduler.JobResultChan: //监听任务执行结果
			scheduler.handleJobResult(jobResult)
		}
		//调度一次任务
		scheduleAfter = scheduler.TrySchedule()
		//重置调度间隔
		scheduleTimer.Reset(scheduleAfter)
	}
	return
}

//处理任务事件
func (scheduler Scheduler) handleJobEvent(jobEvent *common.JobEvent) {
	var (
		jobSchedulerPlan *common.JobSchedulePlan
		jobExisted       bool
		err              error
		jobExecuteInfo   *common.JobExecuteInfo
		jobExecuting     bool
	)
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE: //保存任务事件
		if jobSchedulerPlan, err = common.BuildJobSchedulerPlan(jobEvent.Job); err != nil {
			return
		}
		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulerPlan
	case common.JOB_EVENT_DELETE: //删除任务事件
		if jobSchedulerPlan, jobExisted = scheduler.jobPlanTable[jobEvent.Job.Name]; jobExisted {
			delete(scheduler.jobPlanTable, jobEvent.Job.Name)
		}
	case common.JOB_EVENT_KILL: //强杀任务
		//取消掉command执行,判断任务是否在执行中
		if jobExecuteInfo, jobExecuting = scheduler.jobExecutionTable[jobEvent.Job.Name]; jobExecuting {
			jobExecuteInfo.CancelFunc() //触发command杀死shell子进程，任务得到退出
		}

	}
}

func (scheduler *Scheduler) handleJobResult(jobResult *common.JobExecuteResult) {
	//删除执行状态
	delete(scheduler.jobExecutionTable, jobResult.ExecuteInfo.Job.Name)
	fmt.Printf("任务执行完成:%s,output:%s", jobResult.ExecuteInfo.Job.Name, string(jobResult.Output))
	if jobResult.Err != nil {
		fmt.Printf(",error:%s", jobResult.Err.Error())
	}
	fmt.Println()
}

//推送任务变化事件
func (scheduler *Scheduler) PushJobEvent(jobEvent *common.JobEvent) {
	scheduler.jobEventChan <- jobEvent
}

//回传任务执行结果
func (scheduler *Scheduler) PushJobResult(jobResult *common.JobExecuteResult) {
	scheduler.JobResultChan <- jobResult
}

//初始化调度器
func InitScheduler() (err error) {
	G_scheduler = &Scheduler{
		jobEventChan:      make(chan *common.JobEvent, 1000),
		jobPlanTable:      make(map[string]*common.JobSchedulePlan),
		jobExecutionTable: make(map[string]*common.JobExecuteInfo),
		JobResultChan:     make(chan *common.JobExecuteResult, 1000),
	}
	//启动调度协程
	go G_scheduler.scheduleLoop()
	return
}
