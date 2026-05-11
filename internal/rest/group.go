// Package rest provides HTTP handlers for the API.
package rest

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

// GroupService defines the interface for the group use case.
type GroupService interface {
	ListGroups(ctx context.Context, q string, limit, offset int) ([]domain.Group, int, error)
	GetByID(ctx context.Context, id uint64) (domain.Group, error)
	ListSubgroups(ctx context.Context, id uint64) ([]domain.Group, error)
	ListGroupMembers(ctx context.Context, id uint64, limit, offset int, q string, excludeGroupIDs []uint64) ([]domain.GroupMember, int, int, error)
	Store(ctx context.Context, name, description string, userID uint64) (domain.Group, error)
	Update(ctx context.Context, id uint64, name, description string, userID uint64) (domain.Group, error)
	Delete(ctx context.Context, id uint64, userID uint64) error
	ListNonGroupMembers(ctx context.Context, groupID uint64, limit, offset int, q string) ([]domain.User, int, error)
	AddGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) ([]domain.User, error)
	RemoveGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) error
	CreateSubGroup(ctx context.Context, parentGroupID, childGroupID uint64) (domain.GroupRelation, error)
	DeleteSubGroup(ctx context.Context, parentGroupID, childGroupID uint64) error
}

// GroupHandler handles HTTP requests for the group endpoints.
type GroupHandler struct {
	Service GroupService
}

// NewGroupHandler registers the group routes on the given Echo router group.
func NewGroupHandler(g *echo.Group, svc GroupService) {
	h := &GroupHandler{Service: svc}
	g.GET("/groups", h.ListGroups)
	g.GET("/groups/:id", h.GetByID)
	g.GET("/groups/:id/members", h.ListGroupMembers)
	g.GET("/groups/:id/subgroups", h.ListSubgroups)
	g.GET("/groups/:id/non-members", h.ListNonGroupMembers)
	g.POST("/groups", h.Store)
	g.POST("/groups/:id/members", h.AddGroupMembers)
	g.POST("/groups/:id/subgroups", h.CreateSubGroup)
	g.PUT("/groups/:id", h.Update)
	g.DELETE("/groups/:id", h.Delete)
	g.DELETE("/groups/:id/members", h.DeleteGroupMembers)
	g.DELETE("/groups/:id/subgroups/:childId", h.DeleteSubGroup)
}

type groupListResponse struct {
	Groups []domain.Group `json:"groups"`
	Total  int            `json:"total"`
}

type sourceGroup struct {
	GroupID   uint64 `json:"group_id"`
	GroupName string `json:"group_name"`
}

type groupMember struct {
	ID           uint64        `json:"id"`
	UUID         string        `json:"uuid"`
	FirstName    string        `json:"first_name"`
	LastName     string        `json:"last_name"`
	SourceGroups []sourceGroup `json:"source_groups"`
}

type groupMemberListResponse struct {
	Members        []groupMember `json:"members"`
	Total          int           `json:"total"`
	DuplicateCount int           `json:"duplicate_count"`
}

type storeGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type nonMemberListResponse struct {
	Users []domain.User `json:"users"`
	Total int           `json:"total"`
}

type addGroupMembersRequest struct {
	UserIDs []uint64 `json:"user_ids"`
}

type removeGroupMembersRequest struct {
	UserIDs []uint64 `json:"user_ids"`
}

type addGroupMembersResponse struct {
	Members []domain.User `json:"members"`
}

type createSubGroupRequest struct {
	ChildGroupID uint64 `json:"child_group_id"`
}

type subgroupSummary struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MemberCount int    `json:"member_count"`
}

type subgroupListResponse struct {
	Subgroups []subgroupSummary `json:"subgroups"`
}

type getGroupResponse struct {
	ID          uint64            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	MemberCount int               `json:"member_count"`
	Subgroups   []subgroupSummary `json:"subgroups"`
}

// Store handles POST /api/v1/groups.
func (h *GroupHandler) Store(c echo.Context) error {
	ctx := c.Request().Context()

	var req storeGroupRequest
	if err := c.Bind(&req); err != nil {
		return badRequest(c, err.Error())
	}

	authUser, ok := c.Get("authUser").(domain.User)
	if !ok {
		return respondError(c, domain.ErrUnauthorized)
	}

	result, err := h.Service.Store(ctx, req.Name, req.Description, authUser.ID)
	if err != nil {
		return respondError(c, err)
	}

	return c.JSON(http.StatusCreated, result)
}

// Update handles PUT /api/v1/groups/:id.
func (h *GroupHandler) Update(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	var req updateGroupRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return badRequest(c, bindErr.Error())
	}

	authUser, ok := c.Get("authUser").(domain.User)
	if !ok {
		return respondError(c, domain.ErrUnauthorized)
	}

	result, err := h.Service.Update(ctx, id, req.Name, req.Description, authUser.ID)
	if err != nil {
		return respondError(c, err)
	}

	return c.JSON(http.StatusOK, result)
}

