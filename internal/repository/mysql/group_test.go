//go:build integration

package mysql_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
	mysqlRepo "github.com/hrk-m/spec-to-dev-workflow/sample-api/internal/repository/mysql"
)

func testDB(t *testing.T) *sql.DB {
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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

func countActiveGroups(t *testing.T, db *sql.DB) int {
	t.Helper()

	var total int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM `groups` WHERE deleted_at IS NULL").Scan(&total))

	return total
}

func TestListGroups_DefaultPagination(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := mysqlRepo.NewGroupRepository(db)
	groups, total, err := repo.ListGroups(context.Background(), "", 1, 10)

	assert.NoError(t, err)
	assert.Equal(t, countActiveGroups(t, db), total)
	assert.Len(t, groups, 10)
}

func TestListGroups_Search(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := mysqlRepo.NewGroupRepository(db)
	groups, total, err := repo.ListGroups(context.Background(), "001", 1, 10)

	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, groups, 1)
	assert.Equal(t, "Group 001", groups[0].Name)
}

func TestListGroups_SearchWithSpaceSeparatedTokens(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := mysqlRepo.NewGroupRepository(db)
	groups, total, err := repo.ListGroups(context.Background(), "001 Description", 1, 10)

	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, groups, 1)
	assert.Equal(t, "Group 001", groups[0].Name)
}

func TestListGroups_LastPage(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := mysqlRepo.NewGroupRepository(db)
	groups, total, err := repo.ListGroups(context.Background(), "", 3, 10)

	assert.NoError(t, err)
	assert.Equal(t, countActiveGroups(t, db), total)
	assert.Len(t, groups, 10)
}

func TestListGroups_MemberCount(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := mysqlRepo.NewGroupRepository(db)
	groups, _, err := repo.ListGroups(context.Background(), "030", 1, 10)

	assert.NoError(t, err)
	require.Len(t, groups, 1)
	// g030 is even -> 1 member
	assert.Equal(t, 1, groups[0].MemberCount)
}

func TestListGroups_ExcludesDeleted(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	// Insert a deleted group (updated_by=1 is required after migration).
	result, err := db.Exec("INSERT INTO `groups` (name, description, updated_by, deleted_at) VALUES ('Deleted', '', 1, NOW())")
	require.NoError(t, err)

	deletedID, err := result.LastInsertId()
	require.NoError(t, err)

	defer db.Exec("DELETE FROM `groups` WHERE id = ?", deletedID) //nolint:errcheck

	repo := mysqlRepo.NewGroupRepository(db)
	_, total, err := repo.ListGroups(context.Background(), "", 1, 100)

	assert.NoError(t, err)
	assert.Equal(t, countActiveGroups(t, db), total) // g999 excluded
}

func TestStore_OK(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	// Use user id=1 (Taro Yamada) as the creator.
	const creatorID = uint64(1)

	repo := mysqlRepo.NewGroupRepository(db)
	g, err := repo.Store(context.Background(), "New Group", "A new group description", creatorID)

	require.NoError(t, err)
	assert.NotZero(t, g.ID)
	assert.Equal(t, "New Group", g.Name)
	assert.Equal(t, "A new group description", g.Description)
	assert.Equal(t, 1, g.MemberCount)

	// Verify group_members row was inserted.
	var memberCount int
	require.NoError(t, db.QueryRow(
		"SELECT COUNT(*) FROM group_members WHERE group_id = ? AND user_id = ?",
		g.ID, creatorID,
	).Scan(&memberCount))
	assert.Equal(t, 1, memberCount)

	// Cleanup (group_members cascade or delete explicitly)
	db.Exec("DELETE FROM group_members WHERE group_id = ?", g.ID) //nolint:errcheck
	db.Exec("DELETE FROM `groups` WHERE id = ?", g.ID)            //nolint:errcheck
}

func TestStore_DBError(t *testing.T) {
	db := testDB(t)
	// Close the DB connection to force an INSERT failure.
	db.Close()

	repo := mysqlRepo.NewGroupRepository(db)
	g, err := repo.Store(context.Background(), "Should Fail", "desc", uint64(1))

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	assert.Equal(t, domain.Group{}, g)
}

func TestStore_GroupsInsertFailed_FKViolation(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	// Use a non-existent user id for updated_by to trigger FK violation on groups INSERT.
	// fk_groups_updated_by enforces that updated_by must reference users(id).
	const nonExistentCreatorID = uint64(888888888)

	repo := mysqlRepo.NewGroupRepository(db)
	g, err := repo.Store(context.Background(), "FK Fail Group", "desc", nonExistentCreatorID)

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	assert.Equal(t, domain.Group{}, g)

	// Verify no group was persisted (rollback succeeded).
	var count int
	require.NoError(t, db.QueryRow(
		"SELECT COUNT(*) FROM `groups` WHERE name = 'FK Fail Group'",
	).Scan(&count))
	assert.Equal(t, 0, count)
}

