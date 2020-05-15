package TrafficMonitor

import (
	"XPortForward/DbTables"
	"XPortForward/TypeDefine"
	"XPortForward/XConfig"
	"github.com/go-redis/redis/v7"
	"log"
	"strconv"
)

func Monitor(PortForwardDB DbTables.Forward,RedisClient *redis.Client,TrafficSyncData chan interface{},Proxydata *TypeDefine.ProxyData,NoSync bool) map[string]interface{}{
	var UploadTraffic float64
	var DownloadTraffic float64
	if Proxydata.Status == "running"{
		//defer func(){
		//	_ = RedisPool.Close()
		//}()
		UploadTrafficRedis,UploadTrafficErr := RedisClient.Do("Get",PortForwardDB.ForwardName + "_UploadTraffic").Result()
		if UploadTrafficErr != nil{
			if XConfig.Debug{
				log.Println(PortForwardDB.ForwardName + " Get Upload Traffic Error:" + UploadTrafficErr.Error())
			}
			UploadTraffic = 0
		}else{
			_,UploadTrafficSetErr := RedisClient.Do("SET",PortForwardDB.ForwardName + "_UploadTraffic",0).Result()
			if UploadTrafficSetErr != nil && XConfig.Debug{
				log.Println("Upload Traffic Set 0 Error: " + UploadTrafficSetErr.Error())
			}
			UploadTrafficString,_ :=  UploadTrafficRedis.(string)
			UploadTraffic,_ = strconv.ParseFloat(UploadTrafficString,64)
		}
		DownloadTrafficRedis,DownloadTrafficErr := RedisClient.Do("Get",PortForwardDB.ForwardName + "_DownloadTraffic").Result()
		if DownloadTrafficErr != nil{
			if XConfig.Debug{
				log.Println(PortForwardDB.ForwardName + " Get Download Traffic Error:" + DownloadTrafficErr.Error())
			}
			DownloadTraffic = 0
		}else{
			_,DownloadTrafficSetErr := RedisClient.Do("SET",PortForwardDB.ForwardName + "_DownloadTraffic",0).Result()
			if DownloadTrafficSetErr != nil && XConfig.Debug{
				log.Println("Download Traffic Set 0 Error: " + DownloadTrafficSetErr.Error())
			}
			DownloadTrafficString,_ :=  DownloadTrafficRedis.(string)
			DownloadTraffic,_ = strconv.ParseFloat(DownloadTrafficString,64)
		}
		HourTrafficBandwidth := (UploadTraffic)+(DownloadTraffic)
		// TrafficBandwidth := HourTrafficBandwidth + (PortForwardDB.UsedBandwidth)
		// 2020-5-10改为Kb
		HourTrafficBandwidth = HourTrafficBandwidth / 1024
		TrafficBandwidth := HourTrafficBandwidth + (*PortForwardDB.UsedBandwidth)
		if NoSync{
			return map[string]interface{}{"ForwardName":PortForwardDB.ForwardName,"TrafficAll":TrafficBandwidth,"UploadBandwidth":UploadTraffic,"DownloadBandwidth":DownloadTraffic,"HourTrafficBandwidth":HourTrafficBandwidth}
		}else{
			TrafficSyncData <- map[string]interface{}{"ForwardName":PortForwardDB.ForwardName,"TrafficAll":TrafficBandwidth,"UploadBandwidth":UploadTraffic,"DownloadBandwidth":DownloadTraffic,"HourTrafficBandwidth":HourTrafficBandwidth}
		}
	}
	return map[string]interface{}{}
	//wg.Done()
}
