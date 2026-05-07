// Package mysql provides MySQL implementations of repository interfaces.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/go-sql-driver/mysql"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// GroupRelationRepository is a MySQL implementation of group.GroupRelationRepository.
type GroupRelationRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewGroupRelationRepository returns a new GroupRelationRepository.
func NewGroupRelationRepository(db *sql.DB, logger *slog.Logger) *GroupRelationRepository {
	return &GroupRelationRepository{db: db, logger: logger}
}

// GetAncestorIDs returns all ancestor group IDs of the given group using a recursive CTE.
func (r *GroupRelationRepository) GetAncestorIDs(ctx context.Context, groupID uint64) ([]uint64, error) {
	query := `
WITH RECURSIVE ancestors AS (
  SELECT parent_group_id AS id
  FROM group_relations
  WHERE child_group_id = ?
  UNION ALL
  SELECT gr.parent_group_id
  FROM group_relations gr
  INNER JOIN ancestors a ON gr.child_group_id = a.id
)
SELECT id FROM ancestors`

	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		r.logger.ErrorContext(ctx, "GetAncestorIDs query failed", "error", err)

		return nil, domain.ErrInternalServerError
	}
	defer func() { _ = rows.Close() }()

	var ids []uint64

	for rows.Next() {
		var id uint64
		if scanErr := rows.Scan(&id); scanErr != nil {
			r.logger.ErrorContext(ctx, "GetAncestorIDs scan failed", "error", scanErr)

			return nil, domain.ErrInternalServerError
		}

		ids = append(ids, id)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		r.logger.ErrorContext(ctx, "GetAncestorIDs rows error", "error", rowsErr)

		return nil, domain.ErrInternalServerError
	}

	if ids == nil {
		ids = []uint64{}
	}

	return ids, nil
}

// GetDescendantIDs returns all descendant group IDs of the given group using a recursive CTE.
func (r *GroupRelationRepository) GetDescendantIDs(ctx context.Context, groupID uint64) ([]uint64, error) {
	query := `
WITH RECURSIVE descendants AS (
  SELECT child_group_id AS id
  FROM group_relations
  WHERE parent_group_id = ?
  UNION ALL
  SELECT gr.child_group_id
  FROM group_relations gr
  INNER JOIN descendants d ON gr.parent_group_id = d.id
)
SELECT id FROM descendants`

	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		r.logger.ErrorContext(ctx, "GetDescendantIDs query failed", "error", err)

		return nil, domain.ErrInternalServerError
	}
	defer func() { _ = rows.Close() }()

	var ids []uint64

	for rows.Next() {
		var id uint64
		if scanErr := rows.Scan(&id); scanErr != nil {
			r.logger.ErrorContext(ctx, "GetDescendantIDs scan failed", "error", scanErr)

			return nil, domain.ErrInternalServerError
		}

		ids = append(ids, id)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		r.logger.ErrorContext(ctx, "GetDescendantIDs rows error", "error", rowsErr)

		return nil, domain.ErrInternalServerError
	}

	if ids == nil {
		ids = []uint64{}
	}

	return ids, nil
}

// CountComponentGroups returns the total number of distinct groups in the connected component
// that contains the given group (using an undirected BFS/CTE traversal).
func (r *GroupRelationRepository) CountComponentGroups(ctx context.Context, groupID uint64) (int, error) {
	query := `
WITH RECURSIVE component AS (
  SELECT ? AS id
  UNION
  SELECT gr.child_group_id
  FROM group_relations gr
  INNER JOIN component c ON gr.parent_group_id = c.id
  UNION
  SELECT gr.parent_group_id
  FROM group_relations gr
  INNER JOIN component c ON gr.child_group_id = c.id
)
SELECT COUNT(*) FROM component`

	var count int
	if err := r.db.QueryRowContext(ctx, query, groupID).Scan(&count); err != nil {
		r.logger.ErrorContext(ctx, "CountComponentGroups query failed", "error", err)

		return 0, domain.ErrInternalServerError
	}

	return count, nil
}

