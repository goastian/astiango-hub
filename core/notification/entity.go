package notification

import (
	"github.com/goastian/astiango-hub/core/models/models"
)

type VariableData struct {
	Task     *models.Task              `json:"task"`
	TaskStat *models.TaskStat          `json:"task_stat"`
	Spider   *models.Spider            `json:"spider"`
	Node     *models.Node              `json:"node"`
	Schedule *models.Schedule          `json:"schedule"`
	Alert    *models.NotificationAlert `json:"alert"`
	Metric   *models.Metric            `json:"metric"`
}
