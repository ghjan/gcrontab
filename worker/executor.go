package worker

import (
	"github.com/ghjan/gcrontab/common"
	"math/rand"
	"os/exec"
	"time"
)

//任务执行器
type Executor struct {
}

var (
	G_executor *Executor
)

//执行一个任务
func (executor Executor) ExecuteJob(info *common.JobExecuteInfo) {
	//执行任务放到一个协程中
	go func() {
		var (
			cmd     *exec.Cmd
			err     error
			output  []byte
			result  *common.JobExecuteResult
			jobLock *JobLock
		)
		//任务结果
		result = &common.JobExecuteResult{
			ExecuteInfo: info,
			Output:      make([]byte, 0),
		}

		//首先创建分布式锁
		jobLock = G_jobMgr.CreateJobLock(info.Job.Name)

		//记录任务开始时间
		result.StartTime = time.Now()

		//抢锁
		//随机睡眠(0~1s)
		rand.Seed(time.Now().Unix())
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

		err = jobLock.TryLock()
		//最后协程结束前一定释放锁
		defer jobLock.Unlock()

		if err != nil { //抢锁失败
			result.Err = err
			result.EndTime = time.Now()
		} else {
			//执行shell命令
			cmd = exec.CommandContext(info.CancelCtx, "bash", "-c", info.Job.Command)
			output, err = cmd.CombinedOutput()
			result.Output = output
			result.Err = err
			result.EndTime = time.Now()
		}
		//任务执行完成后，把执行的结果返回给scheduler，scheduler会从excecutingTable中删除掉执行记录
		G_scheduler.PushJobResult(result)
	}()
	return
}

//初始化执行器
func InitExecutor() (err error) {
	G_executor = &Executor{}
	return
}
