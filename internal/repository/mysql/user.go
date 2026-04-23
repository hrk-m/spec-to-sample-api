// Package mysql provides MySQL implementations of repository interfaces.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// UserRepository is a MySQL implementation of user.UserRepository and group.UserRepository.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository returns a new UserRepository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// ListUsers returns paginated active users with optional name search.
func (r *UserRepository) ListUsers(ctx context.Context, q string, limit, offset int) ([]domain.User, int, error) {
	total, err := r.countFilteredUsers(ctx, q)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []domain.User{}, 0, nil
	}

	users, err := r.selectUsers(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// CountByIDs returns the count of existing active users matching the given IDs.
// COUNT(DISTINCT id) is used so that duplicate IDs in the input do not inflate the result.
func (r *UserRepository) CountByIDs(ctx context.Context, ids []uint64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))

	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf( //nolint:gosec
		"SELECT COUNT(DISTINCT id) FROM users WHERE id IN (%s) AND deleted_at IS NULL",
		strings.Join(placeholders, ","),
	)

	var count int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, domain.ErrInternalServerError
	}

	return count, nil
}

// GetByID returns a single active user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id uint64) (*domain.User, error) {
	query := "SELECT id, uuid, first_name, last_name FROM users WHERE id = ? AND deleted_at IS NULL"

	var u domain.User

	err := r.db.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.UUID, &u.FirstName, &u.LastName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}

		return nil, domain.ErrInternalServerError
	}

	return &u, nil
}

// GetByUUID returns a single active user by UUID.
func (r *UserRepository) GetByUUID(ctx context.Context, uuid string) (domain.User, error) {
	query := "SELECT id, uuid, first_name, last_name FROM users WHERE uuid = ? AND deleted_at IS NULL"

	var u domain.User

	err := r.db.QueryRowContext(ctx, query, uuid).Scan(&u.ID, &u.UUID, &u.FirstName, &u.LastName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}

		return domain.User{}, domain.ErrInternalServerError
	}

	return u, nil
}

func (r *UserRepository) countFilteredUsers(ctx context.Context, q string) (int, error) {
	query := "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL"

	var args []interface{}

	if q != "" {
		query += " AND search_key LIKE ?" //nolint:goconst
		args = append(args, "%"+q+"%")
	}

	var total int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, domain.ErrInternalServerError
	}

	return total, nil
}

func (r *UserRepository) selectUsers(ctx context.Context, q string, limit, offset int) ([]domain.User, error) {
	query := "SELECT id, uuid, first_name, last_name FROM users WHERE deleted_at IS NULL"
	args := make([]interface{}, 0, 3)

	if q != "" {
		query += " AND search_key LIKE ?" //nolint:goconst
		args = append(args, "%"+q+"%")
	}

	query += " ORDER BY id ASC LIMIT ? OFFSET ?" //nolint:goconst
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, domain.ErrInternalServerError
	}
	defer func() { _ = rows.Close() }()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if scanErr := rows.Scan(&u.ID, &u.UUID, &u.FirstName, &u.LastName); scanErr != nil {
			return nil, domain.ErrInternalServerError
		}

		users = append(users, u)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, domain.ErrInternalServerError
	}

	if users == nil {
		users = []domain.User{}
	}

	return users, nil
}
