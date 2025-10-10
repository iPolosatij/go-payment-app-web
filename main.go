package main

import (
	"go-payment-app-web/config"
	"go-payment-app-web/handlers"
	"go-payment-app-web/models"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/django/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	engine := django.New("./views", ".html")
	app := fiber.New(fiber.Config{Views: engine})

	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database")
	}

	db.AutoMigrate(&models.User{}, &models.Service{}, &models.Payment{})

	cfg := config.Load()
	handler := handlers.NewHandler(db)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{})
	})

	app.Get("/login", func(c *fiber.Ctx) error {
		return c.Render("login", fiber.Map{})
	})

	app.Get("/dashboard", func(c *fiber.Ctx) error {
		var services []models.Service
		db.Find(&services)
		return c.Render("dashboard", fiber.Map{"Services": services})
	})

	app.Post("/register", handler.Register)
	app.Post("/login", handler.Login)
	app.Post("/services", handler.CreateService)
	app.Get("/payment/:id", handler.InitiatePayment)

	log.Fatal(app.Listen(cfg.ServerPort))
}
