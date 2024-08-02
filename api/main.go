package main

import (
	"fmt"
	"os"
	"urlshortner/routes"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func setupRoutes(app *gin.Engine) {
	app.GET("/:url", routes.ResolveURL)
	app.POST("/api/v1", routes.ShortenURL)
}

func main() {

	err := godotenv.Load()
	if err != nil {
		fmt.Println("error loading environment variables")
	}

	r := gin.Default()
	setupRoutes(r)

	r.Run(os.Getenv("APP_PORT"))
}
