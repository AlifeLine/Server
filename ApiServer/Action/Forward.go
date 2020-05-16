//Forward 相关 Function
package Action

import (
	"XPortForward/DbTables"
	"XPortForward/TrafficMonitor"
	"XPortForward/TypeDefine"
	"github.com/go-redis/redis/v7"
	"github.com/jinzhu/gorm"
)

func Forward_Add(params map[string]interface{},db *gorm.DB) map[string]interface{}{
	var RateLimit float64
	var ForwardExistDbInfo DbTables.Forward
	var ForwardDbPortInfo DbTables.Forward
	ForwardName,_ := params["forwardname"].(string)
	ProxyIP,_ := params["proxyip"].(string)
	ProxyPort,_ := params["proxyport"].(float64)
	ListenIP,_ := params["listenip"].(string)
	ListenPort,_ := params["listenport"].(float64)
	MaxConnNum,_ := params["maxconnnum"].(float64)
	ForwardType,_ := params["forwardtype"].(string)
	if ForwardType != "udp" && ForwardType != "tcp"{
		return map[string]interface{}{"Status":"error","Code":1,"Message":"Unknow Forward Type"}
	}
	if ForwardType == "udp"{
		RateLimit = 0
	}else{
		RateLimit,_ = params["ratelimit"].(float64)
	}
	MaxConnNumInt64 := int64(MaxConnNum)
	RateLimitInt64 := int64(RateLimit)
	ProxyPortInt := int(ProxyPort)
	ListenPortInt := int(ListenPort)
	if ForwardName == "" || ProxyIP == "" || ProxyPortInt == 0 || ListenIP == "" || ListenPortInt == 0 || MaxConnNumInt64 == 0{
		return map[string]interface{}{"Status":"error","Code":2,"Message":"Incomplete parameters"}
	}
	_ = db.Table("forwards").Where("forward_name = ?", ForwardName).Find(&ForwardExistDbInfo)
	if ForwardExistDbInfo != (DbTables.Forward{}){
		return map[string]interface{}{"Status":"error","Code":27,"Message":"Forward Found"}
	}
	_ = db.Table("forwards").Where("listen_port = ?", ListenPortInt).Find(&ForwardDbPortInfo)
	if ForwardDbPortInfo != (DbTables.Forward{}){
		return map[string]interface{}{"Status":"error","Code":28,"Message":"Listen Port Exist"}
	}
	DbErr := db.Table("forwards").Create(&DbTables.Forward{ForwardName:ForwardName,ProxyIP:ProxyIP,ProxyPort:ProxyPortInt,ListenIP:ListenIP,ListenPort:ListenPortInt,MaxConnNum:&MaxConnNumInt64,ForwardType:ForwardType,RateLimit:&RateLimitInt64,Status:"ok"}).Error
	if DbErr != nil{
		return map[string]interface{}{"Status":"error","Code":3,"Message":"Insert To Database Error: " + DbErr.Error()}
	}
	return map[string]interface{}{"pfsync_need":"have","action_name":"forward_reload","pf_name":ForwardName}
}

func Forward_Edit(params map[string]interface{},db *gorm.DB) map[string]interface{}{
	var RateLimit int64
	var RateLimitFloat64 float64
	var ForwardDbInfo DbTables.Forward
	var RateLimitFloat64Success bool
	ForwardName,_ := params["forwardname"].(string)
	if ForwardName == ""{
		return map[string]interface{}{"Status":"error","Code":4,"Message":"Incomplete parameters"}
	}else{
		_ = db.Table("forwards").Where("forward_name = ?", ForwardName).Find(&ForwardDbInfo)
		if ForwardDbInfo == (DbTables.Forward{}){
			return map[string]interface{}{"Status":"error","Code":5,"Message":"Forward Not Found"}
		}
	}
	ProxyIP,_ := params["proxyip"].(string)
	if ProxyIP == ""{
		ProxyIP = ForwardDbInfo.ProxyIP
	}
	ProxyPort,_ := params["proxyport"].(float64)
	ProxyPortInt := int(ProxyPort)
	if ProxyPortInt == 0{
		ProxyPortInt = ForwardDbInfo.ProxyPort
	}
	ListenIP,_ := params["listenip"].(string)
	if ListenIP == ""{
		ListenIP = ForwardDbInfo.ListenIP
	}
	ListenPort,_ := params["listenport"].(float64)
	ListenPortInt := int(ListenPort)
	if ListenPortInt == 0{
		ListenPortInt = ForwardDbInfo.ListenPort
	}else{
		var ForwardDbPortInfo DbTables.Forward
		_ = db.Table("forwards").Where("listen_port = ?", ListenPortInt).Find(&ForwardDbPortInfo)
		if ForwardDbPortInfo != (DbTables.Forward{}){
			return map[string]interface{}{"Status":"error","Code":29,"Message":"Listen Port Exist"}
		}
	}
	MaxConnNum,_ := params["maxconnnum"].(float64)
	MaxConnNumInt64 := int64(MaxConnNum)
	if MaxConnNumInt64 == 0{
		MaxConnNumInt64 = *ForwardDbInfo.MaxConnNum
	}
	ForwardType,_ := params["forwardtype"].(string)
	if ForwardType == ""{
		ForwardType = ForwardDbInfo.ForwardType
	}else{
		if ForwardType != "udp" && ForwardType != "tcp"{
			return map[string]interface{}{"Status":"error","Code":6,"Message":"Unknow Forward Type"}
		}
	}
	if ForwardType == "udp"{
		RateLimit = 0
	}else{
		RateLimitFloat64,RateLimitFloat64Success = params["ratelimit"].(float64)
		if !RateLimitFloat64Success{
			RateLimit = *ForwardDbInfo.RateLimit
		}else{
			RateLimit = int64(RateLimitFloat64)
		}
	}
	DbErr := db.Table("forwards").Where("forward_name=?",ForwardName).Updates(map[string]interface{}{"proxy_ip":ProxyIP,"proxy_port":ProxyPortInt,"listen_ip":ListenIP,"listen_port":ListenPortInt,"max_conn_num":MaxConnNumInt64,"rate_limit":RateLimit,"forward_type":ForwardType}).Error
	if DbErr != nil{
		return map[string]interface{}{"Status":"error","Code":7,"Message":"Update To Database Error: " + DbErr.Error()}
	}
	return map[string]interface{}{"pfsync_need":"have","action_name":"forward_reload","pf_name":ForwardName}
}

