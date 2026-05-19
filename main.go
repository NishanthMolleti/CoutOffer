package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"coutoffer/handlers"
	"coutoffer/models"
	"coutoffer/scheduler"
)

func main() {
	godotenv.Load()

	db, err := gorm.Open(sqlite.Open("jobs.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal("db open failed:", err)
	}
	db.AutoMigrate(&models.Subscription{}, &models.SeenJob{})

	c := cron.New()
	c.AddFunc("@every 1h", func() {
		log.Println("// cron: checking jobs...")
		scheduler.CheckJobs(db)
	})
	c.Start()
	defer c.Stop()

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	h := handlers.New(db)
	r.GET("/", h.Index)
	r.POST("/subscribe", h.Subscribe)
	r.POST("/unsubscribe/:id", h.Unsubscribe)
	r.POST("/check", h.ManualCheck)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("// cout << offer running on :%s", port)
	r.Run(":" + port)
}
