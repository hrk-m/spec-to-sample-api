// Package mysql provides MySQL implementations of repository interfaces.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// GroupRepository is a MySQL implementation of group.GroupRepository.
type GroupRepository struct {
	db *sql.DB
}

// NewGroupRepository returns a new GroupRepository.
func NewGroupRepository(db *sql.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

// ListGroups returns a filtered list of groups with filtered total count.
func (r *GroupRepository) ListGroups(ctx context.Context, q string, limit, offset int) ([]domain.Group, int, error) {
	total, err := r.countFilteredGroups(ctx, q)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []domain.Group{}, 0, nil
	}

	groups, err := r.selectGroups(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return groups, total, nil
}

// countFilteredGroups returns the number of non-deleted groups matching the optional search filter.
func (r *GroupRepository) countFilteredGroups(ctx context.Context, q string) (int, error) {
	query := "SELECT COUNT(*) FROM `groups` g WHERE g.deleted_at IS NULL"

	searchCondition, args := buildSearchCondition(q)
	query += searchCondition

	var total int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, domain.ErrInternalServerError
	}

	return total, nil
}

// selectGroups returns non-deleted groups with member counts, optionally filtered by q.
func (r *GroupRepository) selectGroups(ctx context.Context, q string, limit, offset int) ([]domain.Group, error) {
	query := "SELECT g.id, g.name, g.description, COUNT(gm.id) AS member_count" +
		" FROM `groups` g LEFT JOIN group_members gm ON g.id = gm.group_id" +
		" WHERE g.deleted_at IS NULL"

	searchCondition, args := buildSearchCondition(q)
	query += searchCondition //nolint:gosec // search condition uses parameterized placeholders

	query += " GROUP BY g.id, g.name, g.description"
	query += " ORDER BY g.id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, domain.ErrInternalServerError
	}
	defer func() { _ = rows.Close() }()

	var groups []domain.Group

	for rows.Next() {
		var g domain.Group
		if scanErr := rows.Scan(&g.ID, &g.Name, &g.Description, &g.MemberCount); scanErr != nil {
			return nil, domain.ErrInternalServerError
		}

		groups = append(groups, g)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, domain.ErrInternalServerError
	}

	if groups == nil {
		groups = []domain.Group{}
	}

	return groups, nil
}

// Store inserts a new group and its creator as the first member within a transaction.
func (r *GroupRepository) Store(ctx context.Context, name, description string, userID uint64) (domain.Group, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Group{}, domain.ErrInternalServerError
	}

	result, err := tx.ExecContext(ctx, "INSERT INTO `groups` (name, description, updated_by) VALUES (?, ?, ?)", name, description, userID)
	if err != nil {
		_ = tx.Rollback()

		return domain.Group{}, domain.ErrInternalServerError
	}

	id, err := result.LastInsertId()
	if err != nil || id < 0 {
		_ = tx.Rollback()

		return domain.Group{}, domain.ErrInternalServerError
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO group_members (group_id, user_id) VALUES (?, ?)", id, userID)
	if err != nil {
		_ = tx.Rollback()

		return domain.Group{}, domain.ErrInternalServerError
	}

	if commitErr := tx.Commit(); commitErr != nil {
		_ = tx.Rollback()

		return domain.Group{}, domain.ErrInternalServerError
	}

	return domain.Group{
		ID:          uint64(id), //nolint:gosec // id is validated non-negative above
		Name:        name,
		Description: description,
		MemberCount: 1,
	}, nil
}

// Update modifies a group's name, description, and updated_by, then returns the updated entity.
func (r *GroupRepository) Update(ctx context.Context, id uint64, name, description string, userID uint64) (*domain.Group, error) {
	query := "UPDATE `groups` SET name = ?, description = ?, updated_by = ? WHERE id = ? AND deleted_at IS NULL"

	result, err := r.db.ExecContext(ctx, query, name, description, userID, id)
	if err != nil {
		return nil, domain.ErrInternalServerError
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, domain.ErrInternalServerError
	}

	if rows == 0 {
		return nil, domain.ErrNotFound
	}

	g, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &g, nil
}

