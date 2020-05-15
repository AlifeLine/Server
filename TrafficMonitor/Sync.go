package TrafficMonitor

import (
	"XPortForward/DbTables"
	"XPortForward/XConfig"
	"context"
	"encoding/json"
	"github.com/jinzhu/gorm"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

func Sync_Traffic_Local(TrafficSyncData chan interface{},db *gorm.DB,MainExitCtx context.Context,wg *sync.WaitGroup){
	// 该函数负责处理Sync To DB相关,最大限度的避免出现连接池满及同时操作带来的问题,同时提供其他可能会做的操作准备
	StopFlag := false
	for {
		select {
			case SyncData := <- TrafficSyncData:
				//处理同步
				MapData,Success := SyncData.(map[string]interface{})
				if Success{
					//处理,暂时只有写到DB,UploaBandwidth、DownloaBandwidth、TrafficAll
					// 2020-5-14 TrafficAll参数不准确,废弃
					HourTrafficBandwidthData,_ := MapData["HourTrafficBandwidth"].(float64)
					// _ = db.Table("forwards").Where("forward_name=?",MapData["ForwardName"].(string)).Update("used_bandwidth",MapData["TrafficAll"].(float64))
					_ = db.Table("forwards").Where("forward_name=?",MapData["ForwardName"].(string)).Update("used_bandwidth",gorm.Expr("used_bandwidth + ?", HourTrafficBandwidthData))
					_ = db.Table("traffic_logs").Create(&DbTables.TrafficLog{ForwardName:MapData["ForwardName"].(string),HourUsedBandwidth:&HourTrafficBandwidthData,HourTime:time.Now()})
				}
			case <- MainExitCtx.Done():
				StopFlag = true
				break
		}
		if StopFlag{
			break
		}
	}
	wg.Done()
}

func NoSync_Traffic_Local(TrafficSyncData map[string]interface{},db *gorm.DB){
	// 该函数负责非Channel处理Sync To DB相关,最大限度的避免出现连接池满及同时操作带来的问题,同时提供其他可能会做的操作准备
	HourTrafficBandwidthData,_ := TrafficSyncData["HourTrafficBandwidth"].(float64)
	_ = db.Table("forwards").Where("forward_name=?",TrafficSyncData["ForwardName"].(string)).Update("used_bandwidth",gorm.Expr("used_bandwidth + ?", HourTrafficBandwidthData))
	_ = db.Table("traffic_logs").Create(&DbTables.TrafficLog{ForwardName:TrafficSyncData["ForwardName"].(string),HourUsedBandwidth:&HourTrafficBandwidthData,HourTime:time.Now()})
}

func Sync_Traffic_Remote_Timer(db *gorm.DB,MainExitCtx context.Context,conf *XConfig.Config,wg *sync.WaitGroup){
	// 该函数负责将DB里待同步的数据同步到远端
	ExitFlag := false
	TimerTick := time.NewTicker(time.Minute*23)
	// TimerTick := time.NewTicker(time.Minute*1)
	for {
		select {
		case <- MainExitCtx.Done():
			ExitFlag = true
			TimerTick.Stop()
			break
		case <- TimerTick.C:
			var TrafficLogList []DbTables.TrafficLog
			_ = db.Table("traffic_logs").Where("report_status = ?", "wait").Find(&TrafficLogList)
			if len(TrafficLogList) != 0 {
				ReportInfo := map[string]interface{}{}
				ReportInfo["UsedBandwidth"] = map[string]float64{}
				ReportInfo["TrafficLogs"] = map[string]interface{}{}
				for _, v := range TrafficLogList {
					if _,UsedBandwidthExist := ReportInfo["UsedBandwidth"].(map[string]float64)[v.ForwardName]; !UsedBandwidthExist{
						var ForwardInfo DbTables.Forward
						_ = db.Table("forwards").Where("forward_name = ?", v.ForwardName).Find(&ForwardInfo)
						if ForwardInfo == (DbTables.Forward{}) {
							continue
						}
						ReportInfo["UsedBandwidth"].(map[string]float64)[v.ForwardName] = *ForwardInfo.UsedBandwidth
					}
					ReportInfoMap := map[string]interface{}{}
					ReportInfoMap["Time"] = v.HourTime
					ReportInfoMap["HourUsedBandwidth"] = v.HourUsedBandwidth
					ReportInfoMap["ForwardName"] = v.ForwardName
					ReportInfo["TrafficLogs"].(map[string]interface{})[strconv.Itoa(int(v.ID))] = ReportInfoMap
				}
				ReportJson, ReportJsonGenErr := json.Marshal(ReportInfo)
				if ReportJsonGenErr != nil {
					if XConfig.Debug {
						log.Println("Report Info Json Encode Error: " + ReportJsonGenErr.Error())
					}
					continue
				}
				ReportRequest, _ := http.NewRequest("POST", conf.Get("report", "url"), strings.NewReader(string(ReportJson)))
				ReportResp, ReportRespErr := http.DefaultClient.Do(ReportRequest)
				if ReportRespErr != nil {
					if XConfig.Debug {
						log.Println("Report Resp Get Error: " + ReportRespErr.Error())
					}
					continue
				}
				respBody, _ := ioutil.ReadAll(ReportResp.Body)
				if len(respBody) != 0 {
					var ReportResult map[string]interface{}
					ReportResultGenErr := json.Unmarshal([]byte(respBody), &ReportResult)
					if ReportResultGenErr != nil {
						if XConfig.Debug {
							log.Println("Report Result Info Decode Errror: " + ReportResultGenErr.Error())
						}
						continue
					}
					if ReportResult["success"] != true {
						if XConfig.Debug {
							log.Println("Report Error: " + ReportResult["message"].(string))
						}
						continue
					}
					for _, v := range TrafficLogList {
						_ = db.Table("traffic_logs").Where("id=?", v.ID).Update("report_status", "ok")
					}
				}
			}
		}
		if ExitFlag{
			break
		}
	}
	wg.Done()
}

