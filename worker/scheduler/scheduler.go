package scheduler

import (
	"fmt"
	"github.com/ghjan/gcrontab/common"
	"time"
)

type Scheduler struct {
	jobEventChan chan *common.JobEvent              //etcd任务事件队列
	jobPlanTable map[string]*common.JobSchedulePlan //任务调度计划表
}

var (
	G_scheduler *Scheduler
)

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
			//TODO:过期的任务立即执行
			fmt.Printf("过期的任务立即执行, 执行任务：%s\n", jobPlan.Job.Name)
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
	)
	//初始化一次（第一次是1秒)
	scheduleAfter = scheduler.TrySchedule()

	//调度的延迟定时器
	scheduleTimer = time.NewTimer(scheduleAfter)

	//定时任务 common.Job
	for {
		select {
		case jobEvent=<-scheduler.jobEventChan: //监听任务变化事件
			//对内存中维护的任务列表进行增删改查
			scheduler.handleJobEvent(jobEvent)
		case <-scheduleTimer.C: //最近的任务到期了
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
	}
}

//推送任务变化事件
func (scheduler Scheduler) PushJobEvent(jobEvent *common.JobEvent) {
	scheduler.jobEventChan <- jobEvent
}

//初始化调度器
func InitScheduler() (err error) {
	G_scheduler = &Scheduler{
		jobEventChan: make(chan *common.JobEvent, 1000),
		jobPlanTable: make(map[string]*common.JobSchedulePlan),
	}
	//启动调度协程
	go G_scheduler.scheduleLoop()
	return
}
