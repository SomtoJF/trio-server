package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/somtojf/trio-server/aipi/googlegenai"
	"github.com/somtojf/trio-server/common"
	"github.com/somtojf/trio-server/controllers/auth"
	basicchat "github.com/somtojf/trio-server/controllers/basic-chat"
	basicmessage "github.com/somtojf/trio-server/controllers/basic-chat/basic-message"
	reflectionchat "github.com/somtojf/trio-server/controllers/reflection-chat"
	reflectionmessage "github.com/somtojf/trio-server/controllers/reflection-chat/reflection-message"
	"github.com/somtojf/trio-server/initializers"
	authcheck "github.com/somtojf/trio-server/middleware/auth-check"
)

func init() {

	initializers.LoadEnvVariables()
	initializers.ConnectToPostgresDB()
	initializers.ConnectToQdrant()

	googlegenai.CreateClient(context.Background())
}

func main() {
	r := gin.Default()
	clientAddress := os.Getenv("CLIENT_ADDRESS")

	authCheckMiddleware := authcheck.NewMiddleware(initializers.DB)
	authEndpoint := auth.NewEndpoint(initializers.DB)
	basicChatEndpoint := basicchat.NewEndpoint(initializers.DB)
	reflectionChatEndpoint := reflectionchat.NewEndpoint(initializers.DB)

	deps, err := common.NewDependencies(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	reflectionMessageEndpoint := reflectionmessage.NewEndpoint(initializers.DB, deps.AIPIProvider, initializers.QdrantClient)
	basicMessageEndpoint := basicmessage.NewEndpoint(initializers.DB, deps.AIPIProvider, initializers.QdrantClient)

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{clientAddress}
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}

	r.Use(cors.New(config))

	public := r.Group("/")
	{
		public.POST("/login", authEndpoint.Login)
		public.POST("/signup", authEndpoint.Signup)
		public.POST("/login/guest", authEndpoint.GuestLogin)

		public.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "OK"})
		})
	}

	authenticated := r.Group("/")
	authenticated.Use(authCheckMiddleware.AuthCheck())
	{
		authenticated.POST("/logout", authEndpoint.Logout)
		authenticated.POST("/reset-password", authEndpoint.ResetPassword)
		authenticated.GET("/completions", authEndpoint.GetCurrentUser)
		authenticated.GET("/me", authEndpoint.GetCurrentUser)

		reflectionChats := authenticated.Group("/reflection-chats")
		{
			reflectionChats.GET("/", reflectionChatEndpoint.GetReflectionChats)
			reflectionChats.POST("/", reflectionChatEndpoint.CreateReflectionChat)
			reflectionChats.DELETE("/:id", reflectionChatEndpoint.DeleteReflectionChat)
			reflectionChats.POST("/:id/messages", reflectionMessageEndpoint.SendMessage)
			reflectionChats.GET("/:id", reflectionChatEndpoint.GetReflectionChat)
			// This gets reflections rather than messages
			reflectionChats.GET("/:id/reflections", reflectionChatEndpoint.GetChatReflections)
		}

		basicChats := authenticated.Group("/basic-chats")
		{
			basicChats.GET("/", basicChatEndpoint.GetBasicChats)
			basicChats.POST("/", basicChatEndpoint.CreateBasicChat)
			basicChats.GET("/:id", basicChatEndpoint.GetBasicChat)
			basicChats.PUT("/:id", basicChatEndpoint.UpdateBasicChat)
			basicChats.DELETE("/:id", basicChatEndpoint.DeleteBasicChat)
			basicChats.POST("/:id/messages", basicMessageEndpoint.SendBasicMessage)
			basicChats.GET("/:id/messages", basicMessageEndpoint.GetBasicMessages)
		}

	}

	r.Run() // listen and serve on 0.0.0.0:4000
}
