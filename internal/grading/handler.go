package grading

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/middleware"
	"github.com/randdotdev/e-campus-server/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) SaveRules(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering_id")
		return
	}

	if !authz.Check(c, authz.ResourceGrade, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	userID := middleware.GetUserID(c)

	var req SaveRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	rules := ToRules(req.Rules)
	gr, err := h.service.SaveRules(c.Request.Context(), offeringID, rules, userID)
	if err != nil {
		switch err {
		case ErrOfferingNotFound:
			response.NotFound(c, "offering not found")
		case ErrSemesterArchived:
			response.BadRequest(c, "cannot modify rules for archived semester")
		case ErrInvalidRuleType:
			response.BadRequest(c, "invalid rule type")
		case ErrWeightsMustSum100:
			response.BadRequest(c, "rule weights must sum to 100")
		case ErrExamNotFound:
			response.BadRequest(c, "one or more exams not found in this offering")
		default:
			response.InternalError(c)
		}
		return
	}

	response.OK(c, ToRulesResponse(gr, rules))
}

func (h *Handler) GetRules(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering_id")
		return
	}

	if !authz.Check(c, authz.ResourceGrade, authz.ActionGet, offeringID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	gr, rules, err := h.service.GetRules(c.Request.Context(), offeringID)
	if err != nil {
		if err == ErrRulesNotFound {
			response.NotFound(c, "grading rules not found")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, ToRulesResponse(gr, rules))
}

func (h *Handler) DeleteRules(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering_id")
		return
	}

	if !authz.Check(c, authz.ResourceGrade, authz.ActionDelete, offeringID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	if err := h.service.DeleteRules(c.Request.Context(), offeringID); err != nil {
		if err == ErrSemesterArchived {
			response.BadRequest(c, "cannot delete rules for archived semester")
			return
		}
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

func (h *Handler) FinalizeGrades(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering_id")
		return
	}

	if !authz.Check(c, authz.ResourceGrade, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	count, err := h.service.FinalizeGrades(c.Request.Context(), offeringID)
	if err != nil {
		switch err {
		case ErrSemesterNotGrading:
			response.BadRequest(c, "semester must be in grading status to finalize")
		case ErrAlreadyFinalized:
			response.BadRequest(c, "grades already finalized")
		case ErrRulesNotFound:
			response.BadRequest(c, "grading rules must be set before finalizing")
		case ErrNoEnrollments:
			response.BadRequest(c, "no enrolled students to grade")
		case ErrUngradedExams:
			response.BadRequest(c, "cannot finalize: some exams have ungraded submissions")
		case ErrUngradedAssignments:
			response.BadRequest(c, "cannot finalize: some assignments have ungraded submissions")
		default:
			response.InternalError(c)
		}
		return
	}

	response.OK(c, FinalizeResponse{Finalized: count})
}

func (h *Handler) DefinalizeGrades(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering_id")
		return
	}

	if !authz.Check(c, authz.ResourceGrade, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	if err := h.service.DefinalizeGrades(c.Request.Context(), offeringID); err != nil {
		switch err {
		case ErrSemesterArchived:
			response.BadRequest(c, "cannot definalize archived semester")
		case ErrNotFinalized:
			response.BadRequest(c, "grades are not finalized")
		default:
			response.InternalError(c)
		}
		return
	}

	response.NoContent(c)
}

func (h *Handler) GetGrades(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering_id")
		return
	}

	if !authz.Check(c, authz.ResourceGrade, authz.ActionGet, offeringID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	grades, err := h.service.GetGrades(c.Request.Context(), offeringID)
	if err != nil {
		if err == ErrOfferingNotFound {
			response.NotFound(c, "offering not found")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, ToStudentGradesResponse(grades))
}

func (h *Handler) OverrideGrade(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering_id")
		return
	}

	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		response.BadRequest(c, "invalid student_id")
		return
	}

	if !authz.Check(c, authz.ResourceGrade, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req OverrideGradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if err := h.service.OverrideGrade(c.Request.Context(), offeringID, studentID, req.Grade); err != nil {
		switch err {
		case ErrSemesterArchived:
			response.BadRequest(c, "cannot override grades for archived semester")
		case ErrInvalidGrade:
			response.BadRequest(c, "grade must be between 0 and 100")
		case ErrStudentNotEnrolled:
			response.NotFound(c, "student not enrolled in this offering")
		default:
			response.InternalError(c)
		}
		return
	}

	response.NoContent(c)
}

func (h *Handler) PreviewGrade(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering_id")
		return
	}

	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		response.BadRequest(c, "invalid student_id")
		return
	}

	if !authz.Check(c, authz.ResourceGrade, authz.ActionGet, offeringID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	grade, err := h.service.PreviewGrade(c.Request.Context(), offeringID, studentID)
	if err != nil {
		if err == ErrRulesNotFound {
			response.NotFound(c, "grading rules not set")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, PreviewResponse{Grade: grade})
}

func (h *Handler) GetMyGrade(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	userID := middleware.GetUserID(c)

	grade, err := h.service.PreviewGrade(c.Request.Context(), offeringID, userID)
	if err != nil {
		if err == ErrRulesNotFound {
			response.NotFound(c, "grading rules not set")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, PreviewResponse{Grade: grade})
}
