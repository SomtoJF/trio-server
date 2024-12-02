package main

import (
	"github.com/somtojf/trio-server/initializers"
	"github.com/somtojf/trio-server/models"
)

func init() {
	initializers.LoadEnvVariables()
	initializers.ConnectToPostgresDB()
}

func main() {
	db := initializers.DB

	db.AutoMigrate(&models.User{}, &models.BasicChat{}, &models.ReflectionChat{})
}
