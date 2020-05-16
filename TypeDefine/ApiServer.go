package TypeDefine

import (
	"github.com/go-redis/redis/v7"
	"github.com/jinzhu/gorm"
)

type ApiHandle struct {
	AuthKey string
	db *gorm.DB
	PortSyncChannel chan interface{}
	RedisClient *redis.Client
	TrafficSyncData chan interface{}
	ProxyDataList map[string]*ProxyData
}

type ResultData struct{
	Code int `json:"code"`
	Message string `json:"message"`
	Data interface{} `json:"data"`
}
