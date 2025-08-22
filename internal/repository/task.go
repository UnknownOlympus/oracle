package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/UnknownOlympus/oracle/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// GetTaskSummary retrieves a summary of tasks for a specific user identified by telegramID
// within the given date range defined by startDate and endDate. It returns a slice of
// TaskSummary models and an error if any occurs during the database query or scanning process.
func (r *Repository) GetTaskSummary(ctx context.Context, telegramID int64, startDate, endDate time.Time) (
	[]models.TaskSummary, error,
) {
	var err error
	var summaries []models.TaskSummary

	rows, err := r.db.Query(ctx, GetTaskSummarySQL, telegramID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("error querying task summaries: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var summary models.TaskSummary
		err = rows.Scan(&summary.Type, &summary.Count)
		if err != nil {
			return nil, fmt.Errorf("error scanning summaries row: %w", err)
		}
		summaries = append(summaries, summary)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterating summary rows: %w", err)
	}

	return summaries, nil
}

// GetActiveTasksByExecutor retrieves a list of active tasks assigned to a specific executor.
// It queries the database for tasks that are not closed and are associated with the given
// Telegram ID of the executor. The results are ordered by the task creation date in descending order.
//
// Parameters:
//   - ctx: The context for the database query.
//   - telegramID: The Telegram ID of the executor whose active tasks are to be retrieved.
//
// Returns:
//   - A slice of ActiveTask models representing the active tasks for the specified executor.
//   - An error if the query fails or if there is an issue scanning the results.
func (r *Repository) GetActiveTasksByExecutor(ctx context.Context, telegramID int64) ([]models.ActiveTask, error) {
	query := `
		SELECT t.task_id, t.description
		FROM tasks t
		JOIN task_executors te ON t.task_id = te.task_id
		JOIN bot_users bu ON te.executor_id = bu.employee_id
		WHERE bu.telegram_id = $1 AND t.is_closed = FALSE
		ORDER BY t.creation_date DESC;
	`
	rows, err := r.db.Query(ctx, query, telegramID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.ActiveTask
	for rows.Next() {
		var task models.ActiveTask
		if errScan := rows.Scan(&task.ID, &task.Description); errScan != nil {
			return nil, fmt.Errorf("failed to scan active task row: %w", errScan)
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}

	return tasks, nil
}

// GetCompletedTasksByExecutor retrieves completed tasks for a specific executor
// identified by their Telegram ID within a specified date range. It returns a slice
// of TaskDetails and an error if any occurs during the query execution.
//
// Parameters:
// - ctx: The context for managing request-scoped values and cancellation signals.
// - telegramID: The Telegram ID of the executor whose completed tasks are to be fetched.
// - from: The start date of the range for filtering completed tasks.
// - to: The end date of the range for filtering completed tasks.
//
// Returns:
// - []models.TaskDetails: A slice containing the details of completed tasks.
// - error: An error if the query fails or if there is an issue scanning the results.
func (r *Repository) GetCompletedTasksByExecutor(
	ctx context.Context,
	telegramID int64,
	from, to time.Time,
) ([]models.TaskDetails, error) {
	query := `
		SELECT
			t.task_id,
			tt.type_name,
			t.creation_date,
			t.closing_date,
			t.description,
			t.address,
			ARRAY_AGG(DISTINCT c.name) FILTER (WHERE c.name IS NOT NULL) AS customer_names,
			t.comments
		FROM tasks t
		JOIN task_executors te ON t.task_id = te.task_id
		JOIN bot_users bu ON te.executor_id = bu.employee_id
		JOIN task_types tt ON t.task_type_id = tt.type_id
		LEFT JOIN task_customers tc ON t.task_id = tc.task_id
		LEFT JOIN customers c ON tc.customer_id = c.id
		WHERE
			bu.telegram_id = $1
			AND t.closing_date >= $2
			AND t.closing_date <= $3
			AND t.is_closed = TRUE
		GROUP BY t.task_id, tt.type_name
		ORDER BY tt.type_name, t.creation_date;
	`
	rows, err := r.db.Query(ctx, query, telegramID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query completed tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.TaskDetails
	for rows.Next() {
		var task models.TaskDetails
		if err = rows.Scan(&task.ID, &task.Type, &task.CreationDate, &task.ClosingDate, &task.Description,
			&task.Address, &task.CustomerNames, &task.Comments,
		); err != nil {
			return nil, fmt.Errorf("failed to scan completed task row: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}

	return tasks, nil
}

// GetTaskDetailsByID retrieves the details of a task by its ID.
// It executes a SQL query to fetch task details including type, creation date,
// description, address, customer name, and comments. If the task is not found,
// it returns an error indicating that the task with the specified ID does not exist.
// In case of any other query errors, it returns an error with the details of the failure.
//
// Parameters:
//   - ctx: The context for the database operation.
//   - taskID: The ID of the task to retrieve.
//
// Returns:
//   - A pointer to models.TaskDetails containing the task information, or nil if not found.
//   - An error if the query fails or the task does not exist.
func (r *Repository) GetTaskDetailsByID(ctx context.Context, taskID int) (*models.TaskDetails, error) {
	query := `
		SELECT
			t.task_id,
			tt.type_name,
			t.creation_date,
			t.description,
			t.address,
			ARRAY_AGG(DISTINCT c.name) FILTER (WHERE c.name IS NOT NULL) AS customer_names,
			t.comments,
			t.latitude,
			t.longitude,
			COALESCE(ARRAY_AGG(e.shortname) FILTER (WHERE e.shortname IS NOT NULL), '{}') as executors
		FROM tasks t
		JOIN task_types tt ON t.task_type_id = tt.type_id
		LEFT JOIN task_executors te ON t.task_id = te.task_id
		LEFT JOIN employees e ON te.executor_id = e.id
		LEFT JOIN task_customers tc ON t.task_id = tc.task_id
		LEFT JOIN customers c ON tc.customer_id = c.id
		WHERE t.task_id = $1
		GROUP BY t.task_id, tt.type_name;
	`
	var details models.TaskDetails
	err := r.db.QueryRow(ctx, query, taskID).Scan(
		&details.ID,
		&details.Type,
		&details.CreationDate,
		&details.Description,
		&details.Address,
		&details.CustomerNames,
		&details.Comments,
		&details.Latitude,
		&details.Longitude,
		&details.Executors,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("task with id %d not found", taskID)
		}
		return nil, fmt.Errorf("failed to query task details: %w", err)
	}
	return &details, nil
}

// GetTasksInRadius retrieves a list of active tasks within a specified radius from a given latitude and longitude.
// It executes a SQL query to find tasks that are not closed and fall within the specified distance.
//
// Parameters:
// - ctx: The context for the request, allowing for cancellation and timeout.
// - lat: The latitude of the center point to search from.
// - lng: The longitude of the center point to search from.
// - radius: The radius in kilometers within which to search for tasks.
//
// Returns:
// - A slice of ActiveTask models representing the tasks found within the radius.
// - An error if the query fails or if there is an issue scanning the results.
func (r *Repository) GetTasksInRadius(ctx context.Context, lat, lng float32, radius int) ([]models.ActiveTask, error) {
	query := `
		SELECT
			task_id,
			description
		FROM (
			SELECT
				*,
				(
					6371 * acos(
						cos(radians($1)) * cos(radians(latitude)) *
						cos(radians(longitude) - radians($2)) +
						sin(radians($1)) * sin(radians(latitude))
					)
				) AS distance_km
			FROM tasks
			WHERE
				latitude BETWEEN ($1 - ($3 / 111.0)) AND ($1 + ($3 / 111.0))
				AND longitude BETWEEN ($2 - ($3 / (111.0 * cos(radians($1))))) AND ($2 + ($3 / (111.0 * cos(radians($1)))))
				AND is_closed = false
		) AS subquery
		WHERE distance_km <= $3
		ORDER BY distance_km;
	`
	rows, err := r.db.Query(ctx, query, lat, lng, radius)
	if err != nil {
		return nil, fmt.Errorf("failed to query near tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.ActiveTask
	for rows.Next() {
		var task models.ActiveTask
		if errScan := rows.Scan(&task.ID, &task.Description); errScan != nil {
			return nil, fmt.Errorf("failed to scan near task row: %w", errScan)
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}

	return tasks, nil
}

// GetCustomersByTaskID retrieves a list of customers associated with a specific task ID.
// It executes a SQL query to select customer details from the database, including
// external ID, name, and login. If the task ID is valid, it returns a slice of
// models.Customer and an error if any occurs during the process.
//
// Parameters:
//   - ctx: The context for the database operation.
//   - taskID: The ID of the task for which to retrieve associated customers.
//
// Returns:
//   - A slice of models.Customer containing the customer details.
//   - An error if the operation fails.
func (r *Repository) GetCustomersByTaskID(ctx context.Context, taskID int64) ([]models.Customer, error) {
	query := `
		SELECT external_id, name, login
		FROM customers c
		LEFT JOIN task_customers tc ON tc.customer_id = c.id
		WHERE tc.task_id = $1;
	`
	rows, err := r.db.Query(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to select customers to assigned task %d: %w", taskID, err)
	}
	defer rows.Close()

	var customers []models.Customer
	for rows.Next() {
		var customer models.Customer
		var customerID pgtype.Int8
		if err = rows.Scan(&customerID, &customer.Fullname, &customer.Login); err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}
		if customerID.Valid {
			customer.ID = customerID.Int64
		}
		customers = append(customers, customer)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}

	return customers, nil
}
