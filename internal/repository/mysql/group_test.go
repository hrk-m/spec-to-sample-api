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

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
	groups, total, err := repo.ListGroups(context.Background(), "", 1, 10)

	assert.NoError(t, err)
	assert.Equal(t, countActiveGroups(t, db), total)
	assert.Len(t, groups, 10)
}

func TestListGroups_Search(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
	groups, total, err := repo.ListGroups(context.Background(), "001", 1, 10)

	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, groups, 1)
	assert.Equal(t, "Group 001", groups[0].Name)
}

func TestListGroups_SearchWithSpaceSeparatedTokens(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
	groups, total, err := repo.ListGroups(context.Background(), "001 Description", 1, 10)

	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, groups, 1)
	assert.Equal(t, "Group 001", groups[0].Name)
}

func TestListGroups_LastPage(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
	groups, total, err := repo.ListGroups(context.Background(), "", 3, 10)

	assert.NoError(t, err)
	assert.Equal(t, countActiveGroups(t, db), total)
	assert.Len(t, groups, 10)
}

func TestListGroups_MemberCount(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
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

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
	_, total, err := repo.ListGroups(context.Background(), "", 1, 100)

	assert.NoError(t, err)
	assert.Equal(t, countActiveGroups(t, db), total) // g999 excluded
}

func TestStore_OK(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	// Use user id=1 (Taro Yamada) as the creator.
	const creatorID = uint64(1)

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
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

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
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

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
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

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
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

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
	g, err := repo.Update(context.Background(), uint64(id), "After Update", "new desc", updaterID) //nolint:gosec

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

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
	_, err = repo.Update(context.Background(), uint64(id), "UpdatedBy Test", "desc", updaterID) //nolint:gosec
	require.NoError(t, err)

	// Verify updated_by was written correctly.
	var updatedBy uint64
	require.NoError(t, db.QueryRow("SELECT updated_by FROM `groups` WHERE id = ?", id).Scan(&updatedBy))
	assert.Equal(t, updaterID, updatedBy)
}

func TestUpdate_NotFound(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
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

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
	err = repo.Delete(context.Background(), uint64(id), deleterUserID) //nolint:gosec

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

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
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

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
	err = repo.Delete(context.Background(), uint64(id), uint64(1)) //nolint:gosec

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// TestListChildren_DirectMembersOnly tests that member_count reflects only the child group's
// direct members when the child group has no descendants.
// Seed: group 5 -> group 6; group 6 has 1 member (user 3).
func TestListChildren_DirectMembersOnly(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	const parentGroupID = uint64(5)

	repo := mysqlRepo.NewGroupRelationRepository(db, testLogger())
	groups, err := repo.ListChildren(context.Background(), parentGroupID)

	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, uint64(6), groups[0].ID)
	assert.Equal(t, 1, groups[0].MemberCount)
}

// TestListChildren_RecursiveMemberCount tests that member_count includes members of descendant
// groups. A temporary hierarchy is built: parent -> child -> grandchild.
// child has 1 direct member; grandchild has 2 direct members.
// Expected member_count for child: 3 (1 + 2).
func TestListChildren_RecursiveMemberCount(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	// Create parent group.
	res, err := db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('RC Parent', 'desc', 1)")
	require.NoError(t, err)
	parentID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", parentID) //nolint:errcheck

	// Create child group.
	res, err = db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('RC Child', 'desc', 1)")
	require.NoError(t, err)
	childID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", childID) //nolint:errcheck

	// Create grandchild group.
	res, err = db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('RC Grandchild', 'desc', 1)")
	require.NoError(t, err)
	grandchildID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", grandchildID) //nolint:errcheck

	// parent -> child -> grandchild
	_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?, ?)", parentID, childID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?", parentID, childID) //nolint:errcheck

	_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?, ?)", childID, grandchildID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?", childID, grandchildID) //nolint:errcheck

	// Add 1 direct member to child (user 20).
	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?, 20)", childID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = 20", childID) //nolint:errcheck

	// Add 2 direct members to grandchild (user 21, user 22).
	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?, 21)", grandchildID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = 21", grandchildID) //nolint:errcheck

	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?, 22)", grandchildID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = 22", grandchildID) //nolint:errcheck

	repo := mysqlRepo.NewGroupRelationRepository(db, testLogger())
	groups, err := repo.ListChildren(context.Background(), uint64(parentID)) //nolint:gosec

	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, uint64(childID), groups[0].ID) //nolint:gosec
	// child: 1 direct member + 2 grandchild members = 3
	assert.Equal(t, 3, groups[0].MemberCount)
}

// TestListChildren_DeduplicatesSharedMembers tests that a user who belongs to both child and
// grandchild groups is counted only once (DISTINCT).
func TestListChildren_DeduplicatesSharedMembers(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	// Create parent group.
	res, err := db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('Dedup Parent', 'desc', 1)")
	require.NoError(t, err)
	parentID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", parentID) //nolint:errcheck

	// Create child group.
	res, err = db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('Dedup Child', 'desc', 1)")
	require.NoError(t, err)
	childID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", childID) //nolint:errcheck

	// Create grandchild group.
	res, err = db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('Dedup Grandchild', 'desc', 1)")
	require.NoError(t, err)
	grandchildID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", grandchildID) //nolint:errcheck

	// parent -> child -> grandchild
	_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?, ?)", parentID, childID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?", parentID, childID) //nolint:errcheck

	_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?, ?)", childID, grandchildID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?", childID, grandchildID) //nolint:errcheck

	// user 20 belongs to both child and grandchild.
	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?, 20)", childID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = 20", childID) //nolint:errcheck

	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?, 20)", grandchildID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = 20", grandchildID) //nolint:errcheck

	repo := mysqlRepo.NewGroupRelationRepository(db, testLogger())
	groups, err := repo.ListChildren(context.Background(), uint64(parentID)) //nolint:gosec

	require.NoError(t, err)
	require.Len(t, groups, 1)
	// user 20 counted only once despite belonging to both groups
	assert.Equal(t, 1, groups[0].MemberCount)
}

// TestListChildren_NoChildren tests that an empty slice is returned when the group has no children.
func TestListChildren_NoChildren(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	// group 1 has no children in the seed data.
	const parentGroupID = uint64(1)

	repo := mysqlRepo.NewGroupRelationRepository(db, testLogger())
	groups, err := repo.ListChildren(context.Background(), parentGroupID)

	require.NoError(t, err)
	assert.Empty(t, groups)
}

// TestListChildren_DBError tests that ErrInternalServerError is returned on DB failure.
func TestListChildren_DBError(t *testing.T) {
	db := testDB(t)
	db.Close()

	repo := mysqlRepo.NewGroupRelationRepository(db, testLogger())
	_, err := repo.ListChildren(context.Background(), uint64(1))

	assert.ErrorIs(t, err, domain.ErrInternalServerError)
}
