package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Houeta/radireporter-bot/internal/models"
	"github.com/jackc/pgx/v5"
)

var (
	// ErrUserNotFound is returned when an employee with the specified email is not found in the database.
	ErrUserNotFound = errors.New("employee with this email not found")
	// ErrUserAlreadyLinked is returned when an employee is already linked to a telegram account.
	ErrUserAlreadyLinked = errors.New("this employee is already linked to a telegram account")
	// ErrIDExists is returned when the specified telegram ID already exists in the database.
	ErrIDExists = errors.New("this telegram ID is already exists in the DB")
)

// LinkTelegramIDByEmail links a Telegram ID to an employee's email address in the database.
// It begins a transaction, checks if the employee exists by the provided email,
// verifies if the Telegram ID is already authenticated, and attempts to insert the
// Telegram ID and employee ID into the bot_users table. If the employee does not exist,
// or if the Telegram ID is already linked, appropriate errors are returned.
// The transaction is committed if the insertion is successful, otherwise it is rolled back.
func (r *Repository) LinkTelegramIDByEmail(ctx context.Context, telegramID int64, email string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // omitted because checking for errors will not affect the function

	var employeeID int
	err = tx.QueryRow(ctx, "SELECT id FROM employees WHERE email = $1", email).Scan(&employeeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to find employee by email: %w", err)
	}

	isExists, err := r.IsUserAuthenticated(ctx, telegramID)
	if err != nil {
		return fmt.Errorf("failed to get user by telegram ID: %w", err)
	}
	if isExists {
		return ErrIDExists
	}

	cmdTag, err := tx.Exec(
		ctx,
		"INSERT INTO bot_users (telegram_id, employee_id) VALUES ($1, $2) ON CONFLICT (employee_id) DO NOTHING",
		telegramID,
		employeeID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserAlreadyLinked
		}
		return fmt.Errorf("failed to insert into bot_users: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrUserAlreadyLinked
	}

	return tx.Commit(ctx)
}

// IsUserAuthenticated checks if a user is authenticated based on their Telegram ID.
// It returns true if the user exists in the bot_users table, and false otherwise.
// In case of an error during the database query, it returns false along with the error.
func (r *Repository) IsUserAuthenticated(ctx context.Context, telegramID int64) (bool, error) {
	var exists bool

	err := r.db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM bot_users WHERE telegram_id = $1)", telegramID).
		Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user authentication: %w", err)
	}

	return exists, nil
}

// DeleteUserByID removes a user from the bot_users table by their telegram ID.
// It takes a context and the telegram ID of the user to be deleted as parameters.
// If the deletion fails, it returns an error indicating the failure reason.
func (r *Repository) DeleteUserByID(ctx context.Context, telegramID int64) error {
	_, err := r.db.Exec(ctx, "DELETE FROM bot_users WHERE telegram_id = $1", telegramID)
	if err != nil {
		return fmt.Errorf("failed to delete user %d from bot_users: %w", telegramID, err)
	}

	return nil
}

// GetEmployee retrieves an employee's details from the database using their Telegram ID.
// It returns the employee's information as a models.Employee struct and an error if the operation fails.
//
// Parameters:
//   - ctx: The context for the database operation.
//   - telegramID: The Telegram ID of the user whose employee details are to be fetched.
//
// Returns:
//   - models.Employee: The employee details.
//   - error: An error if the retrieval fails.
func (r *Repository) GetEmployee(ctx context.Context, telegramID int64) (models.Employee, error) {
	var employee models.Employee
	query := `
		SELECT id, fullname, shortname, position, email, phone FROM employees
		WHERE id = (SELECT employee_id FROM bot_users WHERE telegram_id = $1);		
`

	err := r.db.QueryRow(ctx, query, telegramID).Scan(
		&employee.ID, &employee.FullName, &employee.ShortName, &employee.Position, &employee.Email, &employee.Phone,
	)
	if err != nil {
		return models.Employee{}, fmt.Errorf("failed to get employee data: %w", err)
	}

	return employee, nil
}
