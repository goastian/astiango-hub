package utils

import (
	"fmt"
	"net/http"
)

func HandleHealthFn(healthFn func() bool, healthPort int) {
	addr := fmt.Sprintf(":%d", healthPort)
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			logger.Errorf("health check server failed: %v", err)
		}
	}()
	logger.Infof("health check server started on port %d", healthPort)
}