// MaxDepthInComponent returns the maximum path length (number of nodes) from any root to any leaf
// in the component after hypothetically adding the edge parentGroupID → childGroupID.
// The returned value represents the node count of the deepest path.
func (r *GroupRelationRepository) MaxDepthInComponent(ctx context.Context, parentGroupID, childGroupID uint64) (int, error) {
	query := `
WITH RECURSIVE
-- Temporarily include the new edge.
edges AS (
  SELECT parent_group_id, child_group_id FROM group_relations
  UNION ALL
  SELECT ? AS parent_group_id, ? AS child_group_id
),
-- Find all roots (nodes with no parent in the edge set).
roots AS (
  SELECT DISTINCT parent_group_id AS id
  FROM edges
  WHERE parent_group_id NOT IN (SELECT child_group_id FROM edges)
),
-- BFS from each root, counting depth (1-indexed node count).
paths AS (
  SELECT id, 1 AS depth FROM roots
  UNION ALL
  SELECT e.child_group_id, p.depth + 1
  FROM edges e
  INNER JOIN paths p ON e.parent_group_id = p.id
)
SELECT COALESCE(MAX(depth), 1) FROM paths`

	var maxDepth int
	if err := r.db.QueryRowContext(ctx, query, parentGroupID, childGroupID).Scan(&maxDepth); err != nil {
		r.logger.ErrorContext(ctx, "MaxDepthInComponent query failed", "error", err)

		return 0, domain.ErrInternalServerError
	}

	return maxDepth, nil
}

// ListChildren returns the direct child groups of the given parent group.
// member_count includes members of all descendant groups (recursive).
func (r *GroupRelationRepository) ListChildren(ctx context.Context, parentGroupID uint64) ([]domain.Group, error) {
	query := `
WITH RECURSIVE child_descendants AS (
  SELECT child_group_id AS root_id, child_group_id AS node_id
  FROM group_relations
  WHERE parent_group_id = ?
  UNION ALL
  SELECT cd.root_id, gr.child_group_id
  FROM group_relations gr
  INNER JOIN child_descendants cd ON gr.parent_group_id = cd.node_id
)
SELECT g.id, g.name, g.description,
       COUNT(DISTINCT gm.user_id) AS member_count
FROM (SELECT DISTINCT root_id FROM child_descendants) AS direct_children
JOIN ` + "`groups`" + ` g ON g.id = direct_children.root_id
LEFT JOIN child_descendants cd ON cd.root_id = direct_children.root_id
LEFT JOIN group_members gm ON gm.group_id = cd.node_id
WHERE g.deleted_at IS NULL
GROUP BY g.id, g.name, g.description
ORDER BY g.id`

	rows, err := r.db.QueryContext(ctx, query, parentGroupID)
	if err != nil {
		r.logger.ErrorContext(ctx, "ListChildren query failed", "error", err)

		return nil, domain.ErrInternalServerError
	}
	defer func() { _ = rows.Close() }()

	var groups []domain.Group

	for rows.Next() {
		var g domain.Group
		if scanErr := rows.Scan(&g.ID, &g.Name, &g.Description, &g.MemberCount); scanErr != nil {
			r.logger.ErrorContext(ctx, "ListChildren scan failed", "error", scanErr)

			return nil, domain.ErrInternalServerError
		}

		groups = append(groups, g)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		r.logger.ErrorContext(ctx, "ListChildren rows error", "error", rowsErr)

		return nil, domain.ErrInternalServerError
	}

	if groups == nil {
		groups = []domain.Group{}
	}

	return groups, nil
}

// DeleteRelation removes the parent-child relation from group_relations.
// It returns domain.ErrNotFound if the relation does not exist (RowsAffected == 0).
func (r *GroupRelationRepository) DeleteRelation(ctx context.Context, parentGroupID, childGroupID uint64) error {
	result, err := r.db.ExecContext(
		ctx,
		"DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?",
		parentGroupID,
		childGroupID,
	)
	if err != nil {
		r.logger.ErrorContext(ctx, "DeleteRelation exec failed", "error", err)

		return domain.ErrInternalServerError
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.ErrorContext(ctx, "DeleteRelation rows affected failed", "error", err)

		return domain.ErrInternalServerError
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// CreateRelation inserts a new parent-child relation into group_relations.
func (r *GroupRelationRepository) CreateRelation(ctx context.Context, parentGroupID, childGroupID uint64) (domain.GroupRelation, error) {
	_, err := r.db.ExecContext(
		ctx,
		"INSERT INTO group_relations (parent_group_id, child_group_id) VALUES (?, ?)",
		parentGroupID,
		childGroupID,
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.GroupRelation{}, domain.ErrConflict
		}

		r.logger.ErrorContext(ctx, "CreateRelation exec failed", "error", err)

		return domain.GroupRelation{}, domain.ErrInternalServerError
	}

	return domain.GroupRelation{
		ParentGroupID: parentGroupID,
		ChildGroupID:  childGroupID,
	}, nil
}
