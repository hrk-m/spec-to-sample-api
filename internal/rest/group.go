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
	ListGroupMembers(ctx context.Context, id uint64, limit, offset int, q string) ([]domain.User, int, error)
	Store(ctx context.Context, name, description string, userID uint64) (domain.Group, error)
	Update(ctx context.Context, id uint64, name, description string, userID uint64) (*domain.Group, error)
	Delete(ctx context.Context, id uint64, userID uint64) error
	ListNonGroupMembers(ctx context.Context, groupID uint64, limit, offset int, q string) ([]domain.User, int, error)
	AddGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) ([]domain.User, error)
	RemoveGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) error
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
	g.GET("/groups/:id/non-members", h.ListNonGroupMembers)
	g.POST("/groups", h.Store)
	g.POST("/groups/:id/members", h.AddGroupMembers)
	g.PUT("/groups/:id", h.Update)
	g.DELETE("/groups/:id", h.Delete)
	g.DELETE("/groups/:id/members", h.DeleteGroupMembers)
}

type groupListResponse struct {
	Groups []domain.Group `json:"groups"`
	Total  int            `json:"total"`
}

type groupMemberListResponse struct {
	Members []domain.User `json:"members"`
	Total   int           `json:"total"`
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

// Store handles POST /api/v1/groups.
func (h *GroupHandler) Store(c echo.Context) error {
	ctx := c.Request().Context()

	var req storeGroupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	authUser, ok := c.Get("authUser").(domain.User)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ResponseError{Message: "Unauthorized"})
	}

	result, err := h.Service.Store(ctx, req.Name, req.Description, authUser.ID)
	if err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, result)
}

// Update handles PUT /api/v1/groups/:id.
func (h *GroupHandler) Update(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	var req updateGroupRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: bindErr.Error()})
	}

	authUser, ok := c.Get("authUser").(domain.User)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ResponseError{Message: "Unauthorized"})
	}

	result, err := h.Service.Update(ctx, id, req.Name, req.Description, authUser.ID)
	if err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// Delete handles DELETE /api/v1/groups/:id.
func (h *GroupHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	authUser, ok := c.Get("authUser").(domain.User)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ResponseError{Message: "Unauthorized"})
	}

	if err := h.Service.Delete(ctx, id, authUser.ID); err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// GetByID handles GET /api/v1/groups/:id.
func (h *GroupHandler) GetByID(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	result, err := h.Service.GetByID(ctx, id)
	if err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// ListGroupMembers handles GET /api/v1/groups/:id/members.
func (h *GroupHandler) ListGroupMembers(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	limit, limitErr := parseLimit(c.QueryParam("limit"))
	if limitErr != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	offset, offsetErr := parseOffset(c.QueryParam("offset"))
	if offsetErr != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	q := c.QueryParam("q")

	members, total, err := h.Service.ListGroupMembers(ctx, id, limit, offset, q)
	if err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, groupMemberListResponse{Members: members, Total: total})
}

// ListGroups handles GET /api/v1/groups.
func (h *GroupHandler) ListGroups(c echo.Context) error {
	ctx := c.Request().Context()

	limit, limitErr := parseLimit(c.QueryParam("limit"))
	if limitErr != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	offset, offsetErr := parseOffset(c.QueryParam("offset"))
	if offsetErr != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	q := c.QueryParam("q")

	groups, total, err := h.Service.ListGroups(ctx, q, limit, offset)
	if err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, groupListResponse{Groups: groups, Total: total})
}

// ListNonGroupMembers handles GET /api/v1/groups/:id/non-members.
func (h *GroupHandler) ListNonGroupMembers(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	limit, limitErr := parseLimit(c.QueryParam("limit"))
	if limitErr != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	offset, offsetErr := parseOffset(c.QueryParam("offset"))
	if offsetErr != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	q := c.QueryParam("q")

	users, total, err := h.Service.ListNonGroupMembers(ctx, id, limit, offset, q)
	if err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, nonMemberListResponse{Users: users, Total: total})
}

// DeleteGroupMembers handles DELETE /api/v1/groups/:id/members.
func (h *GroupHandler) DeleteGroupMembers(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	var req removeGroupMembersRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: bindErr.Error()})
	}

	if len(req.UserIDs) == 0 {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	if err := h.Service.RemoveGroupMembers(ctx, id, req.UserIDs); err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// AddGroupMembers handles POST /api/v1/groups/:id/members.
func (h *GroupHandler) AddGroupMembers(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parsePathID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	var req addGroupMembersRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: bindErr.Error()})
	}

	if len(req.UserIDs) == 0 {
		return c.JSON(http.StatusBadRequest, ResponseError{Message: domain.ErrBadParamInput.Error()})
	}

	members, err := h.Service.AddGroupMembers(ctx, id, req.UserIDs)
	if err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, addGroupMembersResponse{Members: members})
}
