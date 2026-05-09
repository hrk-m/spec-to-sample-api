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

// TestSelectGroups_L1 tests that member_count equals 3 when G1 has 3 direct members and no descendants.
func TestSelectGroups_L1(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	res, err := db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L1 G1', 'desc', 1)")
	require.NoError(t, err)
	g1ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g1ID) //nolint:errcheck

	// Add 3 direct members (users 1,2,3) to G1.
	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1),(?,2),(?,3)", g1ID, g1ID, g1ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ?", g1ID) //nolint:errcheck

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
	groups, _, err := repo.ListGroups(context.Background(), "L1 G1", 10, 0)

	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, 3, groups[0].MemberCount)
}

// TestSelectGroups_L2 tests G1 with 2 direct members and child G2 with 3 members (no overlap).
// Expected: G1 = 5, G2 = 3.
func TestSelectGroups_L2(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	res, err := db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L2 G1', 'desc', 1)")
	require.NoError(t, err)
	g1ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g1ID) //nolint:errcheck

	res, err = db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L2 G2', 'desc', 1)")
	require.NoError(t, err)
	g2ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g2ID) //nolint:errcheck

	_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?,?)", g1ID, g2ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?", g1ID, g2ID) //nolint:errcheck

	// G1: users 1,2 (2 direct members).
	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1),(?,2)", g1ID, g1ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ?", g1ID) //nolint:errcheck

	// G2: users 3,4,5 (3 direct members).
	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,3),(?,4),(?,5)", g2ID, g2ID, g2ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ?", g2ID) //nolint:errcheck

	repo := mysqlRepo.NewGroupRepository(db, testLogger())

	// Check G1 = 5 (2 direct + 3 from G2).
	g1Groups, _, err := repo.ListGroups(context.Background(), "L2 G1", 10, 0)
	require.NoError(t, err)
	require.Len(t, g1Groups, 1)
	assert.Equal(t, 5, g1Groups[0].MemberCount)

	// Check G2 = 3.
	g2Groups, _, err := repo.ListGroups(context.Background(), "L2 G2", 10, 0)
	require.NoError(t, err)
	require.Len(t, g2Groups, 1)
	assert.Equal(t, 3, g2Groups[0].MemberCount)
}

// TestSelectGroups_L3 tests a 3-level hierarchy G1->G2->G3 with 2 members each (no overlap).
// Expected: G1 = 6, G2 = 4, G3 = 2.
func TestSelectGroups_L3(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	res, err := db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L3 G1', 'desc', 1)")
	require.NoError(t, err)
	g1ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g1ID) //nolint:errcheck

	res, err = db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L3 G2', 'desc', 1)")
	require.NoError(t, err)
	g2ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g2ID) //nolint:errcheck

	res, err = db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L3 G3', 'desc', 1)")
	require.NoError(t, err)
	g3ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g3ID) //nolint:errcheck

	_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?,?)", g1ID, g2ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?", g1ID, g2ID) //nolint:errcheck

	_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?,?)", g2ID, g3ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?", g2ID, g3ID) //nolint:errcheck

	// G1: users 1,2. G2: users 3,4. G3: users 5,6.
	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1),(?,2)", g1ID, g1ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ?", g1ID) //nolint:errcheck

	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,3),(?,4)", g2ID, g2ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ?", g2ID) //nolint:errcheck

	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,5),(?,6)", g3ID, g3ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ?", g3ID) //nolint:errcheck

	repo := mysqlRepo.NewGroupRepository(db, testLogger())

	// G1 = 6 (users 1,2,3,4,5,6).
	g1Groups, _, err := repo.ListGroups(context.Background(), "L3 G1", 10, 0)
	require.NoError(t, err)
	require.Len(t, g1Groups, 1)
	assert.Equal(t, 6, g1Groups[0].MemberCount)

	// G2 = 4 (users 3,4,5,6).
	g2Groups, _, err := repo.ListGroups(context.Background(), "L3 G2", 10, 0)
	require.NoError(t, err)
	require.Len(t, g2Groups, 1)
	assert.Equal(t, 4, g2Groups[0].MemberCount)

	// G3 = 2 (users 5,6).
	g3Groups, _, err := repo.ListGroups(context.Background(), "L3 G3", 10, 0)
	require.NoError(t, err)
	require.Len(t, g3Groups, 1)
	assert.Equal(t, 2, g3Groups[0].MemberCount)
}

