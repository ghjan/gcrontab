package master

import (
	"context"
	"fmt"
	"github.com/ghjan/gcrontab/common"
	"github.com/ghjan/gcrontab/master/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

//mongodb日志管理
type LogMgr struct {
	client        *mongo.Client
	logCollection *mongo.Collection
}

var (
	G_logMgr *LogMgr
)

func InitLogMgr() (err error) {
	var (
		client *mongo.Client
	)
	if G_logMgr == nil || G_logMgr.client == nil {
		// Set G_logMgr options
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
			log.Fatalf("initMongoClient, G_logMgr.Ping error:%s\n", err.Error())
			return
		}

		G_logMgr = &LogMgr{
			client:        client,
			logCollection: client.Database("cron").Collection("log"),
		}
		//启动一个存储协程
		fmt.Println("InitLogMgr, Connected to MongoDB for cron log!")
		//go G_logMgr.writeLoop()
		return
	}
	return
}

//查看任务日志
func (logMgr *LogMgr) ListLog(name string, skip int, limit int) (logArr []*common.JobLog, err error) {
	var (
		filter  *common.JobLogFilter
		logSort *common.SortLogByStartTime
		cursor  *mongo.Cursor
		jobLog  *common.JobLog
	)
	//过滤条件
	filter = &common.JobLogFilter{JobName: name}
	//按照任务开始时间倒排
	logSort = &common.SortLogByStartTime{SortOrder: -1}
	findOptions := options.Find()
	if cursor, err = logMgr.logCollection.Find(context.TODO(), filter, findOptions.SetSort(logSort),
		findOptions.SetSkip(int64(skip)), findOptions.SetLimit(int64(limit))); err != nil {
		return
	}
	//延迟释放游标
	defer cursor.Close(context.TODO())

	logArr = make([]*common.JobLog, 0)
	for cursor.Next(context.TODO()) {
		jobLog = &common.JobLog{}
		//反序列化BSON
		if err = cursor.Decode(jobLog); err != nil {
			continue //有日志不合法
		}
		logArr = append(logArr, jobLog)
	}

	return
}
