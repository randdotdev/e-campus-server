package http

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/classroom"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

type RuleRequest struct {
	Type    string      `json:"type" binding:"required,oneof=single best_of average attendance assignments"`
	Weight  float64     `json:"weight" binding:"required,gt=0"`
	ExamID  *uuid.UUID  `json:"exam_id"`
	ExamIDs []uuid.UUID `json:"exam_ids"`
}

type RulesResponse struct {
	OfferingID uuid.UUID        `json:"offering_id"`
	Rules      []classroom.Rule `json:"rules"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// SaveRules replaces the offering's rule set; an empty list clears it.
func (h *Handler) SaveRules(c *gin.Context) {
	var req struct {
		Rules []RuleRequest `json:"rules" binding:"dive"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	rules := make([]classroom.Rule, len(req.Rules))
	for i, r := range req.Rules {
		rules[i] = classroom.Rule{
			Type: classroom.RuleType(r.Type), Weight: r.Weight,
			ExamID: r.ExamID, ExamIDs: r.ExamIDs,
		}
	}
	gr, err := h.grading.SaveRules(c.Request.Context(), offeringID(c), middleware.GetUserID(c), rules)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, RulesResponse{OfferingID: gr.OfferingID, Rules: rules, UpdatedAt: gr.UpdatedAt})
}

func (h *Handler) GetRules(c *gin.Context) {
	gr, rules, err := h.grading.Rules(c.Request.Context(), offeringID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, RulesResponse{OfferingID: gr.OfferingID, Rules: rules, UpdatedAt: gr.UpdatedAt})
}

// ListGrades shapes by seat: the sheet for teaching staff, the caller's
// own computed preview for students.
func (h *Handler) ListGrades(c *gin.Context) {
	ctx := c.Request.Context()
	if teaching(c) {
		grades, err := h.grading.Grades(ctx, offeringID(c))
		if err != nil {
			h.respondError(c, err)
			return
		}
		type row struct {
			StudentID   uuid.UUID `json:"student_id"`
			StudentName string    `json:"student_name"`
			FinalGrade  *float64  `json:"final_grade"`
			Status      string    `json:"status"`
		}
		result := make([]row, len(grades))
		for i, g := range grades {
			result[i] = row{g.StudentID, g.StudentName, g.FinalGrade, g.Status}
		}
		response.OK(c, result)
		return
	}
	grade, err := h.grading.Preview(ctx, offeringID(c), middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, gin.H{"grade": grade, "preview": true})
}

func (h *Handler) FinalizeGrades(c *gin.Context) {
	count, err := h.grading.Finalize(c.Request.Context(), offeringID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, gin.H{"finalized": count})
}

func (h *Handler) DefinalizeGrades(c *gin.Context) {
	if err := h.grading.Definalize(c.Request.Context(), offeringID(c)); err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, gin.H{"definalized": true})
}

// OverrideGrade writes one student's grade by hand; the ":id" target is
// the student's account ID.
func (h *Handler) OverrideGrade(c *gin.Context) {
	var req struct {
		Grade float64 `json:"grade"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.grading.Override(c.Request.Context(), offeringID(c), targetID(c), req.Grade); err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, gin.H{"grade": req.Grade})
}

// PreviewGrade computes one student's grade without writing it; students
// may only preview themselves.
func (h *Handler) PreviewGrade(c *gin.Context) {
	studentID := targetID(c)
	if !teaching(c) && studentID != middleware.GetUserID(c) {
		response.Forbidden(c, "permission denied")
		return
	}
	grade, err := h.grading.Preview(c.Request.Context(), offeringID(c), studentID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, gin.H{"grade": grade, "preview": true})
}
