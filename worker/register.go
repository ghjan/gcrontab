package worker

import (
	"context"
	"github.com/coreos/etcd/clientv3"
	"github.com/ghjan/gcrontab/common"
	"github.com/ghjan/gcrontab/worker/config"
	"net"
	"time"
)

//注册节点到etcd: /cron/workers/IP地址
type Register struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	localIP string //本机IP
}

var (
	G_register *Register
)

func getLocalIP() (ipv4 string, err error) {
	var (
		addrs   []net.Addr
		addr    net.Addr
		ipNet   *net.IPNet
		isIpNet bool
	)
	//获取所有网卡
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}
	//获取第一个非io的网卡IP
	for _, addr = range addrs {
		//ipv4 ipv6
		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			//跳过IPV6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String()
				return
			}
		}
	}
	err = common.ERR_NO_LOCAL_IP_FOUND
	return
}

//注册到 /cron/workers/IP,并且自动续租
func (register *Register) keepOnline() {
	var (
		regKey         string
		leaseGrantResp *clientv3.LeaseGrantResponse
		err            error
		keepAliveChan  <-chan *clientv3.LeaseKeepAliveResponse
		keepAliveResp  *clientv3.LeaseKeepAliveResponse
		//putResp        *clientv3.PutResponse
		cancelCtx      context.Context
		cancelFunction context.CancelFunc
	)
	for {
		//注册路径
		regKey = common.JOB_WORKERS_DIR + register.localIP
		cancelFunction = nil
		//创建租约
		if leaseGrantResp, err = register.lease.Grant(context.TODO(), 10); err != nil {
			goto RETRY
		}
		//自动续租
		if keepAliveChan, err = register.lease.KeepAlive(context.TODO(), leaseGrantResp.ID); err != nil {
			goto RETRY
		}

		cancelCtx, cancelFunction = context.WithCancel(context.TODO())
		//注册到etcd
		if _, err = register.kv.Put(cancelCtx, regKey, "", clientv3.WithLease(leaseGrantResp.ID)); err != nil {
			goto RETRY
		}
		//处理续租应答
		for {
			select {
			case keepAliveResp = <-keepAliveChan:
				if keepAliveResp == nil { //续租失败
					goto RETRY

				}
			}
		}

	RETRY:
		time.Sleep(1 * time.Second)
		if cancelFunction != nil { //取消续租
			cancelFunction()
		}
	}

}

func InitRegister() (err error) {
	var (
		cfg     clientv3.Config
		localIP string
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

	if localIP, err = getLocalIP(); err == nil {
		G_register = &Register{
			client:  client,
			kv:      kv,
			lease:   lease,
			localIP: localIP,
		}
	}
	go G_register.keepOnline()

	return
}
