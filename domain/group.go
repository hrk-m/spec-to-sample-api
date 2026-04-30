// Package domain defines the core domain models.
package domain

// Group represents a group entity with its member count.
type Group struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MemberCount int    `json:"member_count"`
}

// GroupMemberSource represents a single source group from which a member was found.
type GroupMemberSource struct {
	GroupID   uint64
	GroupName string
}

// GroupMember represents a user who belongs to a group, including all source groups.
// Sources contains all groups (direct or via subgroup) through which the user belongs.
type GroupMember struct {
	ID        uint64
	UUID      string
	FirstName string
	LastName  string
	Sources   []GroupMemberSource
}
