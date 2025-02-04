package main

import (
	"log"

	"github.com/somtojf/trio-server/initializers"
	"github.com/somtojf/trio-server/models"
)

func init() {
	initializers.LoadEnvVariables()
	initializers.ConnectToPostgresDB()
}

func main() {
	db := initializers.DB

	error := db.AutoMigrate(&models.User{}, &models.BasicChat{}, &models.ReflectionChat{}, &models.BasicAgent{}, &models.BasicMessage{}, &models.Reflection{}, &models.ReflectionMessage{}, &models.EvaluatorMessage{})

	if error != nil {
		log.Fatal("Error migrating database: ", error)
	}
	log.Println("Database migrated successfully")
}
