package worker

import (
	"context"
	"fmt"
	"github.com/ghjan/gcrontab/common"
	"github.com/ghjan/gcrontab/worker/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

//mongodb存储日志
type LogSink struct {
	client          *mongo.Client
	logCollection   *mongo.Collection
	logChan         chan *common.JobLog
	autoCommmitChan chan *common.LogBatch
}

var (
	//单例
	G_logSink *LogSink
)

func InitLogSink() (err error) {
	var (
		client *mongo.Client
	)
	if G_logSink == nil || G_logSink.client == nil {
		// Set G_logSink options
		clientOptions1 := options.Client().ApplyURI(config.G_config.MongodbUri)
		clientOptions2 := options.Client().SetConnectTimeout(time.Duration(config.G_config.MongodbConnectTimeout) *
			time.Second)
		// Connect to MongoDB
		if client, err = mongo.Connect(context.TODO(), clientOptions1, clientOptions2); err != nil {
			log.Fatalf("initMongoClient, mongo.Connect error:%s\n", err.Error())
			return
		}

		// Check the connection
		if err = client.Ping(context.TODO(), nil); err != nil {
			log.Fatalf("initMongoClient, G_logSink.Ping error:%s\n", err.Error())
			return
		}

		G_logSink = &LogSink{
			client:          client,
			logCollection:   client.Database("cron").Collection("log"),
			logChan:         make(chan *common.JobLog, 1000),
			autoCommmitChan: make(chan *common.LogBatch, 1000),
		}
		//启动一个存储协程
		fmt.Println("InitLogSink, Connected to MongoDB for cron log!")
		go G_logSink.writeLoop()
		return
	}
	return

}

//批量写入日志
func (logSink *LogSink) saveLogs(batch *common.LogBatch) {
	logSink.logCollection.InsertMany(context.TODO(), batch.Logs)
}

func (logSink *LogSink) writeLoop() {
	var (
		log          *common.JobLog
		logBatch     *common.LogBatch
		commitTimer  *time.Timer
		timeoutBatch *common.LogBatch
	)

	for {
		select {
		case log = <-logSink.logChan:
			//TODO:把这条log写到mongodb中
			//logsink.LogCollection.InsertOne
			//每次插入需要等待mongdb的一次请求往返，耗时可能因为网络慢花费比较长的时间
			if logBatch == nil {
				logBatch = &common.LogBatch{}
				//让这个批次超时自动提交
				commitTimer = time.AfterFunc(time.Duration(config.G_config.JobLogCommitTimeout)*time.Millisecond,
					func(batch *common.LogBatch) func() {
						return func() {
							//发送通知-串行处理 不要直接提交batch
							logSink.autoCommmitChan <- logBatch
						}
					}(logBatch))
			}

			logBatch.Logs = append(logBatch.Logs, log)
			//如果批次满了，就批量保存日志
			if len(logBatch.Logs) >= config.G_config.JobLogBatchSize {
				//取消定时器
				commitTimer.Stop()
				//保存日志
				logSink.saveLogs(logBatch)
				//清空batch
				logBatch = nil
			}
		case timeoutBatch = <-logSink.autoCommmitChan: //定时器到期的批次
			//判断过期批次是否当前批次
			if timeoutBatch != logBatch {
				continue //跳过已经被提交的批次
			}
			//把批次写入到mongo中
			logSink.saveLogs(timeoutBatch)
			logBatch = nil
		} //end of select
	}
}

//发送日志
func (logSink *LogSink) Append(jobLog *common.JobLog) {
	select {
	case logSink.logChan <- jobLog:
	default:
		//队列满了就丢弃

	}
}