// TestSelectGroups_L4 tests that a user belonging to both G1 and child G2 is counted once.
// G1: users A(1),B(2). G2: users A(1),C(3). Expected G1 = 3 (A,B,C unique).
func TestSelectGroups_L4(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	res, err := db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L4 G1', 'desc', 1)")
	require.NoError(t, err)
	g1ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g1ID) //nolint:errcheck

	res, err = db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L4 G2', 'desc', 1)")
	require.NoError(t, err)
	g2ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g2ID) //nolint:errcheck

	_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?,?)", g1ID, g2ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?", g1ID, g2ID) //nolint:errcheck

	// G1: user 1 (A), user 2 (B).
	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1),(?,2)", g1ID, g1ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ?", g1ID) //nolint:errcheck

	// G2: user 1 (A), user 3 (C).
	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1),(?,3)", g2ID, g2ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ?", g2ID) //nolint:errcheck

	repo := mysqlRepo.NewGroupRepository(db, testLogger())

	// G1 = 3 (users 1,2,3 unique).
	g1Groups, _, err := repo.ListGroups(context.Background(), "L4 G1", 10, 0)
	require.NoError(t, err)
	require.Len(t, g1Groups, 1)
	assert.Equal(t, 3, g1Groups[0].MemberCount)
}

// TestSelectGroups_L5 tests that a group with no members and no descendants returns member_count = 0.
func TestSelectGroups_L5(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	res, err := db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L5 G1', 'desc', 1)")
	require.NoError(t, err)
	g1ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g1ID) //nolint:errcheck

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
	groups, _, err := repo.ListGroups(context.Background(), "L5 G1", 10, 0)

	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, 0, groups[0].MemberCount)
}

// TestSelectGroups_L6 tests G1 with 2 direct members and child G2 with no members.
// Expected: G1 = 2 (child contributes 0 additional members).
func TestSelectGroups_L6(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	res, err := db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L6 G1', 'desc', 1)")
	require.NoError(t, err)
	g1ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g1ID) //nolint:errcheck

	res, err = db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L6 G2', 'desc', 1)")
	require.NoError(t, err)
	g2ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g2ID) //nolint:errcheck

	_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?,?)", g1ID, g2ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?", g1ID, g2ID) //nolint:errcheck

	// G1: users 1,2 (2 direct members). G2: no members.
	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1),(?,2)", g1ID, g1ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ?", g1ID) //nolint:errcheck

	repo := mysqlRepo.NewGroupRepository(db, testLogger())
	groups, _, err := repo.ListGroups(context.Background(), "L6 G1", 10, 0)

	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, 2, groups[0].MemberCount)
}

// TestSelectGroups_L7 tests that q filter is applied correctly while recursive unique count is preserved.
// G1 (name has "L7") has child G2; G1: users 1,2 / G2: users 3,4,5. Expected G1 = 5.
func TestSelectGroups_L7(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	res, err := db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L7 G1 unique', 'desc', 1)")
	require.NoError(t, err)
	g1ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g1ID) //nolint:errcheck

	res, err = db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L7 G2 child', 'desc', 1)")
	require.NoError(t, err)
	g2ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g2ID) //nolint:errcheck

	_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?,?)", g1ID, g2ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?", g1ID, g2ID) //nolint:errcheck

	// G1: users 1,2. G2: users 3,4,5.
	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1),(?,2)", g1ID, g1ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ?", g1ID) //nolint:errcheck

	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,3),(?,4),(?,5)", g2ID, g2ID, g2ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ?", g2ID) //nolint:errcheck

	repo := mysqlRepo.NewGroupRepository(db, testLogger())

	// q="L7 G1 unique" filters to G1 only; member_count must include G2's members.
	groups, _, err := repo.ListGroups(context.Background(), "L7 G1 unique", 10, 0)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, 5, groups[0].MemberCount)
}

