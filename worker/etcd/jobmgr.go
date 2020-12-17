package etcd

import (
	"context"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/ghjan/gcrontab/common"
	"github.com/ghjan/gcrontab/worker/config"
	"github.com/ghjan/gcrontab/worker/scheduler"
	"time"
)

//任务管理器
type JobMgr struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
}

var (
	//单例
	G_jobMgr *JobMgr

	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
)

//初始化管理器
func InitJobMgr() (err error) {
	var (
		cfg clientv3.Config
	)
	//初始化配置
	cfg = clientv3.Config{
		Endpoints:   config.G_config.EtcdEndpoints,                                      //集群地址
		DialTimeout: time.Duration(config.G_config.EtcdDialTimeeout) * time.Millisecond, //连接超时
	}

	//建立连接
	if client, err = clientv3.New(cfg); err != nil {
		return
	}

	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	watcher = clientv3.NewWatcher(client)

	G_jobMgr = &JobMgr{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}
	//创建一个watcher

	err = G_jobMgr.WatchJobs()

	return
}

//监听任务变化
func (jobMgr *JobMgr) WatchJobs() (err error) {
	var (
		job                *common.Job
		getResp            *clientv3.GetResponse
		kvpair             *mvccpb.KeyValue
		watchStartRevision int64
		watchChan          clientv3.WatchChan
		watchResp          clientv3.WatchResponse
		watchEvent         *clientv3.Event
		jobName            string
		jobEvent           *common.JobEvent
	)

	//1.get /cron/jobs/ 目录下的所有任务，并且获知当前集群的revision
	if getResp, err = jobMgr.kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix()); err != nil {
		return
	}

	//遍历当前的任务列表
	for _, kvpair = range getResp.Kvs {
		//反序列化json得到job
		if job, err = common.UnpackJob(kvpair.Value); err == nil {
			jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
			//吧这个job同步给scheduler（调度协程）
			fmt.Println(*jobEvent)
			scheduler.G_scheduler.PushJobEvent(jobEvent)
		} else {
			err = nil
		}
	}

	//2.从该revision向后监听变化事件
	go func() { //监听协程
		//从GET时刻的后续版本开始监听变化
		watchStartRevision = getResp.Header.Revision + 1
		//监听 /cron/jobs/ 目录的后续变化
		watchChan = jobMgr.watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithRev(watchStartRevision),
			clientv3.WithPrefix())
		//处理监听事件
		for watchResp = range watchChan {
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: //任务保存事件
					//反序列化job，
					if job, err = common.UnpackJob(watchEvent.Kv.Value); err != nil {
						continue
					}
					//构建一个Event事件
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
				case mvccpb.DELETE: //任务删除事件
					jobName = common.ExtractJobName(string(watchEvent.Kv.Key))

					job = &common.Job{
						Name: jobName,
					}
					//构建一个Event事件
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE, job)
				}
				//推一个事件给scheduler
				if jobEvent!=nil{
					fmt.Println(*jobEvent)
					scheduler.G_scheduler.PushJobEvent(jobEvent)
				}
			}
		}

	}()

	return
}
