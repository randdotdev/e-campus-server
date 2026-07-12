package http

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/classroom"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

type TeamResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      *string   `json:"name"`
	LeaderID  uuid.UUID `json:"leader_id"`
	Status    string    `json:"status"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

func teamResponse(t *classroom.Team) TeamResponse {
	return TeamResponse{
		ID: t.ID, Name: t.Name, LeaderID: t.LeaderID,
		Status: string(t.Status), Version: t.Version, CreatedAt: t.CreatedAt,
	}
}

type TeamMemberResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Name     string    `json:"name"`
	Username string    `json:"username"`
	Avatar   *string   `json:"avatar"`
	JoinedAt time.Time `json:"joined_at"`
}

// teamID parses the ungated team routes' ":id" param; teams sit outside
// the offering gate, so the handler does its own parsing.
func teamID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.NotFound(c, "team not found")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) CreateTeam(c *gin.Context) {
	var req struct {
		Name *string `json:"name" binding:"omitempty,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	team, err := h.teams.Create(c.Request.Context(), middleware.GetUserID(c), req.Name)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, teamResponse(team))
}

func (h *Handler) GetTeam(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	team, err := h.teams.Get(c.Request.Context(), id, middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	members := make([]TeamMemberResponse, len(team.Members))
	for i, m := range team.Members {
		members[i] = TeamMemberResponse{
			UserID: m.UserID, Name: m.Name, Username: m.Username,
			Avatar: m.Avatar, JoinedAt: m.JoinedAt,
		}
	}
	response.OK(c, gin.H{"team": teamResponse(&team.Team), "members": members})
}

func (h *Handler) MyTeams(c *gin.Context) {
	var status *classroom.TeamStatus
	if v := c.Query("status"); v != "" {
		s := classroom.TeamStatus(v)
		status = &s
	}
	teams, err := h.teams.MyTeams(c.Request.Context(), middleware.GetUserID(c), status)
	if err != nil {
		h.respondError(c, err)
		return
	}
	type row struct {
		TeamResponse
		MemberCount int  `json:"member_count"`
		IsLeader    bool `json:"is_leader"`
	}
	result := make([]row, len(teams))
	for i, t := range teams {
		result[i] = row{teamResponse(&t.Team), t.MemberCount, t.IsLeader}
	}
	response.OK(c, result)
}

func (h *Handler) RenameTeam(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	var req struct {
		Name string `json:"name" binding:"required,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	team, err := h.teams.Rename(c.Request.Context(), id, middleware.GetUserID(c), req.Name)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, teamResponse(team))
}

func (h *Handler) DeleteTeam(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	if err := h.teams.Delete(c.Request.Context(), id, middleware.GetUserID(c)); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *Handler) AddTeamMember(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	var req struct {
		UserID uuid.UUID `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.teams.AddMember(c.Request.Context(), id, middleware.GetUserID(c), req.UserID); err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, gin.H{"added": req.UserID})
}

func (h *Handler) RemoveTeamMember(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.NotFound(c, "member not found")
		return
	}
	if err := h.teams.RemoveMember(c.Request.Context(), id, middleware.GetUserID(c), userID); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *Handler) LeaveTeam(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	if err := h.teams.Leave(c.Request.Context(), id, middleware.GetUserID(c)); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *Handler) TransferTeamLeadership(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	var req struct {
		NewLeaderID uuid.UUID `json:"new_leader_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	team, err := h.teams.TransferLeadership(c.Request.Context(), id, middleware.GetUserID(c), req.NewLeaderID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, teamResponse(team))
}