// TestSelectGroups_L8 tests a DAG where G1 and G2 both have G3 as a child.
// G3 has 2 members (users 1,2). Expected: G1 = 2, G2 = 2 (G3 members counted once each).
func TestSelectGroups_L8(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	res, err := db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L8 G1', 'desc', 1)")
	require.NoError(t, err)
	g1ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g1ID) //nolint:errcheck

	res, err = db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L8 G2', 'desc', 1)")
	require.NoError(t, err)
	g2ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g2ID) //nolint:errcheck

	res, err = db.Exec("INSERT INTO `groups` (name, description, updated_by) VALUES ('L8 G3', 'desc', 1)")
	require.NoError(t, err)
	g3ID, err := res.LastInsertId()
	require.NoError(t, err)
	defer db.Exec("DELETE FROM `groups` WHERE id = ?", g3ID) //nolint:errcheck

	// Both G1 and G2 point to G3.
	_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?,?)", g1ID, g3ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?", g1ID, g3ID) //nolint:errcheck

	_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?,?)", g2ID, g3ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?", g2ID, g3ID) //nolint:errcheck

	// G3: users 1,2. G1 and G2 have no direct members.
	_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1),(?,2)", g3ID, g3ID)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM group_members WHERE group_id = ?", g3ID) //nolint:errcheck

	repo := mysqlRepo.NewGroupRepository(db, testLogger())

	// G1 = 2 (G3's members).
	g1Groups, _, err := repo.ListGroups(context.Background(), "L8 G1", 10, 0)
	require.NoError(t, err)
	require.Len(t, g1Groups, 1)
	assert.Equal(t, 2, g1Groups[0].MemberCount)

	// G2 = 2 (G3's members, uniquely counted even though G3 is shared).
	g2Groups, _, err := repo.ListGroups(context.Background(), "L8 G2", 10, 0)
	require.NoError(t, err)
	require.Len(t, g2Groups, 1)
	assert.Equal(t, 2, g2Groups[0].MemberCount)
}

