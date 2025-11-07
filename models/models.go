package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex"`
	Email    string `gorm:"uniqueIndex"`
	Password string
	Role     string // "customer", "executor" или "admin"
}

type Service struct {
	gorm.Model
	Title            string
	Description      string
	Price            float64
	ExecutorID       uint
	Executor         User   `gorm:"foreignKey:ExecutorID"`
	CustomerUsername string // Имя заказчика, для которого создана услуга
	Status           string `gorm:"default:'active'"`
	WorkStatus       string `gorm:"default:'not_started'"` // "not_started", "in_progress", "completed", "confirmed", "cancelled"
	PaymentReceived  bool   `gorm:"default:false"`         // Статус получения оплаты
}

type Payment struct {
	gorm.Model
	ServiceID  uint
	Service    Service `gorm:"foreignKey:ServiceID"`
	CustomerID uint
	Customer   User `gorm:"foreignKey:CustomerID"`
	Amount     float64
	Status     string // "pending", "completed", "failed"
	PaymentURL string
}
