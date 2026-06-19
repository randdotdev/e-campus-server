package team

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/randdotdev/e-campus-server/internal/middleware"
	"github.com/randdotdev/e-campus-server/internal/response"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	log     *zap.Logger
}

func NewHandler(service *Service, log *zap.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log,
	}
}

func (h *Handler) CreateTeam(c *gin.Context) {
	var req CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)

	team, err := h.service.Create(c.Request.Context(), userID, req.Name)
	if err != nil {
		h.log.Error("create team failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	teamWithMembers, err := h.service.GetByID(c.Request.Context(), team.ID)
	if err != nil {
		h.log.Error("get team failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.Created(c, ToTeamResponse(teamWithMembers))
}

func (h *Handler) GetTeam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid team id")
		return
	}

	userID := middleware.GetUserID(c)

	team, err := h.service.GetByIDForUser(c.Request.Context(), id, userID)
	if errors.Is(err, ErrTeamNotFound) {
		response.NotFound(c, "team not found")
		return
	}
	if err != nil {
		h.log.Error("get team failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToTeamResponse(team))
}

func (h *Handler) GetMyTeams(c *gin.Context) {
	userID := middleware.GetUserID(c)
	status := c.Query("status")

	var statusPtr *string
	if status != "" {
		if !IsValidStatus(status) {
			response.BadRequest(c, "invalid status")
			return
		}
		statusPtr = &status
	}

	teams, err := h.service.GetMyTeams(c.Request.Context(), userID, statusPtr)
	if err != nil {
		h.log.Error("get my teams failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToMyTeamsResponse(teams))
}

func (h *Handler) UpdateTeam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid team id")
		return
	}

	var req UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)

	_, err = h.service.UpdateName(c.Request.Context(), id, userID, req.Name)
	if errors.Is(err, ErrTeamNotFound) {
		response.NotFound(c, "team not found")
		return
	}
	if errors.Is(err, ErrNotLeader) {
		response.Forbidden(c, "only leader can update team")
		return
	}
	if err != nil {
		h.log.Error("update team failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	team, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		h.log.Error("get team failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToTeamResponse(team))
}

func (h *Handler) DeleteTeam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid team id")
		return
	}

	userID := middleware.GetUserID(c)

	err = h.service.Delete(c.Request.Context(), id, userID)
	if errors.Is(err, ErrTeamNotFound) {
		response.NotFound(c, "team not found")
	} else if errors.Is(err, ErrNotLeader) {
		response.Forbidden(c, "only leader can delete team")
	} else if errors.Is(err, ErrTeamLocked) {
		response.Conflict(c, "team has submissions and cannot be deleted")
	} else if err != nil {
		h.log.Error("delete team failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

func (h *Handler) AddMember(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid team id")
		return
	}

	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)

	err = h.service.AddMember(c.Request.Context(), id, userID, req.StudentID)
	if errors.Is(err, ErrTeamNotFound) {
		response.NotFound(c, "team not found")
	} else if errors.Is(err, ErrNotLeader) {
		response.Forbidden(c, "only leader can add members")
	} else if errors.Is(err, ErrTeamArchived) {
		response.Conflict(c, "team is archived")
	} else if errors.Is(err, ErrTeamLocked) {
		response.Conflict(c, "team has submissions and cannot be modified")
	} else if errors.Is(err, ErrAlreadyMember) {
		response.Conflict(c, "student is already a member")
	} else if errors.Is(err, ErrMaxMembers) {
		response.Conflict(c, "team has maximum members")
	} else if err != nil {
		h.log.Error("add member failed", zap.Error(err))
		response.InternalError(c)
	} else {
		team, _ := h.service.GetByID(c.Request.Context(), id)
		response.OK(c, ToTeamResponse(team))
	}
}

func (h *Handler) RemoveMember(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid team id")
		return
	}

	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		response.BadRequest(c, "invalid student id")
		return
	}

	userID := middleware.GetUserID(c)

	err = h.service.RemoveMember(c.Request.Context(), id, userID, studentID)
	if errors.Is(err, ErrTeamNotFound) {
		response.NotFound(c, "team not found")
	} else if errors.Is(err, ErrNotLeader) {
		response.Forbidden(c, "only leader can remove members")
	} else if errors.Is(err, ErrCannotRemoveLeader) {
		response.Conflict(c, "cannot remove leader")
	} else if errors.Is(err, ErrTeamLocked) {
		response.Conflict(c, "team has submissions and cannot be modified")
	} else if errors.Is(err, ErrMemberNotFound) {
		response.NotFound(c, "member not found")
	} else if err != nil {
		h.log.Error("remove member failed", zap.Error(err))
		response.InternalError(c)
	} else {
		team, _ := h.service.GetByID(c.Request.Context(), id)
		response.OK(c, ToTeamResponse(team))
	}
}

func (h *Handler) LeaveTeam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid team id")
		return
	}

	userID := middleware.GetUserID(c)

	err = h.service.Leave(c.Request.Context(), id, userID)
	if errors.Is(err, ErrTeamNotFound) {
		response.NotFound(c, "team not found")
	} else if errors.Is(err, ErrLeaderCannotLeave) {
		response.Conflict(c, "leader must transfer leadership first")
	} else if errors.Is(err, ErrTeamLocked) {
		response.Conflict(c, "team has submissions and cannot be modified")
	} else if errors.Is(err, ErrNotMember) {
		response.NotFound(c, "not a member of this team")
	} else if err != nil {
		h.log.Error("leave team failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

func (h *Handler) TransferLeadership(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid team id")
		return
	}

	var req TransferLeadershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)

	_, err = h.service.TransferLeadership(c.Request.Context(), id, userID, req.NewLeaderID)
	if errors.Is(err, ErrTeamNotFound) {
		response.NotFound(c, "team not found")
		return
	}
	if errors.Is(err, ErrNotLeader) {
		response.Forbidden(c, "only leader can transfer leadership")
		return
	}
	if errors.Is(err, ErrNotMember) {
		response.BadRequest(c, "new leader must be a team member")
		return
	}
	if err != nil {
		h.log.Error("transfer leadership failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	team, _ := h.service.GetByID(c.Request.Context(), id)
	response.OK(c, ToTeamResponse(team))
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	teams := r.Group("/teams")
	teams.Use(authMiddleware)
	{
		teams.POST("", h.CreateTeam)
		teams.GET("/me", h.GetMyTeams)
		teams.GET("/:id", h.GetTeam)
		teams.PUT("/:id", h.UpdateTeam)
		teams.DELETE("/:id", h.DeleteTeam)
		teams.POST("/:id/members", h.AddMember)
		teams.DELETE("/:id/members/:student_id", h.RemoveMember)
		teams.POST("/:id/leave", h.LeaveTeam)
		teams.POST("/:id/transfer-leadership", h.TransferLeadership)
	}
}