// TestMysqlGroupRepository_ListGroupMembers_DuplicateCount verifies the new duplicate_count
// semantics: SUM(CASE WHEN JSON_LENGTH(source_groups) >= 2 THEN 1 ELSE 0 END) OVER()
// i.e. the number of unique users belonging to 2+ groups/subgroups.
func TestMysqlGroupRepository_ListGroupMembers_DuplicateCount(t *testing.T) {
	// Base IDs for dynamically created groups in these tests (high range to avoid seed collisions).
	const baseGroupID = 9000

	tests := []struct {
		name            string
		setup           func(t *testing.T, db *sql.DB) (parentID uint64, cleanup func())
		q               string
		excludeGroupIDs []uint64
		wantDuplicate   int
	}{
		{
			name: "T1_everyone_single_group_only",
			setup: func(t *testing.T, db *sql.DB) (uint64, func()) {
				t.Helper()

				// Create parent group with 3 direct members (users 1,2,3). No subgroups.
				res, err := db.Exec("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T1 Parent', 'desc', 1)", baseGroupID+1)
				require.NoError(t, err)
				parentID, _ := res.LastInsertId()

				_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1),(?,2),(?,3)", parentID, parentID, parentID)
				require.NoError(t, err)

				cleanup := func() {
					db.Exec("DELETE FROM group_members WHERE group_id = ?", parentID)         //nolint:errcheck
					db.Exec("DELETE FROM `groups` WHERE id = ?", parentID)                    //nolint:errcheck
				}

				return uint64(parentID), cleanup //nolint:gosec
			},
			q:               "",
			excludeGroupIDs: nil,
			wantDuplicate:   0,
		},
		{
			name: "T2_one_user_in_parent_and_subgroup",
			setup: func(t *testing.T, db *sql.DB) (uint64, func()) {
				t.Helper()

				// parent: user 1 is a direct member.
				// subA: user 1 and user 2 are members.
				// → user 1 belongs to parent + subA (2 groups), user 2 belongs to subA only.
				res, err := db.Exec("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T2 Parent', 'desc', 1)", baseGroupID+10)
				require.NoError(t, err)
				parentID, _ := res.LastInsertId()

				res, err = db.Exec("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T2 SubA', 'desc', 1)", baseGroupID+11)
				require.NoError(t, err)
				subAID, _ := res.LastInsertId()

				_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?, ?)", parentID, subAID)
				require.NoError(t, err)

				_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1)", parentID)
				require.NoError(t, err)

				_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1),(?,2)", subAID, subAID)
				require.NoError(t, err)

				cleanup := func() {
					db.Exec("DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?", parentID, subAID) //nolint:errcheck
					db.Exec("DELETE FROM group_members WHERE group_id IN (?,?)", parentID, subAID)                            //nolint:errcheck
					db.Exec("DELETE FROM `groups` WHERE id IN (?,?)", parentID, subAID)                                      //nolint:errcheck
				}

				return uint64(parentID), cleanup //nolint:gosec
			},
			q:               "",
			excludeGroupIDs: nil,
			wantDuplicate:   1,
		},
		{
			name: "T3_two_users_each_in_multiple_groups",
			setup: func(t *testing.T, db *sql.DB) (uint64, func()) {
				t.Helper()

				// parent: user 1. subA: user 1, user 2. subB: user 1, user 2.
				// → user 1 in parent+subA+subB (3 groups), user 2 in subA+subB (2 groups).
				res, err := db.Exec("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T3 Parent', 'desc', 1)", baseGroupID+20)
				require.NoError(t, err)
				parentID, _ := res.LastInsertId()

				res, err = db.Exec("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T3 SubA', 'desc', 1)", baseGroupID+21)
				require.NoError(t, err)
				subAID, _ := res.LastInsertId()

				res, err = db.Exec("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T3 SubB', 'desc', 1)", baseGroupID+22)
				require.NoError(t, err)
				subBID, _ := res.LastInsertId()

				_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?,?),(?,?)", parentID, subAID, parentID, subBID)
				require.NoError(t, err)

				_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1)", parentID)
				require.NoError(t, err)

				_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1),(?,2)", subAID, subAID)
				require.NoError(t, err)

				_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1),(?,2)", subBID, subBID)
				require.NoError(t, err)

				cleanup := func() {
					db.Exec("DELETE FROM group_relations WHERE parent_group_id = ?", parentID)    //nolint:errcheck
					db.Exec("DELETE FROM group_members WHERE group_id IN (?,?,?)", parentID, subAID, subBID) //nolint:errcheck
					db.Exec("DELETE FROM `groups` WHERE id IN (?,?,?)", parentID, subAID, subBID) //nolint:errcheck
				}

				return uint64(parentID), cleanup //nolint:gosec
			},
			q:               "",
			excludeGroupIDs: nil,
			wantDuplicate:   2,
		},
		{
			name: "T4_one_user_in_5_subgroups_counts_as_1",
			setup: func(t *testing.T, db *sql.DB) (uint64, func()) {
				t.Helper()

				// user 1 in parent+sub1+sub2+sub3+sub4 (5 groups). Should count as 1.
				res, err := db.Exec("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T4 Parent', 'desc', 1)", baseGroupID+30)
				require.NoError(t, err)
				parentID, _ := res.LastInsertId()

				subIDs := make([]int64, 4)
				for i := range subIDs {
					res, err = db.Exec(fmt.Sprintf("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T4 Sub%d', 'desc', 1)", i+1), baseGroupID+31+i)
					require.NoError(t, err)
					subIDs[i], _ = res.LastInsertId()
				}

				for _, subID := range subIDs {
					_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?,?)", parentID, subID)
					require.NoError(t, err)
				}

				// user 1 in parent and all 4 subs
				_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1)", parentID)
				require.NoError(t, err)

				for _, subID := range subIDs {
					_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1)", subID)
					require.NoError(t, err)
				}

				cleanup := func() {
					db.Exec("DELETE FROM group_relations WHERE parent_group_id = ?", parentID) //nolint:errcheck
					allIDs := []int64{int64(parentID)}
					allIDs = append(allIDs, subIDs...)
					for _, id := range allIDs {
						db.Exec("DELETE FROM group_members WHERE group_id = ?", id) //nolint:errcheck
					}
					for _, id := range allIDs {
						db.Exec("DELETE FROM `groups` WHERE id = ?", id) //nolint:errcheck
					}
				}

				return uint64(parentID), cleanup //nolint:gosec
			},
			q:               "",
			excludeGroupIDs: nil,
			wantDuplicate:   1,
		},
		{
			name: "T5_empty_group_returns_0",
			setup: func(t *testing.T, db *sql.DB) (uint64, func()) {
				t.Helper()

				// A fresh parent with no members and no subgroups.
				res, err := db.Exec("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T5 Parent', 'desc', 1)", baseGroupID+40)
				require.NoError(t, err)
				parentID, _ := res.LastInsertId()

				cleanup := func() {
					db.Exec("DELETE FROM `groups` WHERE id = ?", parentID) //nolint:errcheck
				}

				return uint64(parentID), cleanup //nolint:gosec
			},
			q:               "",
			excludeGroupIDs: nil,
			wantDuplicate:   0,
		},
		{
			name: "T6_q_removes_duplicate_user",
			setup: func(t *testing.T, db *sql.DB) (uint64, func()) {
				t.Helper()

				// user 1 (TaroYamada — duplicate) in parent+subA.
				// user 2 (HanakoSuzuki — single) in subA only.
				// q="Suzuki" → only user 2 matches → duplicate_count=0.
				res, err := db.Exec("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T6 Parent', 'desc', 1)", baseGroupID+50)
				require.NoError(t, err)
				parentID, _ := res.LastInsertId()

				res, err = db.Exec("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T6 SubA', 'desc', 1)", baseGroupID+51)
				require.NoError(t, err)
				subAID, _ := res.LastInsertId()

				_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?,?)", parentID, subAID)
				require.NoError(t, err)

				_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1)", parentID)
				require.NoError(t, err)

				_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1),(?,2)", subAID, subAID)
				require.NoError(t, err)

				cleanup := func() {
					db.Exec("DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?", parentID, subAID) //nolint:errcheck
					db.Exec("DELETE FROM group_members WHERE group_id IN (?,?)", parentID, subAID)                            //nolint:errcheck
					db.Exec("DELETE FROM `groups` WHERE id IN (?,?)", parentID, subAID)                                      //nolint:errcheck
				}

				return uint64(parentID), cleanup //nolint:gosec
			},
			// user 2's search_key = "HanakoSuzukiSuzukiHanako" → "Suzuki" matches only user 2
			q:               "Suzuki",
			excludeGroupIDs: nil,
			wantDuplicate:   0,
		},
		{
			name: "T7_exclude_reduces_to_1_group",
			setup: func(t *testing.T, db *sql.DB) (uint64, func()) {
				t.Helper()

				// user 1 in parent+subA. exclude subA → only parent remains → source_groups len=1.
				res, err := db.Exec("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T7 Parent', 'desc', 1)", baseGroupID+60)
				require.NoError(t, err)
				parentID, _ := res.LastInsertId()

				res, err = db.Exec("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T7 SubA', 'desc', 1)", baseGroupID+61)
				require.NoError(t, err)
				subAID, _ := res.LastInsertId()

				_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?,?)", parentID, subAID)
				require.NoError(t, err)

				_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1)", parentID)
				require.NoError(t, err)

				_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1)", subAID)
				require.NoError(t, err)

				cleanup := func() {
					db.Exec("DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?", parentID, subAID) //nolint:errcheck
					db.Exec("DELETE FROM group_members WHERE group_id IN (?,?)", parentID, subAID)                            //nolint:errcheck
					db.Exec("DELETE FROM `groups` WHERE id IN (?,?)", parentID, subAID)                                      //nolint:errcheck
				}

				return uint64(parentID), cleanup //nolint:gosec
			},
			q: "",
			// excludeGroupIDs is set dynamically in the test loop based on subAID
			// We use a sentinel 0 and replace it below — handled inline via a wrapper.
			excludeGroupIDs: nil, // set at test run time
			wantDuplicate:   0,
		},
		{
			name: "T8_exclude_leaves_2_groups",
			setup: func(t *testing.T, db *sql.DB) (uint64, func()) {
				t.Helper()

				// user 1 in parent+subA+subB. exclude subA → parent+subB remain (2 groups).
				res, err := db.Exec("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T8 Parent', 'desc', 1)", baseGroupID+70)
				require.NoError(t, err)
				parentID, _ := res.LastInsertId()

				res, err = db.Exec("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T8 SubA', 'desc', 1)", baseGroupID+71)
				require.NoError(t, err)
				subAID, _ := res.LastInsertId()

				res, err = db.Exec("INSERT INTO `groups` (id, name, description, updated_by) VALUES (?, 'T8 SubB', 'desc', 1)", baseGroupID+72)
				require.NoError(t, err)
				subBID, _ := res.LastInsertId()

				_, err = db.Exec("INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?,?),(?,?)", parentID, subAID, parentID, subBID)
				require.NoError(t, err)

				_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1)", parentID)
				require.NoError(t, err)

				_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1)", subAID)
				require.NoError(t, err)

				_, err = db.Exec("INSERT INTO group_members (group_id, user_id) VALUES (?,1)", subBID)
				require.NoError(t, err)

				cleanup := func() {
					db.Exec("DELETE FROM group_relations WHERE parent_group_id = ?", parentID)           //nolint:errcheck
					db.Exec("DELETE FROM group_members WHERE group_id IN (?,?,?)", parentID, subAID, subBID) //nolint:errcheck
					db.Exec("DELETE FROM `groups` WHERE id IN (?,?,?)", parentID, subAID, subBID)        //nolint:errcheck
				}

				return uint64(parentID), cleanup //nolint:gosec
			},
			q:               "",
			excludeGroupIDs: nil, // set at test run time
			wantDuplicate:   1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := testDB(t)
			defer db.Close()

			parentID, cleanup := tc.setup(t, db)
			defer cleanup()

			excludeIDs := tc.excludeGroupIDs

			// T7: exclude subA (baseGroupID+61)
			if tc.name == "T7_exclude_reduces_to_1_group" {
				excludeIDs = []uint64{uint64(baseGroupID + 61)} //nolint:gosec
			}

			// T8: exclude subA (baseGroupID+71)
			if tc.name == "T8_exclude_leaves_2_groups" {
				excludeIDs = []uint64{uint64(baseGroupID + 71)} //nolint:gosec
			}

			repo := mysqlRepo.NewGroupRepository(db, testLogger())
			_, _, duplicateCount, err := repo.ListGroupMembers(context.Background(), parentID, 500, 0, tc.q, excludeIDs)

			require.NoError(t, err)
			assert.Equal(t, tc.wantDuplicate, duplicateCount, "duplicate_count mismatch for %s", tc.name)
		})
	}
}
