package apps

import (
	"github.com/goastian/astiango-hub/core/utils"
)

var logger = utils.NewLogger("Apps")

func Start(app App) {
	start(app)
}

func start(app App) {
	app.Init()
	go app.Start()
	app.Wait()
	app.Stop()
}

func initModule(name string, fn func() error) (err error) {
	if err := fn(); err != nil {
		logger.Errorf("init %s error: %v", name, err)
		panic(err)
	}
	logger.Infof("initialized %s successfully", name)
	return nil
}
