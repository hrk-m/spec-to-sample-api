// Package domain defines the core domain models.
package domain

// GroupRelation represents a parent-child relationship between two groups.
type GroupRelation struct {
	ParentGroupID uint64 `json:"parent_group_id"`
	ChildGroupID  uint64 `json:"child_group_id"`
}
