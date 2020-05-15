package TypeDefine

import (
	"XPortForward/DbTables"
	"sync"
)

type ProxyData struct{
	DbData DbTables.Forward
	Status string
	Service *ProxyServiceInterface
}

type ProxyServiceInterface interface {
	Start()
	Stop(IsForce bool,wg *sync.WaitGroup)
	SetProxyData(data *ProxyData)
}
