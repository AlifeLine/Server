//Server 相关 Function
package Action

import (
	"XPortForward/XConfig"
	"time"
)

func Server_Info() map[string]interface{}{
	return map[string]interface{}{"Status":"Success","Code":30,"Message":"Success","SoftVersion":XConfig.Version,"ServerTime":int64(time.Now().Unix()),"Author":"Flyqie"}
}