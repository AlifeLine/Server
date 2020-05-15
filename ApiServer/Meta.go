package ApiServer

import (
	"XPortForward/Funcmap"
	"XPortForward/ApiServer/Action"
)

type ActionInfoMap struct {
	Function interface{}
	RedisNeed bool
	DbNeed bool
}
func Action_meta() (Funcmap.Funcs,map[string]ActionInfoMap){
	//预定义相关Action
	ActionData := map[string]ActionInfoMap{}
	//Forward相关Function
	ActionData["add_forward"] = ActionInfoMap{Action.Forward_Add,false,true}
	ActionData["edit_forward"] = ActionInfoMap{Action.Forward_Edit,false,true}
	ActionData["del_forward"] = ActionInfoMap{Action.Forward_Del,false,true}
	ActionData["change_status_forward"] = ActionInfoMap{Action.Forward_StatusChange,false,true}
	ActionData["get_forward_infos"] = ActionInfoMap{Action.Forward_GetInfos,false,true}
	ActionData["get_forward_traffic_logs"] = ActionInfoMap{Action.Forward_GetTrafficLogs,false,true}
	ActionData["reset_forward_bandwidth"] = ActionInfoMap{Action.Forward_Bandwidth_Reset,false,true}
	ActionData["reload_forward"] = ActionInfoMap{Action.Forward_Reload,false,true}
	//Server相关Function
	ActionData["server_info"] = ActionInfoMap{Action.Server_Info,false,false}
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