func Forward_Del(params map[string]interface{},db *gorm.DB,RedisClient *redis.Client,ah *TypeDefine.ApiHandle) map[string]interface{}{
	var ForwardDbInfo DbTables.Forward
	var TrafficInfo map[string]interface{}
	ForwardName,_ := params["forwardname"].(string)
	if ForwardName == ""{
		return map[string]interface{}{"Status":"error","Code":8,"Message":"Incomplete parameters"}
	}else{
		_ = db.Table("forwards").Where("forward_name = ?", ForwardName).Find(&ForwardDbInfo)
		if ForwardDbInfo == (DbTables.Forward{}){
			return map[string]interface{}{"Status":"error","Code":9,"Message":"Forward Not Found"}
		}
	}
	// DbErr := db.Table("forwards").Where("forward_name=?",ForwardName).Delete(&ForwardDbInfo).Error
	DbErr := db.Table("forwards").Where("forward_name=?",ForwardName).Delete(DbTables.Forward{}).Error
	if DbErr != nil{
		return map[string]interface{}{"Status":"error","Code":10,"Message":"Delete To Database Error: " + DbErr.Error()}
	}
	DbErrTrafficLog := db.Table("traffic_logs").Where("forward_name=?",ForwardName).Delete(DbTables.TrafficLog{}).Error
	if DbErrTrafficLog != nil{
		return map[string]interface{}{"Status":"error","Code":10,"Message":"Delete To Database Error(Traffic Log): " + DbErrTrafficLog.Error()}
	}
	//TODO,暂时没找到合适的方法
	if _,ProxyDataExist := ah.ProxyDataList[ForwardDbInfo.ForwardName];ProxyDataExist{
		TrafficInfo = TrafficMonitor.Monitor(ForwardDbInfo,RedisClient,ah.TrafficSyncData,ah.ProxyDataList[ForwardDbInfo.ForwardName],true)
	}else{
		TrafficInfo = map[string]interface{}{"ForwardName":ForwardDbInfo.ForwardName,"TrafficAll":0,"UploadBandwidth":0,"DownloadBandwidth":0,"HourTrafficBandwidth":0}
	}
	return map[string]interface{}{"pfsync_need":"have","action_name":"forward_delete","pf_name":ForwardName,"data":TrafficInfo}
}

func Forward_StatusChange(params map[string]interface{},db *gorm.DB) map[string]interface{}{
	var ForwardDbInfo DbTables.Forward
	ForwardName,_ := params["forwardname"].(string)
	ForwardStatus,_ := params["status"].(string)
	if ForwardName == ""{
		return map[string]interface{}{"Status":"error","Code":11,"Message":"Incomplete parameters"}
	}else{
		_ = db.Table("forwards").Where("forward_name = ?", ForwardName).Find(&ForwardDbInfo)
		if ForwardDbInfo == (DbTables.Forward{}){
			return map[string]interface{}{"Status":"error","Code":12,"Message":"Forward Not Found"}
		}
	}
	if ForwardStatus != "ok" && ForwardStatus != "stop"{
		return map[string]interface{}{"Status":"error","Code":13,"Message":"Status Not Allow"}
	}
	DbErr := db.Table("forwards").Where("forward_name=?",ForwardName).Update("status",ForwardStatus).Error
	if DbErr != nil{
		return map[string]interface{}{"Status":"error","Code":14,"Message":"Update To Database Error: " + DbErr.Error()}
	}
	return map[string]interface{}{"pfsync_need":"have","action_name":"forward_reload","pf_name":ForwardName}
}

