package models

import "time"

// Employee represents an individual employee in the system.
// It contains the employee's ID, full name, short name, position,
// email address, phone number, and the date the record was created.
type Employee struct {
	ID        int       // Unique identifier for the employee
	FullName  string    // Full name of the employee
	ShortName string    // Short name or nickname of the employee
	Position  string    // Job position of the employee
	Email     string    // Email address of the employee
	Phone     string    // Phone number of the employee
	CreatedAt time.Time // Timestamp of when the employee record was created
}
