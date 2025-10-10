package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex"`
	Email    string `gorm:"uniqueIndex"`
	Password string
	Role     string // "customer" или "executor"
}

type Service struct {
	gorm.Model
	Title       string
	Description string
	Price       float64
	ExecutorID  uint
	Executor    User   `gorm:"foreignKey:ExecutorID"`
	Status      string `gorm:"default:'active'"`
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
