package course

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
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

// Course handlers

func (h *Handler) CreateCourse(c *gin.Context) {
	var req CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if !permission.CanAdminDepartment(c, req.DepartmentID) {
		response.Forbidden(c, "department admin access required")
		return
	}

	course, err := h.service.CreateCourse(c.Request.Context(), req)
	if errors.Is(err, ErrDuplicateCode) {
		response.Conflict(c, "course code already exists")
	} else if err != nil {
		h.log.Error("create course failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, ToCourseResponse(course))
	}
}

func (h *Handler) GetCourse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	course, err := h.service.GetCourse(c.Request.Context(), id)
	if errors.Is(err, ErrCourseNotFound) {
		response.NotFound(c, "course not found")
	} else if err != nil {
		h.log.Error("get course failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToCourseResponse(course))
	}
}

func (h *Handler) ListCourses(c *gin.Context) {
	params := pagination.ParsePageParams(c)
	filters, err := h.parseCourseFilters(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	courses, hasMore, err := h.service.ListCourses(c.Request.Context(), params, filters)
	if errors.Is(err, pagination.ErrInvalidCursor) {
		response.BadRequest(c, "invalid cursor")
		return
	} else if err != nil {
		h.log.Error("list courses failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[CourseResponse]{
		Data:    ToCoursesResponse(courses),
		HasMore: hasMore,
	}
	if hasMore && len(courses) > 0 {
		last := courses[len(courses)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UpdateCourse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	course, err := h.service.GetCourse(c.Request.Context(), id)
	if errors.Is(err, ErrCourseNotFound) {
		response.NotFound(c, "course not found")
		return
	} else if err != nil {
		h.log.Error("get course failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !permission.CanAdminDepartment(c, course.DepartmentID) {
		response.Forbidden(c, "department admin access required")
		return
	}

	var req UpdateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updated, err := h.service.UpdateCourse(c.Request.Context(), id, req)
	if err != nil {
		h.log.Error("update course failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToCourseResponse(updated))
}

func (h *Handler) GetSiblingCourses(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	siblings, err := h.service.GetSiblingCourses(c.Request.Context(), id)
	if errors.Is(err, ErrCourseNotFound) {
		response.NotFound(c, "course not found")
	} else if err != nil {
		h.log.Error("get siblings failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToCoursesResponse(siblings))
	}
}

// Offering handlers

func (h *Handler) CreateOffering(c *gin.Context) {
	var req CreateOfferingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	course, err := h.service.GetCourse(c.Request.Context(), req.CourseID)
	if errors.Is(err, ErrCourseNotFound) {
		response.NotFound(c, "course not found")
		return
	} else if err != nil {
		h.log.Error("get course failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !permission.CanAdminDepartment(c, course.DepartmentID) {
		response.Forbidden(c, "department admin access required")
		return
	}

	offering, err := h.service.CreateOffering(c.Request.Context(), req)
	if errors.Is(err, ErrSemesterNotFound) {
		response.NotFound(c, "semester not found")
	} else if errors.Is(err, ErrDuplicateOffering) {
		response.Conflict(c, "offering already exists for this course, semester, and study type")
	} else if err != nil {
		h.log.Error("create offering failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, ToOfferingResponse(offering))
	}
}

func (h *Handler) GetOffering(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	offering, err := h.service.GetOffering(c.Request.Context(), id)
	if errors.Is(err, ErrOfferingNotFound) {
		response.NotFound(c, "offering not found")
	} else if err != nil {
		h.log.Error("get offering failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToOfferingResponse(offering))
	}
}

func (h *Handler) ListOfferings(c *gin.Context) {
	params := pagination.ParsePageParams(c)
	filters, err := h.parseOfferingFilters(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	offerings, hasMore, err := h.service.ListOfferings(c.Request.Context(), params, filters)
	if errors.Is(err, pagination.ErrInvalidCursor) {
		response.BadRequest(c, "invalid cursor")
		return
	} else if err != nil {
		h.log.Error("list offerings failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[OfferingResponse]{
		Data:    ToOfferingsResponse(offerings),
		HasMore: hasMore,
	}
	if hasMore && len(offerings) > 0 {
		last := offerings[len(offerings)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UpdateOffering(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	offering, err := h.service.GetOffering(c.Request.Context(), id)
	if errors.Is(err, ErrOfferingNotFound) {
		response.NotFound(c, "offering not found")
		return
	} else if err != nil {
		h.log.Error("get offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	course, err := h.service.GetCourse(c.Request.Context(), offering.CourseID)
	if err != nil {
		h.log.Error("get course failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !permission.CanAdminDepartment(c, course.DepartmentID) {
		response.Forbidden(c, "department admin access required")
		return
	}

	var req UpdateOfferingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updated, err := h.service.UpdateOffering(c.Request.Context(), id, req)
	if err != nil {
		h.log.Error("update offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToOfferingResponse(updated))
}

// Teacher handlers

func (h *Handler) AddTeacher(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offering_id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	offering, err := h.service.GetOffering(c.Request.Context(), offeringID)
	if errors.Is(err, ErrOfferingNotFound) {
		response.NotFound(c, "offering not found")
		return
	} else if err != nil {
		h.log.Error("get offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	course, err := h.service.GetCourse(c.Request.Context(), offering.CourseID)
	if err != nil {
		h.log.Error("get course failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !permission.CanAdminDepartment(c, course.DepartmentID) {
		response.Forbidden(c, "department admin access required")
		return
	}

	var req AddTeacherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	teacher, err := h.service.AddTeacher(c.Request.Context(), offeringID, req)
	if errors.Is(err, ErrAlreadyTeacher) {
		response.Conflict(c, "user is already a teacher")
	} else if err != nil {
		h.log.Error("add teacher failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, ToTeacherResponse(teacher))
	}
}

func (h *Handler) ListTeachers(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offering_id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	teachers, err := h.service.ListTeachers(c.Request.Context(), offeringID)
	if errors.Is(err, ErrOfferingNotFound) {
		response.NotFound(c, "offering not found")
	} else if err != nil {
		h.log.Error("list teachers failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToTeachersResponse(teachers))
	}
}

func (h *Handler) RemoveTeacher(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offering_id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	offering, err := h.service.GetOffering(c.Request.Context(), offeringID)
	if errors.Is(err, ErrOfferingNotFound) {
		response.NotFound(c, "offering not found")
		return
	} else if err != nil {
		h.log.Error("get offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	course, err := h.service.GetCourse(c.Request.Context(), offering.CourseID)
	if err != nil {
		h.log.Error("get course failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !permission.CanAdminDepartment(c, course.DepartmentID) {
		response.Forbidden(c, "department admin access required")
		return
	}

	err = h.service.RemoveTeacher(c.Request.Context(), offeringID, userID)
	if errors.Is(err, ErrTeacherNotFound) {
		response.NotFound(c, "teacher not found")
	} else if err != nil {
		h.log.Error("remove teacher failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

// Section handlers

func (h *Handler) CreateSection(c *gin.Context) {
	var req CreateSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	teacher, err := h.service.GetTeacherRole(c.Request.Context(), req.OfferingID, userID)
	if errors.Is(err, ErrTeacherNotFound) {
		response.Forbidden(c, "not a teacher of this course")
		return
	} else if err != nil {
		h.log.Error("get teacher role failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !CanTeacherManage(teacher.Role) {
		response.Forbidden(c, "insufficient teacher permissions")
		return
	}

	section, err := h.service.CreateSection(c.Request.Context(), req)
	if errors.Is(err, ErrOfferingNotFound) {
		response.NotFound(c, "offering not found")
	} else if errors.Is(err, ErrDuplicateSection) {
		response.Conflict(c, "section with this order index already exists")
	} else if err != nil {
		h.log.Error("create section failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, ToSectionResponse(section, time.Now()))
	}
}

func (h *Handler) ListSections(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offering_id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	sections, err := h.service.ListSections(c.Request.Context(), offeringID)
	if errors.Is(err, ErrOfferingNotFound) {
		response.NotFound(c, "offering not found")
	} else if err != nil {
		h.log.Error("list sections failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToSectionsResponse(sections, time.Now()))
	}
}

func (h *Handler) GetSection(c *gin.Context) {
	sectionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid section id")
		return
	}

	section, err := h.service.GetSection(c.Request.Context(), sectionID)
	if errors.Is(err, ErrSectionNotFound) {
		response.NotFound(c, "section not found")
	} else if err != nil {
		h.log.Error("get section failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToSectionResponse(section, time.Now()))
	}
}

func (h *Handler) UpdateSection(c *gin.Context) {
	sectionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid section id")
		return
	}

	section, err := h.service.GetSection(c.Request.Context(), sectionID)
	if errors.Is(err, ErrSectionNotFound) {
		response.NotFound(c, "section not found")
		return
	} else if err != nil {
		h.log.Error("get section failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	userID := middleware.GetUserID(c)
	teacher, err := h.service.GetTeacherRole(c.Request.Context(), section.OfferingID, userID)
	if errors.Is(err, ErrTeacherNotFound) {
		response.Forbidden(c, "not a teacher of this course")
		return
	} else if err != nil {
		h.log.Error("get teacher role failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !CanTeacherManage(teacher.Role) {
		response.Forbidden(c, "insufficient teacher permissions")
		return
	}

	var req UpdateSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updated, err := h.service.UpdateSection(c.Request.Context(), sectionID, req)
	if err != nil {
		h.log.Error("update section failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToSectionResponse(updated, time.Now()))
}

func (h *Handler) DeleteSection(c *gin.Context) {
	sectionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid section id")
		return
	}

	section, err := h.service.GetSection(c.Request.Context(), sectionID)
	if errors.Is(err, ErrSectionNotFound) {
		response.NotFound(c, "section not found")
		return
	} else if err != nil {
		h.log.Error("get section failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	userID := middleware.GetUserID(c)
	teacher, err := h.service.GetTeacherRole(c.Request.Context(), section.OfferingID, userID)
	if errors.Is(err, ErrTeacherNotFound) {
		response.Forbidden(c, "not a teacher of this course")
		return
	} else if err != nil {
		h.log.Error("get teacher role failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !CanTeacherManage(teacher.Role) {
		response.Forbidden(c, "insufficient teacher permissions")
		return
	}

	err = h.service.DeleteSection(c.Request.Context(), sectionID)
	if errors.Is(err, ErrSectionNotFound) {
		response.NotFound(c, "section not found")
	} else if err != nil {
		h.log.Error("delete section failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

// Helper functions

func (h *Handler) parseCourseFilters(c *gin.Context) (CourseFilters, error) {
	filters := CourseFilters{
		IsActive: pagination.ParseBool(c, "is_active"),
		Query:    c.Query("q"),
	}

	if deptIDStr := c.Query("department_id"); deptIDStr != "" {
		id, err := uuid.Parse(deptIDStr)
		if err != nil {
			return filters, errors.New("invalid department_id")
		}
		filters.DepartmentID = &id
	}

	if hasRequires := c.Query("has_requires"); hasRequires != "" {
		val := hasRequires == "true"
		filters.HasRequires = &val
	}

	return filters, nil
}

func (h *Handler) parseOfferingFilters(c *gin.Context) (OfferingFilters, error) {
	filters := OfferingFilters{
		IsActive: pagination.ParseBool(c, "is_active"),
	}

	if courseIDStr := c.Query("course_id"); courseIDStr != "" {
		id, err := uuid.Parse(courseIDStr)
		if err != nil {
			return filters, errors.New("invalid course_id")
		}
		filters.CourseID = &id
	}

	if semesterIDStr := c.Query("semester_id"); semesterIDStr != "" {
		id, err := uuid.Parse(semesterIDStr)
		if err != nil {
			return filters, errors.New("invalid semester_id")
		}
		filters.SemesterID = &id
	}

	if shift := c.Query("shift"); shift != "" {
		filters.Shift = &shift
	}

	if cohortYearStr := c.Query("cohort_year"); cohortYearStr != "" {
		var cohortYear int
		if _, err := fmt.Sscanf(cohortYearStr, "%d", &cohortYear); err == nil {
			filters.CohortYear = &cohortYear
		}
	}

	return filters, nil
}