// Delete handles DELETE /api/v1/groups/:id.
func (h *GroupHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	authUser, ok := c.Get("authUser").(domain.User)
	if !ok {
		return respondError(c, domain.ErrUnauthorized)
	}

	if err := h.Service.Delete(ctx, id, authUser.ID); err != nil {
		return respondError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetByID handles GET /api/v1/groups/:id.
func (h *GroupHandler) GetByID(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	grp, err := h.Service.GetByID(ctx, id)
	if err != nil {
		return respondError(c, err)
	}

	children, err := h.Service.ListSubgroups(ctx, id)
	if err != nil {
		return respondError(c, err)
	}

	subs := make([]subgroupSummary, 0, len(children))
	for _, g := range children {
		subs = append(subs, subgroupSummary{
			ID:          g.ID,
			Name:        g.Name,
			Description: g.Description,
			MemberCount: g.MemberCount,
		})
	}

	return c.JSON(http.StatusOK, getGroupResponse{
		ID:          grp.ID,
		Name:        grp.Name,
		Description: grp.Description,
		MemberCount: grp.MemberCount,
		Subgroups:   subs,
	})
}

// ListSubgroups handles GET /api/v1/groups/:id/subgroups.
func (h *GroupHandler) ListSubgroups(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	if _, ok := c.Get("authUser").(domain.User); !ok {
		return respondError(c, domain.ErrUnauthorized)
	}

	children, err := h.Service.ListSubgroups(ctx, id)
	if err != nil {
		return respondError(c, err)
	}

	subs := make([]subgroupSummary, 0, len(children))
	for _, g := range children {
		subs = append(subs, subgroupSummary{
			ID:          g.ID,
			Name:        g.Name,
			Description: g.Description,
			MemberCount: g.MemberCount,
		})
	}

	return c.JSON(http.StatusOK, subgroupListResponse{Subgroups: subs})
}

// ListGroupMembers handles GET /api/v1/groups/:id/members.
func (h *GroupHandler) ListGroupMembers(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	limit, limitErr := parseLimit(c.QueryParam("limit"))
	if limitErr != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	offset, offsetErr := parseOffset(c.QueryParam("offset"))
	if offsetErr != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	q := c.QueryParam("q")

	excludeGroupIDs, excludeErr := parseCommaSeparatedUint64(c.QueryParam("exclude_group_ids"))
	if excludeErr != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	members, total, duplicateCount, err := h.Service.ListGroupMembers(ctx, id, limit, offset, q, excludeGroupIDs)
	if err != nil {
		return respondError(c, err)
	}

	items := make([]groupMember, 0, len(members))
	for _, m := range members {
		item := groupMember{
			ID:           m.ID,
			UUID:         m.UUID,
			FirstName:    m.FirstName,
			LastName:     m.LastName,
			SourceGroups: make([]sourceGroup, len(m.SourceGroups)),
		}
		for i, s := range m.SourceGroups {
			item.SourceGroups[i] = sourceGroup{GroupID: s.GroupID, GroupName: s.GroupName}
		}

		items = append(items, item)
	}

	return c.JSON(http.StatusOK, groupMemberListResponse{Members: items, Total: total, DuplicateCount: duplicateCount})
}

// ListGroups handles GET /api/v1/groups.
func (h *GroupHandler) ListGroups(c echo.Context) error {
	ctx := c.Request().Context()

	limit, limitErr := parseLimit(c.QueryParam("limit"))
	if limitErr != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	offset, offsetErr := parseOffset(c.QueryParam("offset"))
	if offsetErr != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	q := c.QueryParam("q")

	groups, total, err := h.Service.ListGroups(ctx, q, limit, offset)
	if err != nil {
		return respondError(c, err)
	}

	return c.JSON(http.StatusOK, groupListResponse{Groups: groups, Total: total})
}

// ListNonGroupMembers handles GET /api/v1/groups/:id/non-members.
func (h *GroupHandler) ListNonGroupMembers(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	limit, limitErr := parseLimit(c.QueryParam("limit"))
	if limitErr != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	offset, offsetErr := parseOffset(c.QueryParam("offset"))
	if offsetErr != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	q := c.QueryParam("q")

	users, total, err := h.Service.ListNonGroupMembers(ctx, id, limit, offset, q)
	if err != nil {
		return respondError(c, err)
	}

	return c.JSON(http.StatusOK, nonMemberListResponse{Users: users, Total: total})
}

// DeleteGroupMembers handles DELETE /api/v1/groups/:id/members.
func (h *GroupHandler) DeleteGroupMembers(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	var req removeGroupMembersRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return badRequest(c, bindErr.Error())
	}

	if len(req.UserIDs) == 0 {
		return respondError(c, domain.ErrBadParamInput)
	}

	if err := h.Service.RemoveGroupMembers(ctx, id, req.UserIDs); err != nil {
		return respondError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// CreateSubGroup handles POST /api/v1/groups/:id/subgroups.
func (h *GroupHandler) CreateSubGroup(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	if _, ok := c.Get("authUser").(domain.User); !ok {
		return respondError(c, domain.ErrUnauthorized)
	}

	var req createSubGroupRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return badRequest(c, bindErr.Error())
	}

	result, err := h.Service.CreateSubGroup(ctx, id, req.ChildGroupID)
	if err != nil {
		return respondError(c, err)
	}

	return c.JSON(http.StatusCreated, result)
}

// DeleteSubGroup handles DELETE /api/v1/groups/:id/subgroups/:childId.
func (h *GroupHandler) DeleteSubGroup(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	childID, err := parsePathID(c.Param("childId"))
	if err != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	if _, ok := c.Get("authUser").(domain.User); !ok {
		return respondError(c, domain.ErrUnauthorized)
	}

	if err := h.Service.DeleteSubGroup(ctx, id, childID); err != nil {
		return respondError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// AddGroupMembers handles POST /api/v1/groups/:id/members.
func (h *GroupHandler) AddGroupMembers(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return respondError(c, domain.ErrBadParamInput)
	}

	var req addGroupMembersRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return badRequest(c, bindErr.Error())
	}

	if len(req.UserIDs) == 0 {
		return respondError(c, domain.ErrBadParamInput)
	}

	members, err := h.Service.AddGroupMembers(ctx, id, req.UserIDs)
	if err != nil {
		return respondError(c, err)
	}

	return c.JSON(http.StatusCreated, addGroupMembersResponse{Members: members})
}