func TestStore_GroupMembersInsertFailed_Rollback(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	// Use a non-existent user id to trigger FK violation on group_members INSERT.
	const nonExistentUserID = uint64(999999999)

	repo := mysqlRepo.NewGroupRepository(db)
	g, err := repo.Store(context.Background(), "Rollback Group", "desc", nonExistentUserID)

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
	assert.Equal(t, domain.Group{}, g)

	// Verify no group was persisted (rollback succeeded).
	var count int
	require.NoError(t, db.QueryRow(
		"SELECT COUNT(*) FROM `groups` WHERE name = 'Rollback Group'",
	).Scan(&count))
	assert.Equal(t, 0, count)
}

func TestUpdate_OK(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	// Insert a group to update (updated_by=1 is required after migration).
	result, err := db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('Before Update', 'old desc', 1)")
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	defer db.Exec("DELETE FROM `groups` WHERE id = ?", id) //nolint:errcheck

	const updaterID = uint64(1)

	repo := mysqlRepo.NewGroupRepository(db)
	g, err := repo.Update(context.Background(), id, "After Update", "new desc", updaterID)

	require.NoError(t, err)
	assert.Equal(t, uint64(id), g.ID) //nolint:gosec
	assert.Equal(t, "After Update", g.Name)
	assert.Equal(t, "new desc", g.Description)
	assert.Equal(t, 0, g.MemberCount)
}

func TestUpdate_UpdatedByWritten(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	// Insert a group with updated_by=1, then update as user=2.
	result, err := db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('UpdatedBy Test', 'desc', 1)")
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	defer db.Exec("DELETE FROM `groups` WHERE id = ?", id) //nolint:errcheck

	const updaterID = uint64(2)

	repo := mysqlRepo.NewGroupRepository(db)
	_, err = repo.Update(context.Background(), id, "UpdatedBy Test", "desc", updaterID)
	require.NoError(t, err)

	// Verify updated_by was written correctly.
	var updatedBy uint64
	require.NoError(t, db.QueryRow("SELECT updated_by FROM `groups` WHERE id = ?", id).Scan(&updatedBy))
	assert.Equal(t, updaterID, updatedBy)
}

func TestUpdate_NotFound(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := mysqlRepo.NewGroupRepository(db)
	_, err := repo.Update(context.Background(), 999999999, "name", "desc", uint64(1))

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// Case #10: Normal - DELETE succeeds -> deleted_at and updated_by are set correctly in DB.
func TestDelete_OK(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	// Insert a group to delete (updated_by=1 is required after migration).
	result, err := db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('To Delete', 'delete me', 1)")
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	defer db.Exec("DELETE FROM `groups` WHERE id = ?", id) //nolint:errcheck

	const deleterUserID = uint64(99)

	repo := mysqlRepo.NewGroupRepository(db)
	err = repo.Delete(context.Background(), id, deleterUserID)

	require.NoError(t, err)

	// Verify deleted_at is set and updated_by matches the deleter's userID.
	var deletedAt sql.NullTime
	var updatedBy uint64
	row := db.QueryRow("SELECT deleted_at, updated_by FROM `groups` WHERE id = ?", id)
	require.NoError(t, row.Scan(&deletedAt, &updatedBy))
	assert.True(t, deletedAt.Valid)
	assert.Equal(t, deleterUserID, updatedBy)
}

// Case #11: Error - non-existent id -> ErrNotFound.
func TestDelete_NotFound(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := mysqlRepo.NewGroupRepository(db)
	err := repo.Delete(context.Background(), 999999999, uint64(1))

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// Case #12: Error - already soft-deleted group -> ErrNotFound.
func TestDelete_AlreadyDeleted(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	// Insert a group (updated_by=1 is required after migration).
	result, err := db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('Already Deleted', 'desc', 1)")
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	defer db.Exec("DELETE FROM `groups` WHERE id = ?", id) //nolint:errcheck

	// Soft-delete the group directly in DB.
	_, err = db.Exec("UPDATE `groups` SET deleted_at = NOW() WHERE id = ?", id)
	require.NoError(t, err)

	repo := mysqlRepo.NewGroupRepository(db)
	err = repo.Delete(context.Background(), id, uint64(1))

	assert.ErrorIs(t, err, domain.ErrNotFound)
}
