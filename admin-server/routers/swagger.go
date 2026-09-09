package routers

import (
	"net/http"

	docsMiniapp "mall/admin_core/docs-miniapp"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func registerSwaggerRoutes(r *gin.Engine) {
	// Admin Swagger UI
	r.GET("/swagger/admin/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("swagger")))

	// Miniapp Swagger JSON (public documentation endpoint for miniapp client code generation)
	r.GET("/swagger/miniapp/swagger.json", func(c *gin.Context) {
		doc := docsMiniapp.SwaggerInfo.ReadDoc()
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(doc))
	})

	// Redirect /swagger and /swagger/index.html to /swagger/admin/index.html
	r.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/admin/index.html")
	})
	r.GET("/swagger/index.html", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/admin/index.html")
	})
}
