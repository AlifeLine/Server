package ApiServer

import (
	"XPortForward/Funcmap"
	"XPortForward/ApiServer/Action"
)

type ActionInfoMap struct {
	Function interface{}
	RedisNeed bool
	DbNeed bool
	AhNeed bool
}
func Action_meta() (Funcmap.Funcs,map[string]ActionInfoMap){
	//警告:ApiHandleNeed选项将会同时传递关键内容,请确保操作可以正确处理这些敏感信息!!
	//预定义相关Action
	ActionData := map[string]ActionInfoMap{}
	//Forward相关Function
	ActionData["add_forward"] = ActionInfoMap{Action.Forward_Add,false,true,false}
	ActionData["edit_forward"] = ActionInfoMap{Action.Forward_Edit,false,true,false}
	ActionData["del_forward"] = ActionInfoMap{Action.Forward_Del,true,true,true}
	ActionData["change_status_forward"] = ActionInfoMap{Action.Forward_StatusChange,false,true,false}
	ActionData["get_forward_infos"] = ActionInfoMap{Action.Forward_GetInfos,false,true,false}
	ActionData["get_forward_traffic_logs"] = ActionInfoMap{Action.Forward_GetTrafficLogs,false,true,false}
	ActionData["reset_forward_bandwidth"] = ActionInfoMap{Action.Forward_Bandwidth_Reset,false,true,false}
	ActionData["reload_forward"] = ActionInfoMap{Action.Forward_Reload,false,true,false}
	ActionData["forward_del_sync"] = ActionInfoMap{Action.Forward_DelSync,false,true,false}
	//Server相关Function
	ActionData["server_info"] = ActionInfoMap{Action.Server_Info,false,false,false}
	// Action Define End
	ActionList := map[string]ActionInfoMap{}
	FuncMap := Funcmap.NewFuncs(23333)
	//记录可用操作
	for k,v := range ActionData{
		ActionList[k] = v
		_ = FuncMap.Bind(k,v.Function)
	}
	return FuncMap,ActionList
}