func Forward_GetInfos(params map[string]interface{},db *gorm.DB) map[string]interface{}{
	var ForwardDbInfo DbTables.Forward
	ForwardName,_ := params["forwardname"].(string)
	if ForwardName == ""{
		return map[string]interface{}{"Status":"error","Code":15,"Message":"Incomplete parameters"}
	}else{
		_ = db.Table("forwards").Where("forward_name = ?", ForwardName).Find(&ForwardDbInfo)
		if ForwardDbInfo == (DbTables.Forward{}){
			return map[string]interface{}{"Status":"error","Code":16,"Message":"Forward Not Found"}
		}
	}
	ForWardInfos := map[string]interface{}{}
	ForWardInfos["Status"] = ForwardDbInfo.Status
	ForWardInfos["ProxyIP"] = ForwardDbInfo.ProxyIP
	ForWardInfos["ProxyPort"] = ForwardDbInfo.ProxyPort
	ForWardInfos["ListenIP"] = ForwardDbInfo.ListenIP
	ForWardInfos["ListenPort"] = ForwardDbInfo.ListenPort
	ForWardInfos["MaxConnNum"] = ForwardDbInfo.MaxConnNum
	ForWardInfos["RateLimit"] = ForwardDbInfo.RateLimit
	ForWardInfos["ForwardType"] = ForwardDbInfo.ForwardType
	ForWardInfos["UsedBandwidth"] = ForwardDbInfo.UsedBandwidth
	return map[string]interface{}{"Status":"Success","Code":17,"Message":"Success","Info":ForWardInfos}
}

func Forward_GetTrafficLogs(params map[string]interface{},db *gorm.DB) map[string]interface{}{
	var ForwardDbInfo DbTables.Forward
	var ForwardTrafficLogsDbInfo []DbTables.TrafficLog
	ForwardName,_ := params["forwardname"].(string)
	if ForwardName == ""{
		return map[string]interface{}{"Status":"error","Code":18,"Message":"Incomplete parameters"}
	}else{
		_ = db.Table("forwards").Where("forward_name = ?", ForwardName).Find(&ForwardDbInfo)
		if ForwardDbInfo == (DbTables.Forward{}){
			return map[string]interface{}{"Status":"error","Code":19,"Message":"Forward Not Found"}
		}
		if GetUnReportLog,_ := params["un_report_log"].(bool);GetUnReportLog{
			_ = db.Table("traffic_logs").Where("forward_name = ?", ForwardName).Where("report_status = ?", "wait").Find(&ForwardTrafficLogsDbInfo)
		}else{
			_ = db.Table("traffic_logs").Where("forward_name = ?", ForwardName).Find(&ForwardTrafficLogsDbInfo)
		}
	}
	return map[string]interface{}{"Status":"Success","Code":20,"Message":"Success","Info":ForwardTrafficLogsDbInfo}
}

func Forward_Bandwidth_Reset(params map[string]interface{},db *gorm.DB) map[string]interface{}{
	var ForwardDbInfo DbTables.Forward
	ForwardName,_ := params["forwardname"].(string)
	if ForwardName == ""{
		return map[string]interface{}{"Status":"error","Code":21,"Message":"Incomplete parameters"}
	}else{
		_ = db.Table("forwards").Where("forward_name = ?", ForwardName).Find(&ForwardDbInfo)
		if ForwardDbInfo == (DbTables.Forward{}){
			return map[string]interface{}{"Status":"error","Code":22,"Message":"Forward Not Found"}
		}
	}
	DbErr := db.Table("forwards").Where("forward_name=?",ForwardName).Update("used_bandwidth",0).Error
	if DbErr != nil{
		return map[string]interface{}{"Status":"error","Code":23,"Message":"Update To Database Error: " + DbErr.Error()}
	}
	return map[string]interface{}{"Status":"Success","Code":24,"Message":"Success"}
}

func Forward_Reload(params map[string]interface{},db *gorm.DB) map[string]interface{}{
	var ForwardDbInfo DbTables.Forward
	ForwardName,_ := params["forwardname"].(string)
	if ForwardName == ""{
		return map[string]interface{}{"Status":"error","Code":25,"Message":"Incomplete parameters"}
	}else{
		_ = db.Table("forwards").Where("forward_name = ?", ForwardName).Find(&ForwardDbInfo)
		if ForwardDbInfo == (DbTables.Forward{}){
			return map[string]interface{}{"Status":"error","Code":26,"Message":"Forward Not Found"}
		}
	}
	return map[string]interface{}{"pfsync_need":"have","action_name":"forward_reload","pf_name":ForwardName}
}

func Forward_DelSync(params map[string]interface{},db *gorm.DB) map[string]interface{}{
	var ForwardDbInfo DbTables.Forward
	ForwardName,_ := params["forwardname"].(string)
	if ForwardName == ""{
		return map[string]interface{}{"Status":"error","Code":31,"Message":"Incomplete parameters"}
	}else{
		_ = db.Table("forwards").Where("forward_name = ?", ForwardName).Find(&ForwardDbInfo)
		if ForwardDbInfo != (DbTables.Forward{}){
			return map[string]interface{}{"Status":"error","Code":32,"Message":"Forward Found"}
		}
	}
	return map[string]interface{}{"pfsync_need":"have","action_name":"forward_delete","pf_name":ForwardName}
}