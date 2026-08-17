package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Metric struct {
	any                  `collection:"metrics"`
	BaseModel            `bson:",inline"`
	Type                 string             `json:"type" bson:"type" description:"Type"`
	NodeId               primitive.ObjectID `json:"node_id" bson:"node_id" description:"Node ID"`
	CpuUsagePercent      float32            `json:"cpu_usage_percent" bson:"cpu_usage_percent" description:"CPU usage percentage"`
	TotalMemory          uint64             `json:"total_memory" bson:"total_memory" description:"Total memory"`
	AvailableMemory      uint64             `json:"available_memory" bson:"available_memory" description:"Available memory"`
	UsedMemory           uint64             `json:"used_memory" bson:"used_memory" description:"Used memory"`
	UsedMemoryPercent    float32            `json:"used_memory_percent" bson:"used_memory_percent" description:"Used memory percentage"`
	TotalDisk            uint64             `json:"total_disk" bson:"total_disk" description:"Total disk"`
	AvailableDisk        uint64             `json:"available_disk" bson:"available_disk" description:"Available disk"`
	UsedDisk             uint64             `json:"used_disk" bson:"used_disk" description:"Used disk"`
	UsedDiskPercent      float32            `json:"used_disk_percent" bson:"used_disk_percent" description:"Used disk percentage"`
	DiskReadBytesRate    float32            `json:"disk_read_bytes_rate" bson:"disk_read_bytes_rate" description:"Disk read bytes rate"`
	DiskWriteBytesRate   float32            `json:"disk_write_bytes_rate" bson:"disk_write_bytes_rate" description:"Disk write bytes rate"`
	NetworkBytesSentRate float32            `json:"network_bytes_sent_rate" bson:"network_bytes_sent_rate" description:"Network bytes sent rate"`
	NetworkBytesRecvRate float32            `json:"network_bytes_recv_rate" bson:"network_bytes_recv_rate" description:"Network bytes recv rate"`
	GoroutineCount       int32              `json:"goroutine_count" bson:"goroutine_count" description:"Current goroutine count"`
}