// Delete soft-deletes a group by setting deleted_at and updated_by.
func (r *GroupRepository) Delete(ctx context.Context, id uint64, userID uint64) error {
	query := "UPDATE `groups` SET deleted_at = NOW(), updated_by = ? WHERE id = ? AND deleted_at IS NULL"

	result, err := r.db.ExecContext(ctx, query, userID, id)
	if err != nil {
		return domain.ErrInternalServerError
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.ErrInternalServerError
	}

	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// GetByID returns a single group by ID with its member count.
func (r *GroupRepository) GetByID(ctx context.Context, id uint64) (domain.Group, error) {
	query := "SELECT g.id, g.name, g.description, COUNT(gm.id) AS member_count" +
		" FROM `groups` g LEFT JOIN group_members gm ON g.id = gm.group_id" +
		" WHERE g.id = ? AND g.deleted_at IS NULL" +
		" GROUP BY g.id, g.name, g.description"

	var g domain.Group
	err := r.db.QueryRowContext(ctx, query, id).Scan(&g.ID, &g.Name, &g.Description, &g.MemberCount)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Group{}, domain.ErrNotFound
		}

		return domain.Group{}, domain.ErrInternalServerError
	}

	return g, nil
}

// ListGroupMembers returns paginated members for a group with optional name search.
func (r *GroupRepository) ListGroupMembers(ctx context.Context, id uint64, limit, offset int, q string) ([]domain.User, int, error) {
	// Count members for the group with optional q filter.
	countQuery := "SELECT COUNT(*) FROM group_members gm JOIN users u ON gm.user_id = u.id WHERE gm.group_id = ?"
	countArgs := []interface{}{id}

	if q != "" {
		countQuery += " AND u.search_key LIKE ?" //nolint:goconst
		countArgs = append(countArgs, "%"+q+"%")
	}

	var total int

	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, domain.ErrInternalServerError
	}

	if total == 0 {
		return []domain.User{}, 0, nil
	}

	// Fetch paginated members with optional q filter.
	query := "SELECT u.id, u.uuid, u.first_name, u.last_name" +
		" FROM group_members gm JOIN users u ON gm.user_id = u.id" +
		" WHERE gm.group_id = ?"
	args := []interface{}{id}

	if q != "" {
		query += " AND u.search_key LIKE ?" //nolint:goconst
		args = append(args, "%"+q+"%")
	}

	query += " ORDER BY u.id LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, domain.ErrInternalServerError
	}
	defer func() { _ = rows.Close() }()

	var members []domain.User

	for rows.Next() {
		var m domain.User
		if scanErr := rows.Scan(&m.ID, &m.UUID, &m.FirstName, &m.LastName); scanErr != nil {
			return nil, 0, domain.ErrInternalServerError
		}

		members = append(members, m)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, domain.ErrInternalServerError
	}

	if members == nil {
		members = []domain.User{}
	}

	return members, total, nil
}

// ListNonGroupMembers returns paginated users not in the given group.
func (r *GroupRepository) ListNonGroupMembers(ctx context.Context, groupID uint64, limit, offset int, q string) ([]domain.User, int, error) {
	// Count non-members with optional q filter.
	countQuery := "SELECT COUNT(*) FROM users WHERE id NOT IN" +
		" (SELECT user_id FROM group_members WHERE group_id = ?)" +
		" AND deleted_at IS NULL"
	countArgs := []interface{}{groupID}

	if q != "" {
		countQuery += " AND search_key LIKE ?" //nolint:goconst
		countArgs = append(countArgs, "%"+q+"%")
	}

	var total int

	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, domain.ErrInternalServerError
	}

	if total == 0 {
		return []domain.User{}, 0, nil
	}

	// Fetch paginated non-members with optional q filter.
	query := "SELECT id, uuid, first_name, last_name FROM users" +
		" WHERE id NOT IN (SELECT user_id FROM group_members WHERE group_id = ?)" +
		" AND deleted_at IS NULL"
	args := []interface{}{groupID}

	if q != "" {
		query += " AND search_key LIKE ?" //nolint:goconst
		args = append(args, "%"+q+"%")
	}

	query += " ORDER BY id ASC LIMIT ? OFFSET ?" //nolint:goconst
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, domain.ErrInternalServerError
	}
	defer func() { _ = rows.Close() }()

	var users []domain.User

	for rows.Next() {
		var u domain.User
		if scanErr := rows.Scan(&u.ID, &u.UUID, &u.FirstName, &u.LastName); scanErr != nil {
			return nil, 0, domain.ErrInternalServerError
		}

		users = append(users, u)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, domain.ErrInternalServerError
	}

	if users == nil {
		users = []domain.User{}
	}

	return users, total, nil
}

