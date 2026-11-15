package routes

import (
	"github.com/gin-gonic/gin"

	"industrial_admin/config"
	"industrial_admin/controller"
	"industrial_admin/db"
	"industrial_admin/middleware"
	"industrial_admin/response"
)

// New 组装路由树：公开组（登录）与受保护组（业务）分离
func New(cfg *config.Config) *gin.Engine {
	if cfg.App.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), middleware.Recovery(), middleware.CORS())
	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, response.CodeNotFound, "接口不存在")
	})

	authCtl := controller.NewAuthController(cfg, db.DB)
	userCtl := controller.NewUserController(db.DB)
	roleCtl := controller.NewRoleController(db.DB)
	deptCtl := controller.NewDeptController(db.DB)
	cfgCtl := controller.NewConfigController(db.DB)
	dashCtl := controller.NewDashboardController(db.DB)
	logCtl := controller.NewLogController(db.DB)
	permCtl := controller.NewPermissionController(db.DB)

	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", authCtl.Login)
	}

	// 受保护组：先鉴权，再按接口挂权限点，最后统一留痕
	priv := api.Group("")
	priv.Use(middleware.Auth(cfg), middleware.OperationLog(db.DB))
	{
		priv.GET("/auth/profile", authCtl.Profile)

		users := priv.Group("/users")
		{
			users.GET("", middleware.RequirePerm("api:user:list"), userCtl.List)
			users.POST("", middleware.RequirePerm("user:create"), userCtl.Create)
			users.PUT("/:id", middleware.RequirePerm("user:update"), userCtl.Update)
			users.DELETE("/:id", middleware.RequirePerm("user:delete"), userCtl.Delete)
			users.POST("/:id/password", middleware.RequirePerm("user:resetpwd"), userCtl.ResetPassword)
			// 用户角色绑定：授权类操作，复用 role:grant 权限点
			users.GET("/:id/roles", middleware.RequirePerm("role:grant"), userCtl.Roles)
			users.PUT("/:id/roles", middleware.RequirePerm("role:grant"), userCtl.BindRoles)
		}

		roles := priv.Group("/roles")
		{
			roles.GET("", middleware.RequirePerm("sys:role"), roleCtl.List)
			roles.POST("", middleware.RequirePerm("role:grant"), roleCtl.Create)
			roles.PUT("/:id", middleware.RequirePerm("role:grant"), roleCtl.Update)
			roles.DELETE("/:id", middleware.RequirePerm("role:grant"), roleCtl.Delete)
			roles.GET("/:id/permissions", middleware.RequirePerm("role:grant"), roleCtl.Permissions)
			roles.PUT("/:id/permissions", middleware.RequirePerm("role:grant"), roleCtl.Grant)
		}

		// 权限点管理：复用 role:grant 权限码，避免改动初始化权限数据
		perms := priv.Group("/permissions")
		{
			perms.GET("", middleware.RequirePerm("role:grant"), permCtl.List)
			perms.POST("", middleware.RequirePerm("role:grant"), permCtl.Create)
			perms.PUT("/:id", middleware.RequirePerm("role:grant"), permCtl.Update)
			perms.DELETE("/:id", middleware.RequirePerm("role:grant"), permCtl.Delete)
		}

		depts := priv.Group("/depts")
		{
			depts.GET("", middleware.RequirePerm("dept:create"), deptCtl.List)
			depts.GET("/tree", deptCtl.Tree)
			depts.GET("/:id", middleware.RequirePerm("dept:create"), deptCtl.Detail)
			depts.POST("", middleware.RequirePerm("dept:create"), deptCtl.Create)
			depts.PUT("/:id", middleware.RequirePerm("dept:create"), deptCtl.Update)
			depts.DELETE("/:id", middleware.RequirePerm("dept:create"), deptCtl.Remove)
		}

		configs := priv.Group("/configs")
		{
			configs.GET("", cfgCtl.List)
			configs.PUT("", middleware.RequirePerm("config:update"), cfgCtl.BatchUpdate)
			configs.GET("/:id", cfgCtl.Detail)
			configs.PUT("/:id", middleware.RequirePerm("config:update"), cfgCtl.Update)
		}

		priv.GET("/dashboard/overview", dashCtl.Overview)
		priv.GET("/logs", middleware.RequirePerm("audit:log"), logCtl.List)
	}

	return r
}
