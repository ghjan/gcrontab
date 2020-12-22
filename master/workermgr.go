package master

import (
	"context"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/ghjan/gcrontab/common"
	"github.com/ghjan/gcrontab/master/config"
	"time"
)

// /cron/workers/

type WorkerMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

var (
	G_workerMgr *WorkerMgr
)

func (workerMgr *WorkerMgr) ListWorkers() (workerArr []string, err error) {

	var (
		getResp  *clientv3.GetResponse
		kv       *mvccpb.KeyValue
		workerIP string
	)
	//初始化数组
	workerArr = make([]string, 0)
	//获取目录下所有kv
	if getResp, err = workerMgr.kv.Get(context.TODO(), common.JOB_WORKERS_DIR, clientv3.WithPrefix()); err != nil {
		return
	}
	//解析每个节点的IP
	for _, kv = range getResp.Kvs {
		//kv.key:/cron/workers/192.168.2.1
		workerIP = common.ExtractWorkerIP(string(kv.Key))
		workerArr = append(workerArr, workerIP)
	}
	return
}
func InitWorkerMgr() (err error) {
	var (
		cfg    clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
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

	G_workerMgr = &WorkerMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}
	return
}
