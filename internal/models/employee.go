package models

import "time"

// Employee represents an individual employee in the system.
// It contains the employee's ID, full name, short name, position,
// email address, phone number, and the date the record was created.
type Employee struct {
	ID        int       `json:"id"`         // Unique identifier for the employee
	FullName  string    `json:"fullname"`   // Full name of the employee
	ShortName string    `json:"shortname"`  // Short name or nickname of the employee
	Position  string    `json:"position"`   // Job position of the employee
	Email     string    `json:"email"`      // Email address of the employee
	Phone     string    `json:"phone"`      // Phone number of the employee
	IsAdmin   bool      `json:"is_admin"`   // IsAdmin returns a bool value if employee is admin
	CreatedAt time.Time `json:"created_at"` // Timestamp of when the employee record was created
}

// Customer represents an individual client in the system.
type Customer struct {
	ID       int64  `json:"id"`       // Unique identifier for the customer
	Fullname string `json:"fullname"` // Full name of the customer
	Login    string `json:"login"`    // Username of the customer
	Contract string `json:"contract"` // Contract ID og the customer
	Address  string `json:"address"`  // Address which binded to user
	Tariff   string `json:"tariff"`   // Tariff is the current tariff of the customer
}

// BotUser represents an individual user in the bot.
type BotUser struct {
	TelegramID int64 `json:"telegram_id"`
	EmployeeID int   `json:"employee_id"`
}
