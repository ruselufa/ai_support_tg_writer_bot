package web

import (
	"ai_support_tg_writer_bot/internal/config"
	"ai_support_tg_writer_bot/internal/service"
	"log"

	"github.com/gin-gonic/gin"
)

type Server struct {
	router   *gin.Engine
	handlers *WebHandlers
	config   *config.Config
}

func NewServer(config *config.Config, userService service.UserService, ticketService service.TicketService, fileService service.FileService) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	handlers := NewWebHandlers(userService, ticketService, fileService, config)

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Admin-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Публичные маршруты
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		// Админские маршруты
		admin := api.Group("/admin")
		admin.Use(handlers.AdminAuthMiddleware())
		{
			admin.GET("/dashboard", handlers.AdminDashboard)
			admin.GET("/tickets", handlers.GetTickets)
			admin.GET("/tickets/:id", handlers.GetTicket)
			admin.POST("/tickets/:id/reply", handlers.ReplyToTicket)
			admin.POST("/tickets/:id/close", handlers.CloseTicket)
			admin.GET("/stats", handlers.GetStats)
		}
	}

	// Статические файлы для админской панели
	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/templates/*")

	// Главная страница админки
	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "admin.html", gin.H{
			"title": "Social Flow Support Admin",
		})
	})

	return &Server{
		router:   router,
		handlers: handlers,
		config:   config,
	}
}

func (s *Server) Start() error {
	log.Printf("Starting web server on port %s", s.config.ServerPort)
	return s.router.Run(":" + s.config.ServerPort)
}
