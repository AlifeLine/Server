package TrafficMonitor

import (
	"XPortForward/DbTables"
	"XPortForward/TypeDefine"
	"XPortForward/XConfig"
	"context"
	"github.com/go-redis/redis/v7"
	"github.com/jinzhu/gorm"
	"log"
	"sync"
	"time"
)

func StartTrafficMonitor(db *gorm.DB,MainExitCtx context.Context,conf *XConfig.Config,wg *sync.WaitGroup,RedisClient *redis.Client,ProxyList *map[string]*TypeDefine.ProxyData,TrafficSyncData chan interface{}){
	log.Println("Start Traffic Monitor....")
	MonitorTimerTick := time.NewTicker(1*time.Hour)
	// MonitorTimerTick := time.NewTicker(1*time.Minute)
	MonitorWG := sync.WaitGroup{}
	MonitorWG.Add(1)
	go Sync_Traffic_Local(TrafficSyncData,db,MainExitCtx,&MonitorWG)
	MonitorWG.Add(1)
	go Sync_Traffic_Remote_Timer(db,MainExitCtx,conf,&MonitorWG)
	MonitorWG.Add(1)
	go Delete_Timeout_Traffic_Logs(db,MainExitCtx,conf,&MonitorWG)
	ExitFlag := false
	for{
		select{
		case <- MonitorTimerTick.C:
			var ForwardList []DbTables.Forward
			_ = db.Table("forwards").Where("status = ?", "ok").Find(&ForwardList)
			if len(ForwardList) != 0{
				for _,v := range ForwardList{
					if ProxyDataInfo,ProxyDataInfoExist := (*ProxyList)[v.ForwardName]; ProxyDataInfoExist{
						go Monitor(v,RedisClient,TrafficSyncData,ProxyDataInfo,false)
					}
				}
			}
		case <- MainExitCtx.Done():
			ExitFlag = true
			MonitorTimerTick.Stop()
			break
		}
		if ExitFlag{
			break
		}
	}
	MonitorWG.Wait()
	log.Println("Traffic Monitor Stopped Success")
	wg.Done()
}

func Delete_Timeout_Traffic_Logs(db *gorm.DB,MainExitCtx context.Context,conf *XConfig.Config,wg *sync.WaitGroup){
	ExitFlag := false
	TimerTick := time.NewTicker(time.Hour*1)
	// TimerTick := time.NewTicker(time.Minute*2)
	for {
		select {
		case <- MainExitCtx.Done():
			ExitFlag = true
			TimerTick.Stop()
			break
		case <- TimerTick.C:
			_ = db.Where("created_at < ?", time.Now().AddDate(0, 0, -32)).Delete(DbTables.TrafficLog{})
		}
		if ExitFlag{
			break
		}
	}
	wg.Done()
}