package DbTables

import (
	"time"
)

//定义Forward相关表
type TrafficLog struct {
	ID uint `gorm:"primary_key"`
	//转发唯一名称
	ForwardName string `gorm:"type:varchar(100);default:''"`
	//时段总计流量(kb)
	HourUsedBandwidth *float64 `gorm:"type:DECIMAL(65,4);default:0"`
	//记录时间
	HourTime time.Time
	//上报状态
	ReportStatus string `gorm:"type:varchar(100);default:'wait'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
