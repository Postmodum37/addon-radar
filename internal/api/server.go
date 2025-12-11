package api

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"addon-radar/internal/database"
)

type Server struct {
	db     *database.Queries
	router *gin.Engine
}

func NewServer(db *database.Queries) *Server {
	s := &Server{
		db: db,
	}
	s.setupRouter()
	return s
}

// Router returns the Gin engine for testing
func (s *Server) Router() *gin.Engine {
	return s.router
}

func (s *Server) setupRouter() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(s.loggerMiddleware())
	r.Use(s.corsMiddleware())

	api := r.Group("/api/v1")
	{
		api.GET("/health", s.handleHealth)
		api.GET("/addons", s.handleListAddons)
		api.GET("/addons/:slug", s.handleGetAddon)
		api.GET("/addons/:slug/history", s.handleGetAddonHistory)
		api.GET("/categories", s.handleListCategories)
		api.GET("/trending/hot", s.handleTrendingHot)
		api.GET("/trending/rising", s.handleTrendingRising)
	}

	s.router = r
}

func (s *Server) Run(addr string) error {
	slog.Info("starting API server", "addr", addr)
	return s.router.Run(addr)
}

func (s *Server) loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		slog.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
		)
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
	})
}
