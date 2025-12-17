package repository_test

import (
	"regexp"
	"testing"

	"github.com/UnknownOlympus/oracle/internal/models"
	"github.com/UnknownOlympus/oracle/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const selectEmployee = "SELECT id FROM employees WHERE email = \\$1"

const selectExistsEmployee = "SELECT EXISTS \\(SELECT 1 FROM bot_users WHERE telegram_id = \\$1\\)"

const deleteUser = "DELETE FROM bot_users WHERE telegram_id = \\$1"

const insertIntoBotUsers = `
	INSERT INTO bot_users (telegram_id, employee_id)
	VALUES ($1, $2) ON CONFLICT (employee_id) DO NOTHING
`

const selectGetEmployee = `
	SELECT id, fullname, shortname, position, email, phone, is_admin FROM employees
	WHERE id = (SELECT employee_id FROM bot_users WHERE telegram_id = $1);		
`

func TestLinkTelegramIDByEmail(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	telegramID := int64(12345)
	employeeID := 101
	email := "test@test.com"

	t.Run("error - failed to begin transaction", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectBegin().WillReturnError(assert.AnError)

		err = repo.LinkTelegramIDByEmail(ctx, telegramID, email)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to begin transaction")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - user not found", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectBegin()
		mock.ExpectQuery(selectEmployee).WithArgs(email).WillReturnError(pgx.ErrNoRows)

		err = repo.LinkTelegramIDByEmail(ctx, telegramID, email)

		require.Error(t, err)
		require.ErrorIs(t, err, repository.ErrUserNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - failed to find employee", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectBegin()
		mock.ExpectQuery(selectEmployee).WithArgs(email).WillReturnError(assert.AnError)

		err = repo.LinkTelegramIDByEmail(ctx, telegramID, email)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		require.ErrorContains(t, err, "failed to find employee")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - failed to get user by telegram ID", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectBegin()
		mock.ExpectQuery(selectEmployee).
			WithArgs(email).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(employeeID))
		mock.ExpectQuery(selectExistsEmployee).WithArgs(telegramID).WillReturnError(assert.AnError)

		err = repo.LinkTelegramIDByEmail(ctx, telegramID, email)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		require.ErrorContains(t, err, "failed to get user by telegram")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - ID is exists", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectBegin()
		mock.ExpectQuery(selectEmployee).
			WithArgs(email).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(employeeID))
		mock.ExpectQuery(selectExistsEmployee).
			WithArgs(telegramID).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

		err = repo.LinkTelegramIDByEmail(ctx, telegramID, email)

		require.Error(t, err)
		require.ErrorIs(t, err, repository.ErrIDExists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - user already linked", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectBegin()
		mock.ExpectQuery(selectEmployee).
			WithArgs(email).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(employeeID))
		mock.ExpectQuery(selectExistsEmployee).
			WithArgs(telegramID).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
		mock.ExpectExec(regexp.QuoteMeta(insertIntoBotUsers)).
			WithArgs(telegramID, employeeID).
			WillReturnError(pgx.ErrNoRows)

		err = repo.LinkTelegramIDByEmail(ctx, telegramID, email)

		require.Error(t, err)
		require.ErrorIs(t, err, repository.ErrUserAlreadyLinked)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - failed to insert into", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectBegin()
		mock.ExpectQuery(selectEmployee).
			WithArgs(email).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(employeeID))
		mock.ExpectQuery(selectExistsEmployee).
			WithArgs(telegramID).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
		mock.ExpectExec(regexp.QuoteMeta(insertIntoBotUsers)).
			WithArgs(telegramID, employeeID).
			WillReturnError(assert.AnError)

		err = repo.LinkTelegramIDByEmail(ctx, telegramID, email)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		require.ErrorContains(t, err, "failed to insert into")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - no rows affected", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		cmdTag := pgconn.NewCommandTag("CREATE TABLE")

		mock.ExpectBegin()
		mock.ExpectQuery(selectEmployee).
			WithArgs(email).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(employeeID))
		mock.ExpectQuery(selectExistsEmployee).
			WithArgs(telegramID).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
		mock.ExpectExec(regexp.QuoteMeta(insertIntoBotUsers)).
			WithArgs(telegramID, employeeID).
			WillReturnResult(cmdTag)

		err = repo.LinkTelegramIDByEmail(ctx, telegramID, email)

		require.Error(t, err)
		require.ErrorIs(t, err, repository.ErrUserAlreadyLinked)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - link telegram id", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		cmdTag := pgconn.NewCommandTag("1")

		mock.ExpectBegin()
		mock.ExpectQuery(selectEmployee).
			WithArgs(email).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(employeeID))
		mock.ExpectQuery(selectExistsEmployee).
			WithArgs(telegramID).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
		mock.ExpectExec(regexp.QuoteMeta(insertIntoBotUsers)).
			WithArgs(telegramID, employeeID).
			WillReturnResult(cmdTag)
		mock.ExpectCommit()

		err = repo.LinkTelegramIDByEmail(ctx, telegramID, email)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIsUserAuthenticated(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	telegramID := int64(12345)

	t.Run("error - failed to check user", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(selectExistsEmployee).WithArgs(telegramID).WillReturnError(assert.AnError)

		_, err = repo.IsUserAuthenticated(ctx, telegramID)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		require.ErrorContains(t, err, "failed to check user")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - authenticate user", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(selectExistsEmployee).
			WithArgs(telegramID).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

		exists, err := repo.IsUserAuthenticated(ctx, telegramID)

		assert.True(t, exists)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDeleteUserByID(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	telegramID := int64(12345)

	t.Run("error - failed to delete user", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectExec(deleteUser).WithArgs(telegramID).WillReturnError(assert.AnError)

		err = repo.DeleteUserByID(ctx, telegramID)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		require.ErrorContains(t, err, "failed to delete user")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - delete user", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectExec(deleteUser).WithArgs(telegramID).WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err = repo.DeleteUserByID(ctx, telegramID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetEmployee(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	telegramID := int64(12345)

	t.Run("error - failed to get employee", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(selectGetEmployee)).WithArgs(telegramID).WillReturnError(assert.AnError)

		_, err = repo.GetEmployee(ctx, telegramID)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		require.ErrorContains(t, err, "failed to get employee data")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - get employee", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(selectGetEmployee)).
			WithArgs(telegramID).
			WillReturnRows(
				pgxmock.NewRows([]string{"id", "fullname", "shortname", "position", "email", "phone", "is_admin"}).
					AddRow(123, "testFull", "testShort", "testPos", "testEmail", "testPhone", true),
			)

		employee, err := repo.GetEmployee(ctx, telegramID)

		require.NoError(t, err)
		assert.Equal(t, "testFull", employee.FullName)
		assert.Equal(t, "testShort", employee.ShortName)
		assert.Equal(t, "testPos", employee.Position)
		assert.Equal(t, "testEmail", employee.Email)
		assert.Equal(t, "testPhone", employee.Phone)
		assert.True(t, employee.IsAdmin)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetAllTgUserIDs(t *testing.T) {
	ctx := t.Context()
	id := int64(12345678)
	query := "SELECT telegram_id from bot_users"

	t.Run("error - query error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WillReturnError(assert.AnError)

		_, err = repo.GetAllTgUserIDs(ctx)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to get all telegram user IDs")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - scan telegram_id", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WillReturnRows(
				pgxmock.NewRows([]string{"telegram_id"}).
					AddRow("invalid_id"))

		_, err = repo.GetAllTgUserIDs(ctx)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to scan telegram_id row")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - rows error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WillReturnRows(
				pgxmock.NewRows([]string{"telegram_id"}).
					AddRow(id).
					CloseError(assert.AnError),
			)

		_, err = repo.GetAllTgUserIDs(ctx)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to read rows")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - get all telegram_id", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WillReturnRows(
				pgxmock.NewRows([]string{"telegram_id"}).
					AddRow(id),
			)

		actIDs, err := repo.GetAllTgUserIDs(ctx)

		require.NoError(t, err)
		assert.Equal(t, id, actIDs[0])
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetAdmins(t *testing.T) {
	ctx := t.Context()
	query := `
		SELECT telegram_id, employee_id 
		FROM bot_users bu 
		LEFT JOIN employees e ON e.id = bu.employee_id
		WHERE e.is_admin = TRUE
	`
	botUser := models.BotUser{TelegramID: int64(123456), EmployeeID: 9999}

	t.Run("error - query error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WillReturnError(assert.AnError)

		_, err = repo.GetAdmins(ctx)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to get all bot users with admin privileges")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - scan integer type", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WillReturnRows(
				pgxmock.NewRows([]string{"telegram_id", "employee_id"}).
					AddRow("invalid_id", "invalid_id"))

		_, err = repo.GetAdmins(ctx)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to scan bot user row")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - rows error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WillReturnRows(
				pgxmock.NewRows([]string{"telegram_id", "employee_id"}).
					AddRow(botUser.TelegramID, botUser.EmployeeID).
					CloseError(assert.AnError),
			)

		_, err = repo.GetAdmins(ctx)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to read rows")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - get all telegram_id", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WillReturnRows(
				pgxmock.NewRows([]string{"telegram_id", "employee_id"}).
					AddRow(botUser.TelegramID, botUser.EmployeeID),
			)

		users, err := repo.GetAdmins(ctx)

		require.NoError(t, err)
		actUser := users[0]
		assert.Equal(t, botUser.EmployeeID, actUser.EmployeeID)
		assert.Equal(t, botUser.TelegramID, actUser.TelegramID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIsAdmin(t *testing.T) {
	ctx := t.Context()
	telegramID := int64(12345)
	query := `
		SELECT is_admin FROM employees
		WHERE id = (SELECT employee_id FROM bot_users WHERE telegram_id = $1);
	`

	t.Run("error - query error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(telegramID).
			WillReturnError(assert.AnError)

		isAdmin, err := repo.IsAdmin(ctx, telegramID)

		require.Error(t, err)
		assert.False(t, isAdmin)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - query error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(telegramID).
			WillReturnRows(pgxmock.NewRows([]string{"is_admin"}).AddRow(true))

		isAdmin, err := repo.IsAdmin(ctx, telegramID)

		require.NoError(t, err)
		assert.True(t, isAdmin)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
