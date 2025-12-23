package router

import (
	"strconv"
	"time"

	"eventplanner-backend/internal/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func New(auth *handlers.AuthHandler, events *handlers.EventHandler, search *handlers.SearchHandler) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-User-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Use(func(c *gin.Context) {
		if h := c.GetHeader("X-User-ID"); h != "" {
			if id, err := strconv.Atoi(h); err == nil {
				c.Set("userID", id)
			}
		}
		c.Next()
	})

	r.POST("/signup", auth.Signup)
	r.POST("/login", auth.Login)
	r.GET("/health", auth.Health)
	// Events
	r.POST("/events", events.Create)
	r.GET("/events/organized", events.ListOrganized)
	r.GET("/events/invited", events.ListInvited)
	r.POST("/events/:id/invite", events.Invite)
	r.DELETE("/events/:id", events.Delete)
	r.GET("/events/:id/attendees", events.Participants)
	r.PUT("/events/:id/attendance", events.SetAttendance)
	r.PUT("/events/:id/accept", events.AcceptInvite)
	r.POST("/events/:id/tasks", events.CreateTask)

	r.GET("/search", search.Search)

	return r
}
