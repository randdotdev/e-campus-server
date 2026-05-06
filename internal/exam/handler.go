package exam

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/authz"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
	"github.com/ranjdotdev/e-campus-server/internal/response"
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

// Question handlers

func (h *Handler) CreateQuestion(c *gin.Context) {
	// Question bank is not offering-scoped; scope-based admin check only.
	// Teachers need an offering_id in the request for course-role access (future API change).
	if !authz.Check(c, authz.ResourceExam, authz.ActionCreate) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req CreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)
	question, err := h.service.CreateQuestion(c.Request.Context(), req, userID)
	if err != nil {
		h.log.Error("create question failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.Created(c, ToQuestionResponse(question, true))
}

func (h *Handler) GetQuestion(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid question id")
		return
	}

	// Scope-based check only — question bank has no offering context in the URL.
	if !authz.Check(c, authz.ResourceExam, authz.ActionGet) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	question, err := h.service.GetQuestion(c.Request.Context(), id)
	if errors.Is(err, ErrQuestionNotFound) {
		response.NotFound(c, "question not found")
	} else if err != nil {
		h.log.Error("get question failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToQuestionResponse(question, true))
	}
}

func (h *Handler) ListQuestions(c *gin.Context) {
	// Scope-based check only — question bank has no offering context in the URL.
	if !authz.Check(c, authz.ResourceExam, authz.ActionGet) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	params := pagination.ParsePageParams(c)
	filters := h.parseQuestionFilters(c)

	questions, hasMore, err := h.service.ListQuestions(c.Request.Context(), params, filters)
	if errors.Is(err, pagination.ErrInvalidCursor) {
		response.BadRequest(c, "invalid cursor")
		return
	} else if err != nil {
		h.log.Error("list questions failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[QuestionResponse]{
		Data:    ToQuestionsResponse(questions, true),
		HasMore: hasMore,
	}
	if hasMore && len(questions) > 0 {
		last := questions[len(questions)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UpdateQuestion(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid question id")
		return
	}

	// Scope-based check only — question bank has no offering context in the URL.
	if !authz.Check(c, authz.ResourceExam, authz.ActionUpdate) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req UpdateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	question, err := h.service.UpdateQuestion(c.Request.Context(), id, req)
	if errors.Is(err, ErrQuestionNotFound) {
		response.NotFound(c, "question not found")
	} else if err != nil {
		h.log.Error("update question failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToQuestionResponse(question, true))
	}
}

func (h *Handler) DeleteQuestion(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid question id")
		return
	}

	// Scope-based check only — question bank has no offering context in the URL.
	if !authz.Check(c, authz.ResourceExam, authz.ActionDelete) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	err = h.service.DeleteQuestion(c.Request.Context(), id)
	if errors.Is(err, ErrQuestionNotFound) {
		response.NotFound(c, "question not found")
	} else if err != nil {
		h.log.Error("delete question failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

func (h *Handler) BulkCreateQuestions(c *gin.Context) {
	// Scope-based check only — question bank has no offering context in the URL.
	if !authz.Check(c, authz.ResourceExam, authz.ActionCreate) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req BulkCreateQuestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)
	created, skipped, err := h.service.BulkCreateQuestions(c.Request.Context(), req, userID)
	if err != nil {
		h.log.Error("bulk create questions failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.Created(c, BulkCreateResponse{Created: created, Skipped: skipped})
}

func (h *Handler) RandomSelectQuestions(c *gin.Context) {
	// Scope-based check only — question bank has no offering context in the URL.
	if !authz.Check(c, authz.ResourceExam, authz.ActionGet) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	courseCode := c.Query("course_code")
	if courseCode == "" {
		response.BadRequest(c, "course_code is required")
		return
	}

	dist := DifficultyDistribution{}
	if easy := c.Query("easy"); easy != "" {
		if n, err := strconv.Atoi(easy); err == nil {
			dist.Easy = n
		}
	}
	if medium := c.Query("medium"); medium != "" {
		if n, err := strconv.Atoi(medium); err == nil {
			dist.Medium = n
		}
	}
	if hard := c.Query("hard"); hard != "" {
		if n, err := strconv.Atoi(hard); err == nil {
			dist.Hard = n
		}
	}

	questions, warnings, err := h.service.RandomSelectQuestions(c.Request.Context(), courseCode, dist)
	if err != nil {
		h.log.Error("random select questions failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, RandomSelectResponse{
		Questions: ToQuestionsResponse(questions, true),
		Warnings:  warnings,
	})
}

// Exam handlers

func (h *Handler) CreateExam(c *gin.Context) {
	var req CreateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if !authz.Check(c, authz.ResourceExam, authz.ActionCreate, req.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	userID := middleware.GetUserID(c)

	exam, err := h.service.CreateExam(c.Request.Context(), req, userID)
	if errors.Is(err, ErrOfferingNotFound) {
		response.NotFound(c, "offering not found")
	} else if err != nil {
		h.log.Error("create exam failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, ToExamResponse(exam))
	}
}

func (h *Handler) GetExam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid exam id")
		return
	}

	exam, err := h.service.GetExam(c.Request.Context(), id)
	if errors.Is(err, ErrExamNotFound) {
		response.NotFound(c, "exam not found")
		return
	} else if err != nil {
		h.log.Error("get exam failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !authz.Check(c, authz.ResourceExam, authz.ActionGet, exam.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	response.OK(c, ToExamResponse(exam))
}

func (h *Handler) ListExams(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	if !authz.Check(c, authz.ResourceExam, authz.ActionGet, offeringID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	params := pagination.ParsePageParams(c)
	filters := ExamFilters{
		OfferingID: &offeringID,
	}

	if examType := c.Query("type"); examType != "" {
		filters.Type = &examType
	}
	if mode := c.Query("mode"); mode != "" {
		filters.Mode = &mode
	}
	if status := c.Query("status"); status != "" {
		filters.Status = &status
	}

	exams, hasMore, err := h.service.ListExams(c.Request.Context(), params, filters)
	if errors.Is(err, pagination.ErrInvalidCursor) {
		response.BadRequest(c, "invalid cursor")
		return
	} else if err != nil {
		h.log.Error("list exams failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[ExamListResponse]{
		Data:    ToExamsListResponse(exams),
		HasMore: hasMore,
	}
	if hasMore && len(exams) > 0 {
		last := exams[len(exams)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UpdateExam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid exam id")
		return
	}

	exam, err := h.service.GetExam(c.Request.Context(), id)
	if errors.Is(err, ErrExamNotFound) {
		response.NotFound(c, "exam not found")
		return
	} else if err != nil {
		h.log.Error("get exam failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !authz.Check(c, authz.ResourceExam, authz.ActionUpdate, exam.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req UpdateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	updated, err := h.service.UpdateExam(c.Request.Context(), id, req)
	if errors.Is(err, ErrCannotModifyPublished) {
		response.BadRequest(c, "cannot modify published exam")
	} else if err != nil {
		h.log.Error("update exam failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToExamResponse(updated))
	}
}

func (h *Handler) DeleteExam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid exam id")
		return
	}

	exam, err := h.service.GetExam(c.Request.Context(), id)
	if errors.Is(err, ErrExamNotFound) {
		response.NotFound(c, "exam not found")
		return
	} else if err != nil {
		h.log.Error("get exam failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !authz.Check(c, authz.ResourceExam, authz.ActionDelete, exam.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	err = h.service.DeleteExam(c.Request.Context(), id)
	if errors.Is(err, ErrCannotModifyPublished) {
		response.BadRequest(c, "cannot delete published exam")
	} else if err != nil {
		h.log.Error("delete exam failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

func (h *Handler) PublishExam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid exam id")
		return
	}

	exam, err := h.service.GetExam(c.Request.Context(), id)
	if errors.Is(err, ErrExamNotFound) {
		response.NotFound(c, "exam not found")
		return
	} else if err != nil {
		h.log.Error("get exam failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !authz.Check(c, authz.ResourceExam, authz.ActionUpdate, exam.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	err = h.service.PublishExam(c.Request.Context(), id)
	if errors.Is(err, ErrNoQuestionsInExam) {
		response.BadRequest(c, "exam has no questions")
	} else if err != nil {
		h.log.Error("publish exam failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, gin.H{"status": "published"})
	}
}

func (h *Handler) CloseExam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid exam id")
		return
	}

	exam, err := h.service.GetExam(c.Request.Context(), id)
	if errors.Is(err, ErrExamNotFound) {
		response.NotFound(c, "exam not found")
		return
	} else if err != nil {
		h.log.Error("get exam failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !authz.Check(c, authz.ResourceExam, authz.ActionUpdate, exam.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	err = h.service.CloseExam(c.Request.Context(), id)
	if err != nil {
		h.log.Error("close exam failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, gin.H{"status": "closed"})
	}
}

// Attempt handlers - Student

func (h *Handler) StartAttempt(c *gin.Context) {
	examID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid exam id")
		return
	}

	userID := middleware.GetUserID(c)
	attempt, err := h.service.StartAttempt(c.Request.Context(), examID, userID)
	if errors.Is(err, ErrExamNotFound) {
		response.NotFound(c, "exam not found")
	} else if errors.Is(err, ErrExamNotPublished) {
		response.BadRequest(c, "exam not published")
	} else if errors.Is(err, ErrExamNotAvailable) {
		response.BadRequest(c, "exam not available at this time")
	} else if errors.Is(err, ErrStudentNotFound) {
		response.Forbidden(c, "student record not found")
	} else if errors.Is(err, ErrNotEnrolled) {
		response.Forbidden(c, "not enrolled in this course")
	} else if errors.Is(err, ErrMaxAttemptsReached) {
		response.BadRequest(c, "max attempts reached")
	} else if errors.Is(err, ErrAttemptAlreadySubmitted) {
		response.Conflict(c, "attempt already submitted")
	} else if err != nil {
		h.log.Error("start attempt failed", zap.Error(err))
		response.InternalError(c)
	} else {
		exam, _ := h.service.GetExam(c.Request.Context(), examID)
		response.OK(c, ToAttemptResponse(attempt, exam.AvailableUntil))
	}
}

func (h *Handler) SaveAnswers(c *gin.Context) {
	attemptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid attempt id")
		return
	}

	var req SaveAnswersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	err = h.service.SaveAnswers(c.Request.Context(), attemptID, req.Answers)
	if errors.Is(err, ErrAttemptNotFound) {
		response.NotFound(c, "attempt not found")
	} else if errors.Is(err, ErrAttemptAlreadySubmitted) {
		response.BadRequest(c, "attempt already submitted")
	} else if err != nil {
		h.log.Error("save answers failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, gin.H{"saved": true})
	}
}

func (h *Handler) SubmitAttempt(c *gin.Context) {
	attemptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid attempt id")
		return
	}

	var req SaveAnswersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	attempt, err := h.service.SubmitAttempt(c.Request.Context(), attemptID, req.Answers)
	if errors.Is(err, ErrAttemptNotFound) {
		response.NotFound(c, "attempt not found")
	} else if errors.Is(err, ErrAttemptAlreadySubmitted) {
		response.BadRequest(c, "attempt already submitted")
	} else if err != nil {
		h.log.Error("submit attempt failed", zap.Error(err))
		response.InternalError(c)
	} else {
		exam, _ := h.service.GetExam(c.Request.Context(), attempt.ExamID)
		response.OK(c, ToAttemptResponse(attempt, exam.AvailableUntil))
	}
}

func (h *Handler) GetMyAttempt(c *gin.Context) {
	examID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid exam id")
		return
	}

	userID := middleware.GetUserID(c)
	attempt, err := h.service.GetStudentAttempt(c.Request.Context(), examID, userID)
	if errors.Is(err, ErrStudentNotFound) {
		response.NotFound(c, "student record not found")
	} else if errors.Is(err, ErrAttemptNotFound) {
		response.NotFound(c, "attempt not found")
	} else if err != nil {
		h.log.Error("get my attempt failed", zap.Error(err))
		response.InternalError(c)
	} else {
		exam, _ := h.service.GetExam(c.Request.Context(), examID)
		response.OK(c, ToAttemptResponse(attempt, exam.AvailableUntil))
	}
}

// Attempt handlers - Teacher

func (h *Handler) ListAttempts(c *gin.Context) {
	examID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid exam id")
		return
	}

	exam, err := h.service.GetExam(c.Request.Context(), examID)
	if errors.Is(err, ErrExamNotFound) {
		response.NotFound(c, "exam not found")
		return
	} else if err != nil {
		h.log.Error("get exam failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	// ActionUpdate intentionally used: staff-only listing of all student attempts.
	// ActionGet would allow enrolled students to see each other's attempt records.
	if !authz.Check(c, authz.ResourceExam, authz.ActionUpdate, exam.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	params := pagination.ParsePageParams(c)
	filters := AttemptFilters{
		ExamID: &examID,
		Query:  params.Query,
	}

	if isSubmitted := c.Query("is_submitted"); isSubmitted != "" {
		val := isSubmitted == "true"
		filters.IsSubmitted = &val
	}
	if isGraded := c.Query("is_graded"); isGraded != "" {
		val := isGraded == "true"
		filters.IsGraded = &val
	}

	attempts, hasMore, err := h.service.ListAttempts(c.Request.Context(), params, filters)
	if err != nil {
		h.log.Error("list attempts failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	data := make([]AttemptListResponse, len(attempts))
	for i := range attempts {
		data[i] = ToAttemptListResponse(&attempts[i], exam.AvailableUntil, "")
	}

	result := pagination.PageResult[AttemptListResponse]{
		Data:    data,
		HasMore: hasMore,
	}
	if hasMore && len(attempts) > 0 {
		last := attempts[len(attempts)-1]
		if last.StartedAt != nil {
			result.NextCursor = pagination.EncodeCursor(*last.StartedAt, last.ID)
		}
	}

	response.OK(c, result)
}

func (h *Handler) BulkCreateResults(c *gin.Context) {
	examID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid exam id")
		return
	}

	exam, err := h.service.GetExam(c.Request.Context(), examID)
	if errors.Is(err, ErrExamNotFound) {
		response.NotFound(c, "exam not found")
		return
	} else if err != nil {
		h.log.Error("get exam failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !authz.Check(c, authz.ResourceExam, authz.ActionUpdate, exam.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req BulkResultsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	err = h.service.BulkCreateResults(c.Request.Context(), examID, req)
	if errors.Is(err, ErrInvalidExamMode) {
		response.BadRequest(c, "bulk results only supported for manual mode exams")
	} else if err != nil {
		h.log.Error("bulk create results failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, gin.H{"created": len(req.Results)})
	}
}

func (h *Handler) SetLateDecision(c *gin.Context) {
	attemptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid attempt id")
		return
	}

	attempt, err := h.service.GetAttempt(c.Request.Context(), attemptID)
	if errors.Is(err, ErrAttemptNotFound) {
		response.NotFound(c, "attempt not found")
		return
	} else if err != nil {
		h.log.Error("get attempt failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	exam, err := h.service.GetExam(c.Request.Context(), attempt.ExamID)
	if err != nil {
		h.log.Error("get exam failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !authz.Check(c, authz.ResourceExam, authz.ActionUpdate, exam.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req LateDecisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	err = h.service.SetLateDecision(c.Request.Context(), attemptID, req.Accepted)
	if err != nil {
		h.log.Error("set late decision failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, gin.H{"late_accepted": req.Accepted})
	}
}

func (h *Handler) GradeShortAnswers(c *gin.Context) {
	attemptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid attempt id")
		return
	}

	attempt, err := h.service.GetAttempt(c.Request.Context(), attemptID)
	if errors.Is(err, ErrAttemptNotFound) {
		response.NotFound(c, "attempt not found")
		return
	} else if err != nil {
		h.log.Error("get attempt failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	exam, err := h.service.GetExam(c.Request.Context(), attempt.ExamID)
	if err != nil {
		h.log.Error("get exam failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !authz.Check(c, authz.ResourceExam, authz.ActionCreate, exam.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	userID := middleware.GetUserID(c)

	var req GradeShortAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	updated, err := h.service.GradeShortAnswers(c.Request.Context(), attemptID, req.Scores, userID)
	if err != nil {
		h.log.Error("grade short answers failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToAttemptResponse(updated, exam.AvailableUntil))
	}
}

func (h *Handler) SetVisibility(c *gin.Context) {
	attemptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid attempt id")
		return
	}

	attempt, err := h.service.GetAttempt(c.Request.Context(), attemptID)
	if errors.Is(err, ErrAttemptNotFound) {
		response.NotFound(c, "attempt not found")
		return
	} else if err != nil {
		h.log.Error("get attempt failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	exam, err := h.service.GetExam(c.Request.Context(), attempt.ExamID)
	if err != nil {
		h.log.Error("get exam failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !authz.Check(c, authz.ResourceExam, authz.ActionCreate, exam.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req SetVisibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	err = h.service.SetVisibility(c.Request.Context(), attemptID, req.Visibility, req.VisibleAt)
	if err != nil {
		h.log.Error("set visibility failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, gin.H{"visibility": req.Visibility})
	}
}

// Helper functions

func (h *Handler) parseQuestionFilters(c *gin.Context) QuestionFilters {
	filters := QuestionFilters{
		Query: c.Query("q"),
	}

	if courseCode := c.Query("course_code"); courseCode != "" {
		filters.CourseCode = &courseCode
	}
	if qType := c.Query("type"); qType != "" {
		filters.Type = &qType
	}
	if difficulty := c.Query("difficulty"); difficulty != "" {
		filters.Difficulty = &difficulty
	}
	if isActive := c.Query("is_active"); isActive != "" {
		val := isActive == "true"
		filters.IsActive = &val
	}
	if createdByStr := c.Query("created_by"); createdByStr != "" {
		if id, err := uuid.Parse(createdByStr); err == nil {
			filters.CreatedBy = &id
		}
	}

	return filters
}

// RegisterRoutes registers exam routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// Question routes
	questions := r.Group("/questions")
	questions.Use(authMiddleware)
	{
		questions.GET("", h.ListQuestions)
		questions.POST("", h.CreateQuestion)
		questions.POST("/bulk", h.BulkCreateQuestions)
		questions.GET("/random", h.RandomSelectQuestions)
		questions.GET("/:id", h.GetQuestion)
		questions.PUT("/:id", h.UpdateQuestion)
		questions.DELETE("/:id", h.DeleteQuestion)
	}

	// Exam routes
	exams := r.Group("/exams")
	exams.Use(authMiddleware)
	{
		exams.GET("/:id", h.GetExam)
		exams.PUT("/:id", h.UpdateExam)
		exams.DELETE("/:id", h.DeleteExam)
		exams.POST("/:id/publish", h.PublishExam)
		exams.POST("/:id/close", h.CloseExam)
		exams.POST("/:id/start", h.StartAttempt)
		exams.GET("/:id/attempts", h.ListAttempts)
		exams.POST("/:id/results", h.BulkCreateResults)
		exams.GET("/:id/my-attempt", h.GetMyAttempt)
	}

	// Attempt routes
	attempts := r.Group("/exam-attempts")
	attempts.Use(authMiddleware)
	{
		attempts.PUT("/:id/save", h.SaveAnswers)
		attempts.POST("/:id/submit", h.SubmitAttempt)
		attempts.PUT("/:id/late-decision", h.SetLateDecision)
		attempts.PUT("/:id/grade", h.GradeShortAnswers)
		attempts.PUT("/:id/visibility", h.SetVisibility)
	}

	// Offering nested routes
	offerings := r.Group("/offerings")
	offerings.Use(authMiddleware)
	{
		offerings.GET("/:id/exams", h.ListExams)
		offerings.POST("/:id/exams", h.createExamWithOfferingID)
	}
}

func (h *Handler) createExamWithOfferingID(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	var req CreateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	req.OfferingID = offeringID

	if !authz.Check(c, authz.ResourceExam, authz.ActionCreate, req.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	userID := middleware.GetUserID(c)

	exam, err := h.service.CreateExam(c.Request.Context(), req, userID)
	if errors.Is(err, ErrOfferingNotFound) {
		response.NotFound(c, "offering not found")
	} else if err != nil {
		h.log.Error("create exam failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, ToExamResponse(exam))
	}
}
