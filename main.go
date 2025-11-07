package main

import (
	"fmt"
	"go-payment-app-web/config"
	"go-payment-app-web/handlers"
	"go-payment-app-web/models"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/django/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	engine := django.New("./views", ".html")

	// Добавляем кастомные функции для шаблонов
	engine.AddFunc("formatPrice", func(price float64) string {
		return fmt.Sprintf("%.2f ₽", price)
	})

	app := fiber.New(fiber.Config{Views: engine})

	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database")
	}

	db.AutoMigrate(&models.User{}, &models.Service{}, &models.Payment{})

	// Создаем администратора при запуске
	createAdminUser(db)

	cfg := config.Load()
	handler := handlers.NewHandler(db)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{})
	})

	app.Get("/login", func(c *fiber.Ctx) error {
		return c.Render("login", fiber.Map{})
	})

	app.Get("/dashboard", func(c *fiber.Ctx) error {
		userID := c.Cookies("user_id")
		userRole := c.Cookies("user_role")
		username := c.Cookies("username")
		searchQuery := c.Query("search")

		var services []models.Service

		if userRole == "executor" {
			// Исполнитель видит только свои услуги
			db.Preload("Executor").Where("executor_id = ?", userID).Find(&services)
		} else if userRole == "customer" {
			// Заказчик видит только услуги, созданные для него
			db.Preload("Executor").Where("customer_username = ?", username).Find(&services)
		} else if userRole == "admin" {
			// Админ видит все услуги с возможностью поиска
			query := db.Preload("Executor")
			if searchQuery != "" {
				// Пробуем преобразовать поисковый запрос в число (поиск по ID)
				if id, err := strconv.Atoi(searchQuery); err == nil {
					query = query.Where("id = ?", id)
				} else {
					// Если не число, ищем по текстовым полям
					query = query.Where("title LIKE ? OR description LIKE ? OR customer_username LIKE ?",
						"%"+searchQuery+"%", "%"+searchQuery+"%", "%"+searchQuery+"%")
				}
			}
			query.Find(&services)
		}

		return c.Render("dashboard", fiber.Map{
			"Services":     services,
			"user_id":      userID,
			"user_role":    userRole,
			"username":     username,
			"search_query": searchQuery,
		})
	})

	app.Post("/register", handler.Register)
	app.Post("/login", handler.Login)
	app.Post("/services", handler.CreateService)
	app.Get("/payment/:id", handler.InitiatePayment)
	app.Post("/service/:id/status", handler.UpdateWorkStatus)
	app.Post("/service/:id/receive-payment", handler.ReceivePayment)
	app.Post("/service/:id/admin-payment-received", handler.AdminMarkPaymentReceived)

	log.Fatal(app.Listen(cfg.ServerPort))
}

func createAdminUser(db *gorm.DB) {
	var adminUser models.User
	result := db.Where("username = ?", "admin").First(&adminUser)

	if result.Error != nil {
		// Администратора нет, создаем его
		admin := models.User{
			Username: "admin",
			Email:    "admin@paymentapp.com",
			Password: "wahadmin",
			Role:     "admin",
		}
		db.Create(&admin)
		log.Println("Admin user created successfully")
	} else {
		log.Println("Admin user already exists")
	}
}
