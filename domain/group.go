// Package domain defines the core domain models.
package domain

// Group represents a group entity with its member count.
type Group struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MemberCount int    `json:"member_count"`
}

// SourceGroup represents a single group from which a member was found.
type SourceGroup struct {
	GroupID   uint64
	GroupName string
}

// GroupMember represents a user who belongs to a group, including all source groups.
// SourceGroups contains all groups (direct or via subgroup) through which the user belongs.
type GroupMember struct {
	ID           uint64
	UUID         string
	FirstName    string
	LastName     string
	SourceGroups []SourceGroup
}

// GroupRelation represents a parent-child relationship between two groups.
type GroupRelation struct {
	ParentGroupID uint64 `json:"parent_group_id"`
	ChildGroupID  uint64 `json:"child_group_id"`
}
