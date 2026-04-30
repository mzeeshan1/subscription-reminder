package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"subscription-manager/cache"
	"subscription-manager/config"
	"subscription-manager/db"
	"subscription-manager/handlers"
	"subscription-manager/middleware"
	"subscription-manager/notifications"
	"subscription-manager/worker"
)

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	if err := db.Migrate(database); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	redisCache, err := cache.New(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis connect: %v", err)
	}

	notifier := notifications.New(cfg)

	w := worker.New(database, redisCache, notifier)
	w.Start()
	defer w.Stop()

	router := gin.Default()
	router.LoadHTMLGlob("templates/*.html")
	router.Static("/static", "./static")

	authH := handlers.NewAuthHandler(database, redisCache, cfg)
	subH := handlers.NewSubscriptionHandler(database, redisCache)
	settingsH := handlers.NewSettingsHandler(database, redisCache)

	authMW := middleware.Auth(cfg, redisCache)
	rateMW := middleware.LoginRateLimit(redisCache)

	router.GET("/login", authH.LoginPage)
	router.POST("/login", rateMW, authH.Login)
	router.GET("/register", authH.RegisterPage)
	router.POST("/register", authH.Register)

	app := router.Group("/")
	app.Use(authMW)
	{
		app.GET("/", subH.Dashboard)
		app.GET("/subscriptions/add", subH.AddPage)
		app.POST("/subscriptions", subH.Create)
		app.GET("/subscriptions/:id/edit", subH.EditPage)
		app.POST("/subscriptions/:id", subH.Update)
		app.POST("/subscriptions/:id/delete", subH.Delete)
		app.GET("/settings", settingsH.SettingsPage)
		app.POST("/settings", settingsH.UpdateSettings)
		app.POST("/logout", authH.Logout)
	}

	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(router.Run(":" + cfg.Port))
}
