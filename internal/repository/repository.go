package repository

import (
	"context"
	"time"

	"github.com/UnknownOlympus/oracle/internal/models"
)

type Repository struct {
	db Database
}

// BotManager defines the interface for repository operations related to user authentication
// and management. It includes methods for linking a Telegram ID to an email, checking user
// authentication status, and deleting a user by their Telegram ID.
type BotManager interface {
	LinkTelegramIDByEmail(ctx context.Context, telegramID int64, email string) error
	IsUserAuthenticated(ctx context.Context, telegramID int64) (bool, error)
	DeleteUserByID(ctx context.Context, telegramID int64) error
	IsAdmin(ctx context.Context, telegramID int64) (bool, error)
	GetAllTgUserIDs(ctx context.Context) ([]int64, error)
	GetAdmins(ctx context.Context) ([]models.BotUser, error)
	SetUserLanguage(ctx context.Context, telegramID int64, langCode string) error
	GetUserLanguage(ctx context.Context, telegramID int64) (string, error)
}

// TaskManager defines the interface for repository operations related to task management.
// It includes methods for get employee, get tasks with different status, etc.
type TaskManager interface {
	GetEmployee(ctx context.Context, telegramID int64) (models.Employee, error)
	GetTaskSummary(ctx context.Context, telegramID int64, startDate, endDate time.Time) ([]models.TaskSummary, error)
	GetActiveTasksByExecutor(ctx context.Context, telegramID int64) ([]models.ActiveTask, error)
	GetTaskDetailsByID(ctx context.Context, taskID int) (*models.TaskDetails, error)
	GetCompletedTasksByExecutor(ctx context.Context, telegramID int64, from, to time.Time) ([]models.TaskDetails, error)
	GetTasksInRadius(ctx context.Context, lat, lng float32, radius int) ([]models.ActiveTask, error)
	GetCustomersByTaskID(ctx context.Context, taskID int64) ([]models.Customer, error)
}

// NewRepository creates a new instance of Repository with the provided Database.
// It returns a pointer to the newly created Repository.
func NewRepository(db Database) *Repository {
	return &Repository{db: db}
}
