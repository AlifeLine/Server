package main

import (
	"XPortForward/ApiServer"
	"XPortForward/DbTables"
	"XPortForward/Proxy"
	"XPortForward/TrafficMonitor"
	"XPortForward/TypeDefine"
	"XPortForward/XConfig"
	"context"
	"github.com/go-redis/redis/v7"
	"github.com/jinzhu/gorm"
	"net/http"
	"os"
	"os/signal"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"log"
	_ "net/http/pprof"
	"sync"
)

func main(){
	log.Println("Welcome To Use Port Forward Server!")
	log.Println("Version: " + XConfig.Version)
	if XConfig.Debug{
		log.Println("This is a debug version, please do not use it in a production environment")
	}else{
		log.Println("This version can be used in a production environment, if you encounter problems in use, please go to the Github project homepage to submit Issue")
	}
	if XConfig.Debug{
		log.Println("Go pprof Work On 0.0.0.0:11111")
		go func() {
			_ = http.ListenAndServe("0.0.0.0:11111", nil)
		}()
	}
	MainExitCtx, MainExitFunction := context.WithCancel(context.Background())
	// 端口转发Sync Channel,负责处理TrafficMonitor、ApiServer与转发服务的沟通
	PortSyncChannel := make(chan interface{},233)
	// 流量监控同步Channel
	TrafficSyncData := make(chan interface{})
	OsSignal := make(chan os.Signal, 1)
	var wg sync.WaitGroup
	var ProxyDataList = map[string]*TypeDefine.ProxyData{}
	ElfPath,ElfPathErr := XConfig.GetCurrentPath()
	if ElfPathErr != nil{
		log.Fatal("Elf Path Get Error: " + ElfPathErr.Error())
	}
	Xconf := XConfig.InitConfig(ElfPath)
	// 2020-5-14弃用Sqlite,转为Mysql
	// db, DbErr := gorm.Open("sqlite3", ElfPath + "data.db")
	db, DbErr := gorm.Open("mysql",Xconf.Get("database","url"))
	if DbErr != nil {
		log.Fatal("Failed to connect database: " + DbErr.Error())
	}
	if !db.HasTable("forwards"){
		//初始化数据表
		db.CreateTable(&DbTables.Forward{})
	}
	if !db.HasTable("traffic_logs"){
		//初始化数据表
		db.CreateTable(&DbTables.TrafficLog{})
	}
	// SetMaxIdleCons 设置连接池中的最大闲置连接数。
	db.DB().SetMaxIdleConns(23)
	// SetMaxOpenCons 设置数据库的最大连接数量。
	db.DB().SetMaxOpenConns(23)
	// SetConnMaxLifetiment 设置连接的最大可复用时间。
	db.DB().SetConnMaxLifetime(time.Hour)
	defer func(){
		DbCloseErr := db.Close()
		// 告知DB链接关闭失败,可能会造成数据库损坏(几率较小)
		if DbCloseErr != nil{
			log.Println("Db Close Error: " + DbCloseErr.Error())
		}
	}()
	RedisClient := redis.NewClient(&redis.Options{
		Addr:     Xconf.Get("redis","address"),
		Password: Xconf.Get("redis","password"), // no password set
		DB:       0,  // use default DB
	})
	defer func(){
		_ = RedisClient.Close()
	}()
	signal.Notify(OsSignal, os.Interrupt, os.Kill)
	wg.Add(2)
	go Proxy.Service(db,MainExitCtx,PortSyncChannel,&wg,RedisClient,ProxyDataList,TrafficSyncData)
	wg.Add(1)
	go ApiServer.Start_api_server(Xconf.Get("system","authkey"),db,MainExitCtx,&wg,PortSyncChannel,RedisClient)
	wg.Add(1)
	go TrafficMonitor.StartTrafficMonitor(db,MainExitCtx,Xconf,&wg,RedisClient,&ProxyDataList,TrafficSyncData)
	go OsSignalListen(OsSignal,MainExitCtx,MainExitFunction)
	wg.Wait()
	//退出
	// os.Exit(0)
}

func OsSignalListen(OsSignal chan os.Signal,MainExitCtx context.Context,MainExitFunction context.CancelFunc){
	for{
		select {
		case <-OsSignal:
			MainExitFunction()
			break
		case <-MainExitCtx.Done():
			break
		}
		break
	}
}
