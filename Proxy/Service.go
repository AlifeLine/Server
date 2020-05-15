package Proxy

import (
	"XPortForward/DbTables"
	"XPortForward/TrafficMonitor"
	"XPortForward/XConfig"
	"XPortForward/TypeDefine"
	"context"
	"github.com/go-redis/redis/v7"
	"github.com/jinzhu/gorm"
	"io"
	"log"
	"sync"
)

var RedisClient *redis.Client
var db *gorm.DB

func Service(Mdb *gorm.DB,MainExitCtx context.Context,PortSyncChannel chan interface{},wg *sync.WaitGroup,Mredis_client *redis.Client,ProxyList map[string]*TypeDefine.ProxyData,TrafficSyncData chan interface{}){
	var ProxyDBList []DbTables.Forward
	log.Println("Start Proxy Service....")
	RedisClient = Mredis_client
	db = Mdb
	_ = db.Table("forwards").Where("status = ?", "ok").Find(&ProxyDBList)
	for _,v := range ProxyDBList{
		Service_Proxy_Init(v,ProxyList)
	}
	go Service_Sync(wg,ProxyList,MainExitCtx,PortSyncChannel,TrafficSyncData)
	<- MainExitCtx.Done()
	//关闭所有打开的proxy,清理操作
	StopWg := sync.WaitGroup{}
	for _,v := range ProxyList{
		if v.Status == "running"{
			//	同步流量到数据库
			TrafficInfo := TrafficMonitor.Monitor(v.DbData,RedisClient,TrafficSyncData,ProxyList[v.DbData.ForwardName],true)
			if len(TrafficInfo) != 0{
				TrafficMonitor.NoSync_Traffic_Local(TrafficInfo,db)
			}
			StopWg.Add(1)
			go (*v.Service).Stop(true,&StopWg)
		}
	}
	StopWg.Wait()
	log.Println("Proxy Service Stopped Success")
	wg.Done()
}

func Service_Proxy_Init(v DbTables.Forward,ProxyList map[string]*TypeDefine.ProxyData){
	var wg sync.WaitGroup
	var ProxyService TypeDefine.ProxyServiceInterface
	if v.ForwardType != "tcp" && v.ForwardType != "udp"{
		log.Println("Forward Type Not Support: ForwardName[" + v.ForwardName + "],Type[" + v.ForwardType + "]")
		return
	}
	if v.ForwardType == "tcp"{
		ProxyService = NewTcpProxy(v.ForwardName,v.ProxyIP,v.ProxyPort,v.ListenIP,v.ListenPort,*v.MaxConnNum,*v.RateLimit,RedisClient)
	}else{
		ProxyService = NewUdpProxy(v.ForwardName,v.ProxyIP,v.ProxyPort,v.ListenIP,v.ListenPort,RedisClient)
	}
	ProxyList[v.ForwardName] = &TypeDefine.ProxyData{v,"running",&ProxyService}
	ProxyService.SetProxyData(ProxyList[v.ForwardName])
	wg.Add(1)
	go func(wg *sync.WaitGroup,ProxyData *TypeDefine.ProxyData){
		defer func() {
			err := recover()
			if err != nil {
				ProxyData.Status = "stopped"
				if XConfig.Debug{
					log.Println(err.(string))
				}
			}
			wg.Done()
		}()
		ProxyService.Start()
	}(&wg,ProxyList[v.ForwardName])
	wg.Wait()
}
func Service_Sync(wg *sync.WaitGroup,ProxyList map[string]*TypeDefine.ProxyData,MainExitCtx context.Context,PortSyncChannel chan interface{},TrafficSyncData chan interface{}){
	ExitFlag := false
	for{
		select {
		case <- MainExitCtx.Done():
			ExitFlag = true
			break
		case v:= <- PortSyncChannel:
			if PortSyncChannelData,PortSyncChannelDataSuccess := v.(map[string]string); PortSyncChannelDataSuccess {
				if PortSyncChannelData["action"] == "forward_delete"{
					if ProxyMapData,ProxyMapDataExist := ProxyList[(PortSyncChannelData["pfname"])]; ProxyMapDataExist{
						if ProxyMapData.Status == "running"{
							// 同步流量数据到数据库
							// TrafficMonitor.Monitor(ProxyMapData.DbData,RedisPool,TrafficSyncData,(*ProxyList)[(PortSyncChannelData["pfname"])],false)
							(*ProxyMapData.Service).Stop(false,&sync.WaitGroup{})
						}
						ProxyList[(PortSyncChannelData["pfname"])] = &TypeDefine.ProxyData{}
						delete(ProxyList,PortSyncChannelData["pfname"])
					}
				}else if PortSyncChannelData["action"] == "forward_reload"{
					var ProxyDb DbTables.Forward
					//Reload 包含编辑和新增,所以存在的话先delete掉原始
					if ProxyMapData,ProxyMapDataExist := ProxyList[(PortSyncChannelData["pfname"])]; ProxyMapDataExist{
						if ProxyMapData.Status == "running"{
							// 同步流量数据到数据库
							TrafficMonitor.Monitor(ProxyMapData.DbData,RedisClient,TrafficSyncData,ProxyList[(PortSyncChannelData["pfname"])],false)
							(*ProxyMapData.Service).Stop(false,&sync.WaitGroup{})
						}
						ProxyList[(PortSyncChannelData["pfname"])] = &TypeDefine.ProxyData{}
						delete(ProxyList,PortSyncChannelData["pfname"])
					}
					_ = db.Table("forwards").Where("forward_name=?",PortSyncChannelData["pfname"]).First(&ProxyDb)
					if ProxyDb != (DbTables.Forward{}){
						//数据存在且状态正常
						if ProxyDb.Status == "ok"{
							Service_Proxy_Init(ProxyDb,ProxyList)
						}
					}
				}
			}
		}
		if ExitFlag{
			break
		}
	}
	log.Println("Proxy Service Sync Stopped Success")
	wg.Done()
}

//Copy From https://github.com/snail007/goproxy
func ioCopy(dst io.Writer, src io.Reader, fn ...func(count int)) (written int64, isSrcErr bool, err error) {
	buf := make([]byte, 1024)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
				if len(fn) == 1 {
					fn[0](nw)
				}
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			err = er
			isSrcErr = true
			break
		}
	}
	return written, isSrcErr, err
}