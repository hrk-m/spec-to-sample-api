//go:build integration

package mysql_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
	mysqlRepo "github.com/hrk-m/spec-to-dev-workflow/sample-api/internal/repository/mysql"
)

func userTestDB(t *testing.T) *sql.DB {
	t.Helper()

	host := getEnv("MYSQL_HOST", "localhost")
	port := getEnv("MYSQL_PORT", "3306")
	user := getEnv("MYSQL_USER", "root")
	pass := getEnv("MYSQL_PASSWORD", "password")
	dbname := getEnv("MYSQL_DATABASE", "sample")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, dbname)

	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())

	return db
}

func activeUsersCount(t *testing.T, db *sql.DB) int {
	t.Helper()

	var total int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&total))

	return total
}

func TestListUsers_DefaultPagination(t *testing.T) {
	db := userTestDB(t)
	defer db.Close()

	repo := mysqlRepo.NewUserRepository(db)
	users, total, err := repo.ListUsers(context.Background(), "", 10, 0)

	assert.NoError(t, err)
	assert.Equal(t, activeUsersCount(t, db), total)
	assert.Len(t, users, 10)
}

func TestListUsers_Search(t *testing.T) {
	db := userTestDB(t)
	defer db.Close()

	repo := mysqlRepo.NewUserRepository(db)
	users, total, err := repo.ListUsers(context.Background(), "Suzuki", 10, 0)

	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, users, 1)
	assert.Equal(t, "Hanako", users[0].FirstName)
	assert.Equal(t, "Suzuki", users[0].LastName)
}

func TestListUsers_SearchKeyLike(t *testing.T) {
	db := userTestDB(t)
	defer db.Close()

	repo := mysqlRepo.NewUserRepository(db)
	users, total, err := repo.ListUsers(context.Background(), "HanakoSuz", 10, 0)

	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, users, 1)
	assert.Equal(t, "Hanako", users[0].FirstName)
	assert.Equal(t, "Suzuki", users[0].LastName)
}

func TestListUsers_EmptyResultWhenPagingPastEnd(t *testing.T) {
	db := userTestDB(t)
	defer db.Close()

	repo := mysqlRepo.NewUserRepository(db)
	offset := activeUsersCount(t, db) + 100
	users, total, err := repo.ListUsers(context.Background(), "", 10, offset)

	assert.NoError(t, err)
	assert.Equal(t, activeUsersCount(t, db), total)
	assert.Empty(t, users)
	assert.NotNil(t, users)
}

func TestListUsers_ExcludesDeleted(t *testing.T) {
	db := userTestDB(t)
	defer db.Close()

	result, err := db.Exec("INSERT INTO users (first_name, last_name, deleted_at) VALUES ('Deleted', 'User', NOW())")
	require.NoError(t, err)

	deletedID, err := result.LastInsertId()
	require.NoError(t, err)

	defer db.Exec("DELETE FROM users WHERE id = ?", deletedID) //nolint:errcheck

	repo := mysqlRepo.NewUserRepository(db)
	users, total, err := repo.ListUsers(context.Background(), "", 10, 0)

	assert.NoError(t, err)
	assert.Equal(t, activeUsersCount(t, db), total)
	assert.Len(t, users, 10)
}
