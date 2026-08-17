package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type DatabaseMetric struct {
	any               `collection:"database_metrics"`
	BaseModel         `bson:",inline"`
	DatabaseId        primitive.ObjectID `json:"database_id" bson:"database_id" description:"Database ID"`
	CpuUsagePercent   float32            `json:"cpu_usage_percent" bson:"cpu_usage_percent" description:"CPU usage percentage"`
	TotalMemory       uint64             `json:"total_memory" bson:"total_memory" description:"Total memory"`
	AvailableMemory   uint64             `json:"available_memory" bson:"available_memory" description:"Available memory"`
	UsedMemory        uint64             `json:"used_memory" bson:"used_memory" description:"Used memory"`
	UsedMemoryPercent float32            `json:"used_memory_percent" bson:"used_memory_percent" description:"Used memory percentage"`
	TotalDisk         uint64             `json:"total_disk" bson:"total_disk" description:"Total disk"`
	AvailableDisk     uint64             `json:"available_disk" bson:"available_disk" description:"Available disk"`
	UsedDisk          uint64             `json:"used_disk" bson:"used_disk" description:"Used disk"`
	UsedDiskPercent   float32            `json:"used_disk_percent" bson:"used_disk_percent" description:"Used disk percentage"`
	Connections       int                `json:"connections" bson:"connections" description:"Connections"`
	QueryPerSecond    float64            `json:"query_per_second" bson:"query_per_second" description:"Query per second"`
	TotalQuery        uint64             `json:"total_query,omitempty" bson:"total_query,omitempty" description:"Total query"`
	CacheHitRatio     float64            `json:"cache_hit_ratio" bson:"cache_hit_ratio" description:"Cache hit ratio"`
	ReplicationLag    float64            `json:"replication_lag" bson:"replication_lag" description:"Replication lag"`
	LockWaitTime      float64            `json:"lock_wait_time" bson:"lock_wait_time" description:"Lock wait time"`
}
