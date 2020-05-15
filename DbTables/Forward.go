package DbTables

import (
	"time"
)

//定义Forward相关表
type Forward struct {
	ID uint `gorm:"primary_key"`
	//转发唯一名称
	ForwardName string `gorm:"type:varchar(100);default:''"`
	//转发到的服务器IP
	ProxyIP string `gorm:"type:text"`
	//转发到的服务器端口
	ProxyPort int
	//转发服务器监听IP地址
	ListenIP string `gorm:"type:text"`
	//转发服务器监听的端口
	ListenPort int
	//并发数
	MaxConnNum *int64 `gorm:"type:varchar(100);default:0"`
	//速度(kb/s)
	RateLimit  *int64 `gorm:"default:0"`
	//转发类别(tcp/udp)
	ForwardType string `gorm:"type:varchar(100);default:'tcp'"`
	//已用流量(kb)
	UsedBandwidth *float64 `gorm:"type:DECIMAL(65,4);default:0"`
	//转发状态
	Status string `gorm:"type:varchar(100);default:'wait'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
