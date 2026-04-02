package university

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
	"github.com/ranjdotdev/e-campus-server/internal/permission"
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

// College handlers

func (h *Handler) CreateCollege(c *gin.Context) {
	if !permission.CanAdminUniversity(c) {
		response.Forbidden(c, "university admin access required")
		return
	}

	var req CreateCollegeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	college, err := h.service.CreateCollege(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCollegeLimitReached) {
			response.Forbidden(c, "college limit reached for current subscription")
			return
		}
		if errors.Is(err, ErrCodeExists) {
			response.Conflict(c, "college code already exists")
			return
		}
		h.log.Error("create college failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.Created(c, ToCollegeResponse(college))
}

func (h *Handler) GetCollege(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid college id")
		return
	}

	college, err := h.service.GetCollege(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrCollegeNotFound) {
			response.NotFound(c, "college not found")
			return
		}
		h.log.Error("get college failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToCollegeResponse(college))
}

// ListColleges returns all colleges. This endpoint is intentionally accessible
// to all authenticated users as university structure is public directory information.
func (h *Handler) ListColleges(c *gin.Context) {
	params := pagination.ParsePageParams(c)
	filters, err := h.parseCollegeFilters(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	colleges, hasMore, err := h.service.ListColleges(c.Request.Context(), params, filters)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list colleges failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[CollegeResponse]{
		Data:    ToCollegesResponse(colleges),
		HasMore: hasMore,
	}
	if hasMore && len(colleges) > 0 {
		last := colleges[len(colleges)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UpdateCollege(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid college id")
		return
	}

	if !permission.CanAdminUniversity(c) {
		response.Forbidden(c, "university admin access required")
		return
	}

	var req UpdateCollegeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	college, err := h.service.UpdateCollege(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrCollegeNotFound) {
			response.NotFound(c, "college not found")
			return
		}
		if errors.Is(err, ErrCodeExists) {
			response.Conflict(c, "college code already exists")
			return
		}
		h.log.Error("update college failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToCollegeResponse(college))
}

// Department handlers

func (h *Handler) CreateDepartment(c *gin.Context) {
	var req CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if !permission.CanAdminCollege(c, req.CollegeID) {
		response.Forbidden(c, "college admin access required")
		return
	}

	dept, err := h.service.CreateDepartment(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCollegeNotFound) {
			response.NotFound(c, "college not found")
			return
		}
		if errors.Is(err, ErrDepartmentLimitReached) {
			response.Forbidden(c, "department limit reached for this college")
			return
		}
		if errors.Is(err, ErrCodeExists) {
			response.Conflict(c, "department code already exists in this college")
			return
		}
		h.log.Error("create department failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.Created(c, ToDepartmentResponse(dept))
}

func (h *Handler) GetDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid department id")
		return
	}

	dept, err := h.service.GetDepartment(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrDepartmentNotFound) {
			response.NotFound(c, "department not found")
			return
		}
		h.log.Error("get department failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToDepartmentResponse(dept))
}

// ListDepartments returns departments. This endpoint is intentionally accessible
// to all authenticated users as university structure is public directory information.
func (h *Handler) ListDepartments(c *gin.Context) {
	params := pagination.ParsePageParams(c)
	filters, err := h.parseDepartmentFilters(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	depts, hasMore, err := h.service.ListDepartments(c.Request.Context(), params, filters)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		if errors.Is(err, ErrCollegeNotFound) {
			response.NotFound(c, "college not found")
			return
		}
		h.log.Error("list departments failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[DepartmentResponse]{
		Data:    ToDepartmentsResponse(depts),
		HasMore: hasMore,
	}
	if hasMore && len(depts) > 0 {
		last := depts[len(depts)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UpdateDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid department id")
		return
	}

	dept, err := h.service.GetDepartment(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrDepartmentNotFound) {
			response.NotFound(c, "department not found")
			return
		}
		h.log.Error("get department failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !permission.CanAdminCollege(c, dept.CollegeID) {
		response.Forbidden(c, "college admin access required")
		return
	}

	var req UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updated, err := h.service.UpdateDepartment(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrCodeExists) {
			response.Conflict(c, "department code already exists in this college")
			return
		}
		h.log.Error("update department failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToDepartmentResponse(updated))
}

// Program handlers

func (h *Handler) CreateProgram(c *gin.Context) {
	var req CreateProgramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Check department-level permission, or college-level permission for the department's college
	if !permission.CanAdminDepartment(c, req.DepartmentID) {
		dept, err := h.service.GetDepartment(c.Request.Context(), req.DepartmentID)
		if err != nil {
			if errors.Is(err, ErrDepartmentNotFound) {
				response.NotFound(c, "department not found")
				return
			}
			h.log.Error("get department failed", zap.Error(err))
			response.InternalError(c)
			return
		}
		if !permission.CanAdminCollege(c, dept.CollegeID) {
			response.Forbidden(c, "department admin access required")
			return
		}
	}

	program, err := h.service.CreateProgram(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrDepartmentNotFound) {
			response.NotFound(c, "department not found")
			return
		}
		if errors.Is(err, ErrProgramLimitReached) {
			response.Forbidden(c, "program limit reached for this department")
			return
		}
		if errors.Is(err, ErrCodeExists) {
			response.Conflict(c, "program code already exists in this department")
			return
		}
		h.log.Error("create program failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.Created(c, ToProgramResponse(program))
}

func (h *Handler) GetProgram(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program id")
		return
	}

	program, err := h.service.GetProgram(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrProgramNotFound) {
			response.NotFound(c, "program not found")
			return
		}
		h.log.Error("get program failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToProgramResponse(program))
}

// ListPrograms returns programs. This endpoint is intentionally accessible
// to all authenticated users as university structure is public directory information.
func (h *Handler) ListPrograms(c *gin.Context) {
	params := pagination.ParsePageParams(c)
	filters, err := h.parseProgramFilters(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	programs, hasMore, err := h.service.ListPrograms(c.Request.Context(), params, filters)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		if errors.Is(err, ErrDepartmentNotFound) {
			response.NotFound(c, "department not found")
			return
		}
		h.log.Error("list programs failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[ProgramResponse]{
		Data:    ToProgramsResponse(programs),
		HasMore: hasMore,
	}
	if hasMore && len(programs) > 0 {
		last := programs[len(programs)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UpdateProgram(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program id")
		return
	}

	program, err := h.service.GetProgram(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrProgramNotFound) {
			response.NotFound(c, "program not found")
			return
		}
		h.log.Error("get program failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	// Check department-level permission, or college-level permission for the program's college
	if !permission.CanAdminDepartment(c, program.DepartmentID) {
		dept, err := h.service.GetDepartment(c.Request.Context(), program.DepartmentID)
		if err != nil {
			h.log.Error("get department failed", zap.Error(err))
			response.InternalError(c)
			return
		}
		if !permission.CanAdminCollege(c, dept.CollegeID) {
			response.Forbidden(c, "department admin access required")
			return
		}
	}

	var req UpdateProgramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updated, err := h.service.UpdateProgram(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrCodeExists) {
			response.Conflict(c, "program code already exists in this department")
			return
		}
		h.log.Error("update program failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToProgramResponse(updated))
}

func (h *Handler) parseCollegeFilters(c *gin.Context) (CollegeFilters, error) {
	return CollegeFilters{
		IsActive: pagination.ParseBool(c, "is_active"),
	}, nil
}

func (h *Handler) parseDepartmentFilters(c *gin.Context) (DepartmentFilters, error) {
	filters := DepartmentFilters{
		IsActive: pagination.ParseBool(c, "is_active"),
	}
	// Check path param first, then query param
	collegeIDStr := c.Param("id")
	if collegeIDStr == "" {
		collegeIDStr = c.Query("college_id")
	}
	if collegeIDStr != "" {
		id, err := uuid.Parse(collegeIDStr)
		if err != nil {
			return filters, errors.New("invalid college_id")
		}
		filters.CollegeID = &id
	}
	return filters, nil
}

func (h *Handler) parseProgramFilters(c *gin.Context) (ProgramFilters, error) {
	filters := ProgramFilters{
		IsActive: pagination.ParseBool(c, "is_active"),
	}
	// Check path param first, then query param
	deptIDStr := c.Param("id")
	if deptIDStr == "" {
		deptIDStr = c.Query("department_id")
	}
	if deptIDStr != "" {
		id, err := uuid.Parse(deptIDStr)
		if err != nil {
			return filters, errors.New("invalid department_id")
		}
		filters.DepartmentID = &id
	}
	if degreeType := c.Query("degree_type"); degreeType != "" {
		filters.DegreeType = &degreeType
	}
	return filters, nil
}

func (h *Handler) GetPublicColleges(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")
	params := pagination.ParsePageParams(c)
	filters := CollegeFilters{IsActive: ptrBool(true)}

	colleges, hasMore, err := h.service.ListColleges(c.Request.Context(), params, filters)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list public colleges failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[CollegePublicResponse]{
		Data:    ToCollegesPublicResponse(colleges, lang),
		HasMore: hasMore,
	}
	if hasMore && len(colleges) > 0 {
		last := colleges[len(colleges)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) GetPublicCollege(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid college id")
		return
	}

	college, err := h.service.GetCollege(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrCollegeNotFound) {
			response.NotFound(c, "college not found")
			return
		}
		h.log.Error("get public college failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !college.IsActive {
		response.NotFound(c, "college not found")
		return
	}

	response.OK(c, ToCollegePublicResponse(college, lang))
}

func (h *Handler) GetPublicDepartments(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")
	collegeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid college id")
		return
	}

	params := pagination.ParsePageParams(c)
	filters := DepartmentFilters{CollegeID: &collegeID, IsActive: ptrBool(true)}

	depts, hasMore, err := h.service.ListDepartments(c.Request.Context(), params, filters)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list public departments failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[DepartmentPublicResponse]{
		Data:    ToDepartmentsPublicResponse(depts, lang),
		HasMore: hasMore,
	}
	if hasMore && len(depts) > 0 {
		last := depts[len(depts)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) GetPublicDepartment(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid department id")
		return
	}

	dept, err := h.service.GetDepartment(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrDepartmentNotFound) {
			response.NotFound(c, "department not found")
			return
		}
		h.log.Error("get public department failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !dept.IsActive {
		response.NotFound(c, "department not found")
		return
	}

	response.OK(c, ToDepartmentPublicResponse(dept, lang))
}

func (h *Handler) GetPublicPrograms(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")
	deptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid department id")
		return
	}

	params := pagination.ParsePageParams(c)
	filters := ProgramFilters{DepartmentID: &deptID, IsActive: ptrBool(true)}

	programs, hasMore, err := h.service.ListPrograms(c.Request.Context(), params, filters)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list public programs failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[ProgramPublicResponse]{
		Data:    ToProgramsPublicResponse(programs, lang),
		HasMore: hasMore,
	}
	if hasMore && len(programs) > 0 {
		last := programs[len(programs)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func ptrBool(b bool) *bool {
	return &b
}
