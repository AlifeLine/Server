package ApiServer

import (
	"XPortForward/XConfig"
	_ "XPortForward/DbTables"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"log"
	"net/http"
	"reflect"
	// "strconv"
	"github.com/jinzhu/gorm"
	"XPortForward/Funcmap"
	"sync"
	"time"
	_ "XPortForward/ApiServer/Action"
	// "github.com/pascaldekloe/latest"
)
type ApiHandle struct {
	AuthKey string
	db *gorm.DB
	PortSyncChannel chan interface{}
	RedisClient *redis.Client
}

type ResultData struct{
	Code int `json:"code"`
	Message string `json:"message"`
	Data interface{} `json:"data"`
}

var (
	ActionList map[string]ActionInfoMap
	FuncMap Funcmap.Funcs
)
func (ah *ApiHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//Sign签名
	AuthSign := fmt.Sprintf("%x", md5.Sum([]byte(ah.AuthKey+"|PortForward")))
	if r.URL.Query().Get("sign") != AuthSign {
		_,err := w.Write(ah.ResultGen(ResultData{403,"Auth Error",""},true))
		if err != nil && XConfig.Debug{
			log.Println("[Auth Error] Send Error: " + err.Error())
		}
		return
	}
	if r.URL.Query().Get("action") == ""{
		_,err := w.Write(ah.ResultGen(ResultData{403,"Action cannot be empty",""},true))
		if err != nil && XConfig.Debug{
			log.Println("[Action cannot be empty] Send Error: " + err.Error())
		}
		return
	}
	if _,ActionFound := ActionList[r.URL.Query().Get("action")]; !ActionFound {
		_,err := w.Write(ah.ResultGen(ResultData{403,"Action Not Found",""},true))
		if err != nil && XConfig.Debug{
			log.Println("[Action Not Found] Send Error: " + err.Error())
		}
		return
	}
	var ActionCallResult []reflect.Value
	var ActionCallError error
	ActionCallParameter := map[string]interface{}{}
	if r.URL.Query().Get("action_data") != "" {
		var TmpActionCallParameter map[string]interface{}
		err := json.Unmarshal([]byte(r.URL.Query().Get("action_data")), &TmpActionCallParameter)
		if err != nil && XConfig.Debug {
			log.Println("Decode Action Parameter Error: " + err.Error())
		} else {
			for k, v := range TmpActionCallParameter {
				ActionCallParameter[k] = v
			}
		}
	}
	ActionSetting := ActionList[r.URL.Query().Get("action")]
	if len(ActionCallParameter) != 0{
		//很不优雅的写法,TODO
		if ActionSetting.DbNeed{
			if ActionSetting.RedisNeed{
				ActionCallResult,ActionCallError = FuncMap.Call(r.URL.Query().Get("action"),ActionCallParameter,ah.db,ah.RedisClient)
			}else{
				ActionCallResult,ActionCallError = FuncMap.Call(r.URL.Query().Get("action"),ActionCallParameter,ah.db)
			}
		}else{
			if ActionSetting.RedisNeed{
				ActionCallResult,ActionCallError = FuncMap.Call(r.URL.Query().Get("action"),ActionCallParameter,ah.RedisClient)
			}else{
				ActionCallResult,ActionCallError = FuncMap.Call(r.URL.Query().Get("action"),ActionCallParameter)
			}
		}
	}else{
		//很不优雅的写法,TODO
		if ActionSetting.DbNeed{
			if ActionSetting.RedisNeed{
				ActionCallResult,ActionCallError = FuncMap.Call(r.URL.Query().Get("action"),ah.db,ah.RedisClient)
			}else{
				ActionCallResult,ActionCallError = FuncMap.Call(r.URL.Query().Get("action"),ah.db)
			}
		}else{
			if ActionSetting.RedisNeed{
				ActionCallResult,ActionCallError = FuncMap.Call(r.URL.Query().Get("action"),ah.RedisClient)
			}else{
				ActionCallResult,ActionCallError = FuncMap.Call(r.URL.Query().Get("action"))
			}
		}
	}
	if ActionCallError != nil{
		_,err := w.Write(ah.ResultGen(ResultData{500,string(ActionCallError.Error()),""},true))
		if err != nil && XConfig.Debug{
			log.Println("Action Call Error Send Error: " + err.Error())
		}
	}else{
		_,err := w.Write(ah.ResultGen(ResultData{200,"Success",ActionCallResult[0].Interface()},false))
		if err != nil && XConfig.Debug{
			// Debug 模式下显示Send的具体Error
			log.Println("Action Result Send Error: " + err.Error())
		}
	}
}

func (ah *ApiHandle) ResultGen(info ResultData,IsErr bool) ([]byte){
	if IsErr{
		if XConfig.Debug {
			log.Println("Action Error msg: " + info.Message)
		}
	}
	TaskInfo,_ := info.Data.(map[string]interface{})
	if data,_:= TaskInfo["pfsync_need"]; data != nil{
		timeout := make(chan bool, 1)
		go func() {
			time.Sleep(time.Second*6)
			timeout <- true
		}()
		select {
		case <-timeout:
			info = ResultData{200,"Sync Error",map[string]interface{}{"Status":"error","Code":0,"message":"Sync Error: Time out to channel add data,Please Manual Reload This Forward"}}
		case ah.PortSyncChannel <- map[string]string{"action":TaskInfo["action_name"].(string),"pfname":TaskInfo["pf_name"].(string)}:
			info = ResultData{200,"Success",map[string]interface{}{"Status":"Success"}}

		}
	}
	Encodejson,Encodeerr := json.Marshal(info)
	if Encodeerr != nil{
		log.Println("Json Encode Error: " + Encodeerr.Error())
		return []byte("Something Error!")
	}else{
		return Encodejson
	}
}

func Start_api_server(AuthKey string,db *gorm.DB,MainExitCtx context.Context,wg *sync.WaitGroup,PortSyncChannel chan interface{},RedisClient *redis.Client){
	//定义可用Action及相关映射
	FuncMap,ActionList = Action_meta()
	//启动http服务器
	mux := http.NewServeMux()
	mux.Handle("/", &ApiHandle{AuthKey,db,PortSyncChannel,RedisClient})
	server := &http.Server{
		Addr:         ":2333",
		Handler:      mux,
	}
	log.Println("Start Api Server....")
	var ServerStopFlag = false
	go func(server *http.Server,ServerStopFlag *bool){
		err := server.ListenAndServe()
		if !*ServerStopFlag{
			if err != nil{
				log.Fatal("Api Server Error: " + err.Error())
			}else{
				log.Println("Api Server Exit...")
				wg.Done()
			}
		}
	}(server,&ServerStopFlag)
	<- MainExitCtx.Done()
	log.Println("Api Server Stopping...")
	ServerStopFlag = true
	ServerExitError := server.Close()
	if ServerExitError != nil{
		log.Println("Api Server Exit Error: " + ServerExitError.Error())
	}else{
		log.Println("Api Server Stopped Success")
	}
	wg.Done()
}