// AddGroupMembers inserts all userIDs into group_members within a transaction and returns added users.
func (r *GroupRepository) AddGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) ([]domain.User, error) {
	// Check for existing members in one query before starting the transaction.
	placeholders := make([]string, len(userIDs))
	checkArgs := make([]interface{}, 0, len(userIDs)+1)
	checkArgs = append(checkArgs, groupID)

	for i, uid := range userIDs {
		placeholders[i] = "?"
		checkArgs = append(checkArgs, uid)
	}

	checkQuery := fmt.Sprintf( //nolint:gosec
		"SELECT COUNT(*) FROM group_members WHERE group_id = ? AND user_id IN (%s)",
		strings.Join(placeholders, ","),
	)

	var existingCount int
	if err := r.db.QueryRowContext(ctx, checkQuery, checkArgs...).Scan(&existingCount); err != nil {
		return nil, domain.ErrInternalServerError
	}

	if existingCount > 0 {
		return nil, domain.ErrConflict
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, domain.ErrInternalServerError
	}

	for _, userID := range userIDs {
		_, execErr := tx.ExecContext(ctx, "INSERT INTO group_members (group_id, user_id) VALUES (?, ?)", groupID, userID)
		if execErr != nil {
			_ = tx.Rollback()

			if isUniqueConstraintError(execErr) {
				return nil, domain.ErrConflict
			}

			return nil, domain.ErrInternalServerError
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		_ = tx.Rollback()

		return nil, domain.ErrInternalServerError
	}

	// Fetch the added users. Reuse the same placeholders slice built above.
	selectArgs := make([]interface{}, len(userIDs))

	for i, id := range userIDs {
		selectArgs[i] = id
	}

	selectQuery := fmt.Sprintf("SELECT id, uuid, first_name, last_name FROM users WHERE id IN (%s) ORDER BY id ASC", //nolint:gosec
		strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, selectQuery, selectArgs...)
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

// RemoveGroupMembers removes the given users from a group within a transaction.
// Returns ErrNotFound if any userID is not currently a member of the group.
func (r *GroupRepository) RemoveGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) error {
	placeholders := make([]string, len(userIDs))
	deleteArgs := make([]interface{}, 0, len(userIDs)+1)
	deleteArgs = append(deleteArgs, groupID)

	for i, uid := range userIDs {
		placeholders[i] = "?"
		deleteArgs = append(deleteArgs, uid)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ErrInternalServerError
	}

	deleteQuery := fmt.Sprintf( //nolint:gosec
		"DELETE FROM group_members WHERE group_id = ? AND user_id IN (%s)",
		strings.Join(placeholders, ","),
	)

	result, execErr := tx.ExecContext(ctx, deleteQuery, deleteArgs...)
	if execErr != nil {
		_ = tx.Rollback()

		return domain.ErrInternalServerError
	}

	affected, raErr := result.RowsAffected()
	if raErr != nil {
		_ = tx.Rollback()

		return domain.ErrInternalServerError
	}

	if int(affected) != len(userIDs) {
		_ = tx.Rollback()

		return domain.ErrNotFound
	}

	if commitErr := tx.Commit(); commitErr != nil {
		_ = tx.Rollback()

		return domain.ErrInternalServerError
	}

	return nil
}

// isUniqueConstraintError reports whether err is a MySQL duplicate entry error (error 1062).
func isUniqueConstraintError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// buildSearchCondition returns an AND search condition for each whitespace-delimited token.
func buildSearchCondition(search string) (string, []interface{}) {
	tokens := strings.Fields(search)
	if len(tokens) == 0 {
		return "", nil
	}

	conditions := make([]string, 0, len(tokens))
	args := make([]interface{}, 0, len(tokens)*2)

	for _, token := range tokens {
		conditions = append(conditions, "(g.name LIKE ? OR g.description LIKE ?)")

		like := "%" + token + "%"
		args = append(args, like, like)
	}

	return " AND " + strings.Join(conditions, " AND "), args
}
