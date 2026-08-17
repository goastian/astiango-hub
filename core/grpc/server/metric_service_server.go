package server

import (
	"context"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/goastian/astiango-hub/grpc"
	"go.mongodb.org/mongo-driver/bson"
)

type MetricServiceServer struct {
	grpc.UnimplementedMetricServiceServer
	interfaces.Logger
}

func (svr MetricServiceServer) Send(_ context.Context, req *grpc.MetricServiceSendRequest) (res *grpc.Response, err error) {
	svr.Debugf("received metric from node: " + req.NodeKey)
	n, err := service.NewModelService[models.Node]().GetOne(bson.M{"key": req.NodeKey}, nil)
	if err != nil {
		svr.Errorf("failed to get node: %v", err)
		return HandleError(err)
	}
	metric := models.Metric{
		Type:                 req.Type,
		NodeId:               n.Id,
		CpuUsagePercent:      req.CpuUsagePercent,
		TotalMemory:          req.TotalMemory,
		AvailableMemory:      req.AvailableMemory,
		UsedMemory:           req.UsedMemory,
		UsedMemoryPercent:    req.UsedMemoryPercent,
		TotalDisk:            req.TotalDisk,
		AvailableDisk:        req.AvailableDisk,
		UsedDisk:             req.UsedDisk,
		UsedDiskPercent:      req.UsedDiskPercent,
		DiskReadBytesRate:    req.DiskReadBytesRate,
		DiskWriteBytesRate:   req.DiskWriteBytesRate,
		NetworkBytesSentRate: req.NetworkBytesSentRate,
		NetworkBytesRecvRate: req.NetworkBytesRecvRate,
		GoroutineCount:       req.GoroutineCount,
	}
	metric.CreatedAt = time.Unix(req.Timestamp, 0)
	_, err = service.NewModelService[models.Metric]().InsertOne(metric)
	if err != nil {
		svr.Errorf("failed to insert metric: %v", err)
		return HandleError(err)
	}
	return HandleSuccess()
}

func newMetricsServer() *MetricServiceServer {
	return &MetricServiceServer{
		Logger: utils.NewLogger("MetricServiceServer"),
	}
}

var metricsServer *MetricServiceServer
var metricsServerOnce = &sync.Once{}

func GetMetricsServer() *MetricServiceServer {
	metricsServerOnce.Do(func() {
		metricsServer = newMetricsServer()
	})
	return metricsServer
}
