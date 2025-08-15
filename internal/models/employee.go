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
	CreatedAt time.Time `json:"created_at"` // Timestamp of when the employee record was created
}
