package repository

import (
	"context"
)

type Repository struct {
	db Database
}

// Interface defines the interface for repository operations related to user authentication
// and management. It includes methods for linking a Telegram ID to an email, checking user
// authentication status, and deleting a user by their Telegram ID.
type Interface interface {
	LinkTelegramIDByEmail(ctx context.Context, telegramID int64, email string) error
	IsUserAuthenticated(ctx context.Context, telegramID int64) (bool, error)
	DeleteUserByID(ctx context.Context, telegramID int64) error
	// GetEmployeesForReport(ctx context.Context, from, to time.Time) ([]models.Employee, error)
}

// NewRepository creates a new instance of Repository with the provided Database.
// It returns a pointer to the newly created Repository.
func NewRepository(db Database) *Repository {
	return &Repository{db: db}
}
