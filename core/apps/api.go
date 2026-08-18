package apps

import (
	"context"
	"errors"
	"github.com/goastian/astiango-hub/core/controllers"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/middlewares"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
	"sync"
	"time"
)

func init() {
	// set gin mode
	gin.SetMode(utils.GetGinMode())
}

type Api struct {
	// internals
	app         *gin.Engine
	ln          net.Listener
	srv         *http.Server
	initialized bool
	interfaces.Logger
}

func (app *Api) Init() {
	// skip if already initialized
	if app.initialized {
		return
	}

	// initialize middlewares
	_ = app.initModuleWithApp("middlewares", middlewares.InitMiddlewares)

	// initialize routes
	_ = app.initModuleWithApp("routes", controllers.InitRoutes)

	// set initialized
	app.initialized = true
}

func (app *Api) Start() {
	// address
	address := utils.GetServerAddress()

	// http server
	app.srv = &http.Server{
		Handler:           app.GetHttpServerHandler(),
		Addr:              address,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       utils.GetAPITimeout(),
		WriteTimeout:      utils.GetAPITimeout(),
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	// listen
	var err error
	app.ln, err = net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	app.Infof("api server listening on %s", address)

	// serve
	if err := http.Serve(app.ln, app.app); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			app.Errorf("run api server error: %v", err)
		} else {
			app.Info("api server graceful down")
		}
	}
}

func (app *Api) Wait() {
	utils.DefaultWait()
}

func (app *Api) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.srv.Shutdown(ctx); err != nil {
		app.Errorf("shutdown api server error: %v", err)
	}
}

func (app *Api) GetHttpServerHandler() http.Handler {
	return controllers.GetGlobalFizzWrapper().GetFizz()
}

func (app *Api) GetGinEngine() *gin.Engine {
	return app.app
}

func (app *Api) GetHttpServer() *http.Server {
	return app.srv
}

func (app *Api) initModuleWithApp(name string, fn func(app *gin.Engine) error) (err error) {
	return initModule(name, func() error {
		return fn(app.app)
	})
}

func newApi() *Api {
	api := &Api{
		app:    gin.New(),
		Logger: utils.NewLogger("Api"),
	}
	api.Init()
	return api
}

var api *Api
var apiOnce sync.Once

func GetApi() *Api {
	apiOnce.Do(func() {
		api = newApi()
	})
	return api
}
