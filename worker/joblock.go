package worker

import (
	"context"
	"github.com/coreos/etcd/clientv3"
	"github.com/ghjan/gcrontab/common"
)

//分布式锁(TXN事务)
type JobLock struct {
	//etcd客户端
	kv         clientv3.KV
	lease      clientv3.Lease
	jobName    string             //任务名
	CancelFunc context.CancelFunc //用于终止租约
	leaseId    clientv3.LeaseID
	isLocked   bool
}

//初始化一把锁
func InitJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease) (jobLock *JobLock) {
	jobLock = &JobLock{
		kv:      kv,
		lease:   lease,
		jobName: jobName,
	}
	return
}
func (jobLock *JobLock) TryLock() (err error) {
	var (
		leaseGrantResp    *clientv3.LeaseGrantResponse
		cancelCtx         context.Context
		cancelFunc        context.CancelFunc
		leaseId           clientv3.LeaseID
		keepAliveRespChan <-chan *clientv3.LeaseKeepAliveResponse
		txn               clientv3.Txn
		lockKey           string
		txnResp           *clientv3.TxnResponse
	)
	//1.创建租约(5秒)
	if leaseGrantResp, err = jobLock.lease.Grant(context.TODO(), 5); err != nil {
		goto FAIL
	}

	//context用于取消自动续租
	cancelCtx, cancelFunc = context.WithCancel(context.TODO())

	//租约ID
	leaseId = leaseGrantResp.ID
	//2.自动续租
	if keepAliveRespChan, err = jobLock.lease.KeepAlive(cancelCtx, leaseId); err != nil {
		goto FAIL
	}

	//3.处理续租应当的协程
	go func() {
		var (
			keepResp *clientv3.LeaseKeepAliveResponse
		)
		for {
			select {
			case keepResp = <-keepAliveRespChan: //自动续租的应答
				if keepResp == nil {
					goto END
				}
			}
		}
	END:
	}()
	//4.创建事务txn
	txn = jobLock.kv.Txn(context.TODO())
	//锁路径
	lockKey = common.JOB_LOCK_DIR + jobLock.jobName
	//5.事务抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseId))).
		Else(clientv3.OpGet(lockKey))
	//提交事务
	if txnResp, err = txn.Commit(); err != nil {
		goto FAIL
	}

	//6.成功返回，失败释放租约
	if !txnResp.Succeeded { //锁被占用
		err = common.ERR_LOCK_ALREADY_REQUIRED
		goto FAIL
	}
	//抢锁成功
	jobLock.isLocked = true
	jobLock.leaseId = leaseId
	jobLock.CancelFunc = cancelFunc
	return

FAIL:
	if cancelFunc != nil {
		cancelFunc() //取消自动租约
	}
	if jobLock != nil {
		jobLock.lease.Revoke(context.TODO(), leaseId)
	}
	return
}

func (jobLock *JobLock) Unlock() {
	if jobLock.isLocked {
		jobLock.CancelFunc()                                  //取消程序自动续租的协程
		jobLock.lease.Revoke(context.TODO(), jobLock.leaseId) //释放租约
		jobLock.isLocked = false
	}
	return
}
