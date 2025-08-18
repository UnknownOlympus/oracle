package repository_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/UnknownOlympus/oracle/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTaskSummary(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	telegramID := int64(123456)
	to := time.Now()
	from := to.AddDate(0, -1, 0)

	t.Run("error - query task summaries", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(repository.GetTaskSummarySQL)).
			WithArgs(telegramID, from, to).
			WillReturnError(assert.AnError)

		_, err = repo.GetTaskSummary(ctx, telegramID, from, to)

		require.Error(t, err)
		require.ErrorContains(t, err, "error querying task")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - scan summaries", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(repository.GetTaskSummarySQL)).
			WithArgs(telegramID, from, to).
			WillReturnRows(
				pgxmock.NewRows([]string{"task_type", "count"}).AddRow("Task Type", "invalid_count"),
			)

		_, err = repo.GetTaskSummary(ctx, telegramID, from, to)

		require.Error(t, err)
		require.ErrorContains(t, err, "error scanning summaries")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - rows error", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(repository.GetTaskSummarySQL)).
			WithArgs(telegramID, from, to).
			WillReturnRows(
				pgxmock.NewRows([]string{"task_type", "count"}).AddRow("Task Type", 1).
					RowError(1, assert.AnError),
			)

		_, err = repo.GetTaskSummary(ctx, telegramID, from, to)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to iterating")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - get task summary", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(repository.GetTaskSummarySQL)).
			WithArgs(telegramID, from, to).
			WillReturnRows(
				pgxmock.NewRows([]string{"task_type", "count"}).AddRow("Task Type", 1).AddRow("Test", 2),
			)

		summ, err := repo.GetTaskSummary(ctx, telegramID, from, to)

		require.NoError(t, err)
		summ1 := summ[0]
		assert.Equal(t, "Task Type", summ1.Type)
		assert.Equal(t, 1, summ1.Count)
		summ2 := summ[1]
		assert.Equal(t, "Test", summ2.Type)
		assert.Equal(t, 2, summ2.Count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetActiveTasksByExecutor(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	telegramID := int64(123456)
	query := `
		SELECT t.task_id, t.description
		FROM tasks t
		JOIN task_executors te ON t.task_id = te.task_id
		JOIN bot_users bu ON te.executor_id = bu.employee_id
		WHERE bu.telegram_id = $1 AND t.is_closed = FALSE
		ORDER BY t.creation_date DESC;
	`

	t.Run("error - query error", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(telegramID).
			WillReturnError(assert.AnError)

		_, err = repo.GetActiveTasksByExecutor(ctx, telegramID)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to query")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - scan active tasks", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(telegramID).
			WillReturnRows(
				pgxmock.NewRows([]string{"task_id", "description"}).AddRow("invalid_id", "some descr"),
			)

		_, err = repo.GetActiveTasksByExecutor(ctx, telegramID)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to scan")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - rows error", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(telegramID).
			WillReturnRows(
				pgxmock.NewRows([]string{"task_id", "description"}).AddRow(123, "descr").
					RowError(1, assert.AnError),
			)

		_, err = repo.GetActiveTasksByExecutor(ctx, telegramID)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to read rows")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - get active tasks", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(telegramID).
			WillReturnRows(
				pgxmock.NewRows([]string{"task_id", "description"}).AddRow(12345, "12345").AddRow(12346, "12346"),
			)

		tasks, err := repo.GetActiveTasksByExecutor(ctx, telegramID)

		require.NoError(t, err)
		task1 := tasks[0]
		assert.Equal(t, 12345, task1.ID)
		assert.Equal(t, "12345", task1.Description)
		task2 := tasks[1]
		assert.Equal(t, 12346, task2.ID)
		assert.Equal(t, "12346", task2.Description)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetCompletedTasksByExecutor(t *testing.T) {
	ctx := t.Context()
	telegramID := int64(123456)
	to := time.Now()
	from := to.AddDate(0, -1, 0)
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

	t.Run("error - query error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(telegramID, from, to).
			WillReturnError(assert.AnError)

		_, err = repo.GetCompletedTasksByExecutor(ctx, telegramID, from, to)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to query")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - scan completed tasks", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(telegramID, from, to).
			WillReturnRows(
				pgxmock.NewRows([]string{
					"task_id", "type_name", "creation_date", "closing_date", "description",
					"address", "customer_names", "comments",
				}).
					AddRow("invalid_id", "repair", time.Now(), time.Now(), "descr",
						"test addr", []string{"test user"}, []string{"1 comm", "2 comm"}),
			)

		_, err = repo.GetCompletedTasksByExecutor(ctx, telegramID, from, to)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to scan")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - rows error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(telegramID, from, to).
			WillReturnRows(
				pgxmock.NewRows([]string{
					"task_id", "type_name", "creation_date", "closing_date", "description",
					"address", "customer_names", "comments",
				}).
					AddRow(12345, "repair", time.Now(), time.Now(), "descr",
						"test addr", []string{"test user"}, []string{"1 comm", "2 comm"}).
					RowError(1, assert.AnError),
			)

		_, err = repo.GetCompletedTasksByExecutor(ctx, telegramID, from, to)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to read rows")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - get active tasks", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		now := time.Now()

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(telegramID, from, to).
			WillReturnRows(
				pgxmock.NewRows([]string{
					"task_id", "type_name", "creation_date", "closing_date", "description",
					"address", "customer_names", "comments",
				}).
					AddRow(12345, "repair", now, now, "descr",
						"test addr", []string{"test user"}, []string{"1 comm", "2 comm"}),
			)

		tasks, err := repo.GetCompletedTasksByExecutor(ctx, telegramID, from, to)

		require.NoError(t, err)
		task := tasks[0]
		assert.Equal(t, 12345, task.ID)
		assert.Equal(t, "repair", task.Type)
		assert.Equal(t, now, task.CreationDate)
		assert.Equal(t, now, task.ClosingDate)
		assert.Equal(t, "descr", task.Description)
		assert.Equal(t, "test addr", task.Address)
		assert.Equal(t, []string{"test user"}, task.CustomerNames)
		assert.Equal(t, []string{"1 comm", "2 comm"}, task.Comments)
	})
}

func TestGetTaskDetailsByID(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	taskID := 12345
	now := time.Now()
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

	t.Run("error - query task details", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(taskID).
			WillReturnError(assert.AnError)

		_, err = repo.GetTaskDetailsByID(ctx, taskID)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to query task details")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - task not found", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(taskID).
			WillReturnError(pgx.ErrNoRows)

		_, err = repo.GetTaskDetailsByID(ctx, taskID)

		require.Error(t, err)
		require.ErrorContains(t, err, "not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - get task details", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(taskID).
			WillReturnRows(mock.NewRows([]string{
				"task_id", "type_name", "creation_date", "description",
				"address", "customer_names", "comments", "latitude", "longitude", "executors",
			}).
				AddRow(123, "type", now, "descr", "addr", []string{"test user"}, []string{"1", "2"}, 12.345, 23.456, []string{"test", "executor 1"}),
			)

		task, err := repo.GetTaskDetailsByID(ctx, taskID)

		require.NoError(t, err)
		assert.Equal(t, 123, task.ID)
		assert.Equal(t, "type", task.Type)
		assert.Equal(t, now, task.CreationDate)
		assert.Equal(t, "descr", task.Description)
		assert.Equal(t, "addr", task.Address)
		assert.Equal(t, []string{"test user"}, task.CustomerNames)
		assert.Equal(t, []string{"1", "2"}, task.Comments)
		assert.InEpsilon(t, 12.345, task.Latitude.Float64, 0.001)
		assert.InEpsilon(t, 23.456, task.Longitude.Float64, 0.001)
		assert.Equal(t, []string{"test", "executor 1"}, task.Executors)
	})
}

func TestGetTasksInRadius(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	lat := float32(12.345)
	lng := float32(23.456)
	radius := 10
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

	t.Run("error - query error", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(lat, lng, radius).
			WillReturnError(assert.AnError)

		_, err = repo.GetTasksInRadius(ctx, lat, lng, radius)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to query near tasks")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - scan active tasks", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(lat, lng, radius).
			WillReturnRows(
				pgxmock.NewRows([]string{"task_id", "description"}).AddRow("invalid_id", "some descr"),
			)

		_, err = repo.GetTasksInRadius(ctx, lat, lng, radius)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to scan near task row")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - rows error", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(lat, lng, radius).
			WillReturnRows(
				pgxmock.NewRows([]string{"task_id", "description"}).AddRow(123, "descr").
					RowError(1, assert.AnError),
			)

		_, err = repo.GetTasksInRadius(ctx, lat, lng, radius)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to read rows")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - get active tasks", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(lat, lng, radius).
			WillReturnRows(
				pgxmock.NewRows([]string{"task_id", "description"}).AddRow(12345, "12345").AddRow(12346, "12346"),
			)

		tasks, err := repo.GetTasksInRadius(ctx, lat, lng, radius)

		require.NoError(t, err)
		task1 := tasks[0]
		assert.Equal(t, 12345, task1.ID)
		assert.Equal(t, "12345", task1.Description)
		task2 := tasks[1]
		assert.Equal(t, 12346, task2.ID)
		assert.Equal(t, "12346", task2.Description)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
