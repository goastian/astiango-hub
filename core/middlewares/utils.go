package middlewares

import "github.com/gin-gonic/gin"

func InitMiddlewares(app *gin.Engine) (err error) {
	// default logger
	app.Use(gin.Logger())

	// recovery from panics
	app.Use(gin.Recovery())

	app.Use(SecurityHeadersMiddleware())
	app.Use(RequestBodyLimitMiddleware())
	app.Use(AbuseProtectionMiddleware())
	app.Use(CSRFMiddleware())

	// cors
	app.Use(CORSMiddleware())

	return nil
}
