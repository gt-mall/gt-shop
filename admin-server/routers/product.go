package routers

import (
	"mall/admin_core/internal/auth"
	productcategory "mall/admin_core/internal/product/category"
	productdictservice "mall/admin_core/internal/product/dict_service"
	productdictunit "mall/admin_core/internal/product/dict_unit"
	productfreight "mall/admin_core/internal/product/freight"
	productinventory "mall/admin_core/internal/product/inventory"
	productproduct "mall/admin_core/internal/product/product"
	productreview "mall/admin_core/internal/product/review"
	systemfile "mall/admin_core/internal/system/file"
	"mall/admin_core/pkg/cache"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerProductRoutes(router *gin.Engine, api *gin.RouterGroup, db *gorm.DB, authorization *auth.Service, c cache.Cache) {
	categoryService := productcategory.NewService(db, c)
	categoryAPI := productcategory.NewAPI(categoryService)

	productService := productproduct.NewService(db, c)
	productAPI := productproduct.NewAPI(productService)

	inventoryService := productinventory.NewService(db)
	inventoryAPI := productinventory.NewAPI(inventoryService)

	freightService := productfreight.NewService(db)
	freightAPI := productfreight.NewAPI(freightService)

	dictServiceService := productdictservice.NewService(db)
	dictServiceAPI := productdictservice.NewAPI(dictServiceService)

	dictUnitService := productdictunit.NewService(db)
	dictUnitAPI := productdictunit.NewAPI(dictUnitService)

	fileService := systemfile.NewService(db)
	reviewService := productreview.NewService(db, fileService, c)
	reviewAPI := productreview.NewAPI(reviewService)

	protected := api.Group("/product")
	protected.Use(authorization.Middleware())

	protected.GET("/dict-services", authorization.Permission(auth.PermissionProductDictServiceList), dictServiceAPI.List)
	protected.POST("/dict-service", authorization.Permission(auth.PermissionProductDictServiceCreate), dictServiceAPI.Create)
	protected.PUT("/dict-service/:id", authorization.Permission(auth.PermissionProductDictServiceUpdate), dictServiceAPI.Update)
	protected.DELETE("/dict-service/:id", authorization.Permission(auth.PermissionProductDictServiceDelete), dictServiceAPI.Delete)

	protected.GET("/dict-units", authorization.Permission(auth.PermissionProductDictUnitList), dictUnitAPI.List)
	protected.POST("/dict-unit", authorization.Permission(auth.PermissionProductDictUnitCreate), dictUnitAPI.Create)
	protected.PUT("/dict-unit/:id", authorization.Permission(auth.PermissionProductDictUnitUpdate), dictUnitAPI.Update)
	protected.DELETE("/dict-unit/:id", authorization.Permission(auth.PermissionProductDictUnitDelete), dictUnitAPI.Delete)

	categories := protected.Group("/categories")
	categories.POST("/list", authorization.Permission(auth.PermissionProductCategoryList), categoryAPI.List)
	categories.POST("/options", authorization.PermissionAny(
		auth.PermissionProductCategoryList,
		auth.PermissionProductProductList,
		auth.PermissionProductProductCreate,
		auth.PermissionProductProductUpdate,
	), categoryAPI.Options)
	categories.POST("/create", authorization.Permission(auth.PermissionProductCategoryCreate), categoryAPI.Create)
	categories.POST("/update", authorization.Permission(auth.PermissionProductCategoryUpdate), categoryAPI.Update)
	categories.POST("/delete", authorization.Permission(auth.PermissionProductCategoryDelete), categoryAPI.Delete)

	products := protected.Group("/products")
	products.POST("/list", authorization.Permission(auth.PermissionProductProductList), productAPI.List)
	products.POST("/tab-counts", authorization.Permission(auth.PermissionProductProductList), productAPI.TabCounts)
	products.POST("/detail", authorization.Permission(auth.PermissionProductProductList), productAPI.Detail)
	products.POST("/create", authorization.Permission(auth.PermissionProductProductCreate), productAPI.Create)
	products.POST("/update", authorization.Permission(auth.PermissionProductProductUpdate), productAPI.Update)
	products.POST("/status", authorization.Permission(auth.PermissionProductProductUpdate), productAPI.Status)
	products.POST("/delete", authorization.Permission(auth.PermissionProductProductDelete), productAPI.Delete)

	inventories := protected.Group("/inventories")
	inventories.POST("/list", authorization.Permission(auth.PermissionProductInventoryList), inventoryAPI.List)
	inventories.POST("/detail", authorization.Permission(auth.PermissionProductInventoryList), inventoryAPI.Detail)
	inventories.POST("/adjust", authorization.PermissionAll(auth.PermissionProductInventoryList, auth.PermissionProductInventoryAdjust), inventoryAPI.Adjust)
	inventories.POST("/warning", authorization.PermissionAll(auth.PermissionProductInventoryList, auth.PermissionProductInventoryWarning), inventoryAPI.Warning)
	inventories.POST("/logs/list", authorization.Permission(auth.PermissionProductInventoryLog), inventoryAPI.LogsList)

	freights := protected.Group("/freight-templates")
	freights.POST("/list", authorization.Permission(auth.PermissionProductFreightList), freightAPI.List)
	freights.POST("/options", authorization.PermissionAny(
		auth.PermissionProductFreightList,
		auth.PermissionProductProductList,
		auth.PermissionProductProductCreate,
		auth.PermissionProductProductUpdate,
	), freightAPI.Options)
	freights.POST("/detail", authorization.Permission(auth.PermissionProductFreightList), freightAPI.Detail)
	freights.POST("/create", authorization.Permission(auth.PermissionProductFreightCreate), freightAPI.Create)
	freights.POST("/update", authorization.Permission(auth.PermissionProductFreightUpdate), freightAPI.Update)
	freights.POST("/status", authorization.Permission(auth.PermissionProductFreightUpdate), freightAPI.Status)
	freights.POST("/delete", authorization.Permission(auth.PermissionProductFreightDelete), freightAPI.Delete)
	freights.POST("/regions", authorization.PermissionAny(
		auth.PermissionProductFreightList,
		auth.PermissionProductFreightCreate,
		auth.PermissionProductFreightUpdate,
	), freightAPI.Regions)

	reviews := protected.Group("/reviews")
	reviews.POST("/list", authorization.Permission(auth.PermissionProductReviewList), reviewAPI.List)
	reviews.POST("/detail", authorization.Permission(auth.PermissionProductReviewDetail), reviewAPI.Detail)
	reviews.POST("/audit", authorization.Permission(auth.PermissionProductReviewAudit), reviewAPI.Audit)
	reviews.POST("/reply", authorization.Permission(auth.PermissionProductReviewReply), reviewAPI.Reply)
}
