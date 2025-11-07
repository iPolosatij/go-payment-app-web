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
	c.Cookie(&fiber.Cookie{
		Name:  "username",
		Value: user.Username,
	})

	return c.Redirect("/dashboard")
}

func (h *Handler) CreateService(c *fiber.Ctx) error {
	userID := c.Cookies("user_id")
	userRole := c.Cookies("user_role")

	if userRole != "executor" {
		return c.Status(403).SendString("Access denied")
	}

	// Используем структуру для парсинга формы
	var input struct {
		Title            string  `form:"title"`
		Description      string  `form:"description"`
		Price            float64 `form:"price"`
		CustomerUsername string  `form:"customer_username"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).SendString("Invalid input: " + err.Error())
	}

	// Проверяем, существует ли заказчик с указанным username
	var customer models.User
	if err := h.DB.Where("username = ? AND role = ?", input.CustomerUsername, "customer").First(&customer).Error; err != nil {
		return c.Status(400).SendString("Customer with username '" + input.CustomerUsername + "' not found")
	}

	// Создаем услугу с данными из формы
	service := &models.Service{
		Title:            input.Title,
		Description:      input.Description,
		Price:            input.Price,
		CustomerUsername: input.CustomerUsername,
		ExecutorID:       stringToUint(userID),
		WorkStatus:       "not_started",
		PaymentReceived:  false,
	}

	result := h.DB.Create(&service)
	if result.Error != nil {
		return c.Status(500).SendString("Error creating service: " + result.Error.Error())
	}

	return c.Redirect("/dashboard")
}

func (h *Handler) InitiatePayment(c *fiber.Ctx) error {
	serviceID := c.Params("id")
	userID := c.Cookies("user_id")
	username := c.Cookies("username")

	var service models.Service
	if err := h.DB.First(&service, serviceID).Error; err != nil {
		return c.Status(404).SendString("Service not found")
	}

	// Проверяем, что услуга предназначена для текущего пользователя
	if service.CustomerUsername != username {
		return c.Status(403).SendString("This service is not for you")
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

func (h *Handler) UpdateWorkStatus(c *fiber.Ctx) error {
	serviceID := c.Params("id")
	userRole := c.Cookies("user_role")

	var service models.Service
	if err := h.DB.First(&service, serviceID).Error; err != nil {
		return c.Status(404).SendString("Service not found")
	}

	// Админ не может менять статусы работы, только статус оплаты
	if userRole == "admin" {
		return c.Status(403).SendString("Admin can only mark payments as received")
	}

	var input struct {
		WorkStatus string `form:"work_status"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).SendString("Invalid input")
	}

	// Единственное ограничение: только заказчик может подтверждать
	if input.WorkStatus == "confirmed" && userRole != "customer" {
		return c.Status(403).SendString("Only customer can confirm completion")
	}

	// Обновляем статус работы
	h.DB.Model(&service).Update("work_status", input.WorkStatus)

	return c.Redirect("/dashboard")
}

// Добавим новый обработчик для получения оплаты
func (h *Handler) ReceivePayment(c *fiber.Ctx) error {
	serviceID := c.Params("id")
	userID := c.Cookies("user_id")
	userRole := c.Cookies("user_role")

	if userRole != "executor" {
		return c.Status(403).SendString("Access denied")
	}

	var service models.Service
	if err := h.DB.First(&service, serviceID).Error; err != nil {
		return c.Status(404).SendString("Service not found")
	}

	// Проверяем, что услуга принадлежит исполнителю
	if fmt.Sprintf("%d", service.ExecutorID) != userID {
		return c.Status(403).SendString("This service doesn't belong to you")
	}

	// Проверяем, что статус "confirmed"
	if service.WorkStatus != "confirmed" {
		return c.Status(400).SendString("Service must be confirmed by customer before receiving payment")
	}

	// TODO: Добавить логику получения оплаты
	// Пока просто перенаправляем обратно
	return c.Redirect("/dashboard")
}

// Новый обработчик для администратора - отметить оплату полученной
func (h *Handler) AdminMarkPaymentReceived(c *fiber.Ctx) error {
	serviceID := c.Params("id")
	userRole := c.Cookies("user_role")

	if userRole != "admin" {
		return c.Status(403).SendString("Access denied")
	}

	var service models.Service
	if err := h.DB.First(&service, serviceID).Error; err != nil {
		return c.Status(404).SendString("Service not found")
	}

	// Помечаем оплату как полученную
	h.DB.Model(&service).Update("payment_received", true)

	return c.Redirect("/dashboard")
}

func generateRobokassaURL(amount float64, serviceID string) string {
	return fmt.Sprintf("https://auth.robokassa.ru/Merchant/Index.aspx?MerchantLogin=your_login&OutSum=%.2f&InvId=%s", amount, serviceID)
}

func stringToUint(s string) uint {
	var i uint
	fmt.Sscanf(s, "%d", &i)
	return i
}
