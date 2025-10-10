package handlers

import (
	"fmt"
	"go-payment-app-web/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

func (h *Handler) Register(c *fiber.Ctx) error {
	user := new(models.User)
	if err := c.BodyParser(user); err != nil {
		return c.Status(400).SendString("Invalid input")
	}

	result := h.DB.Create(&user)
	if result.Error != nil {
		return c.Status(500).SendString("Error creating user")
	}

	return c.Redirect("/login")
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var input struct {
		Username string `form:"username"`
		Password string `form:"password"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).SendString("Invalid input")
	}

	var user models.User
	if err := h.DB.Where("username = ? AND password = ?", input.Username, input.Password).First(&user).Error; err != nil {
		return c.Status(401).SendString("Invalid credentials")
	}

	c.Cookie(&fiber.Cookie{
		Name:  "user_id",
		Value: fmt.Sprintf("%d", user.ID),
	})
	c.Cookie(&fiber.Cookie{
		Name:  "user_role",
		Value: user.Role,
	})

	return c.Redirect("/dashboard")
}

func (h *Handler) CreateService(c *fiber.Ctx) error {
	userID := c.Cookies("user_id")
	userRole := c.Cookies("user_role")

	if userRole != "executor" {
		return c.Status(403).SendString("Access denied")
	}

	service := new(models.Service)
	if err := c.BodyParser(service); err != nil {
		return c.Status(400).SendString("Invalid input")
	}

	service.ExecutorID = stringToUint(userID)
	result := h.DB.Create(&service)
	if result.Error != nil {
		return c.Status(500).SendString("Error creating service")
	}

	return c.Redirect("/dashboard")
}

func (h *Handler) InitiatePayment(c *fiber.Ctx) error {
	serviceID := c.Params("id")
	userID := c.Cookies("user_id")

	var service models.Service
	if err := h.DB.First(&service, serviceID).Error; err != nil {
		return c.Status(404).SendString("Service not found")
	}

	payment := models.Payment{
		ServiceID:  stringToUint(serviceID),
		CustomerID: stringToUint(userID),
		Amount:     service.Price,
		Status:     "pending",
		PaymentURL: generateRobokassaURL(service.Price, serviceID),
	}

	h.DB.Create(&payment)

	return c.Redirect(payment.PaymentURL)
}

func generateRobokassaURL(amount float64, serviceID string) string {
	return fmt.Sprintf("https://auth.robokassa.ru/Merchant/Index.aspx?MerchantLogin=your_login&OutSum=%.2f&InvId=%s", amount, serviceID)
}

func stringToUint(s string) uint {
	var i uint
	fmt.Sscanf(s, "%d", &i)
	return i
}
