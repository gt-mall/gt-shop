package routers

import (
	"mall/admin_core/internal/auth"
	productcategory "mall/admin_core/internal/product/category"
	productfreight "mall/admin_core/internal/product/freight"
	productinventory "mall/admin_core/internal/product/inventory"
	productproduct "mall/admin_core/internal/product/product"
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

	protected := api.Group("/product")
	protected.Use(authorization.Middleware())

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
}
