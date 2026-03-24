package attendance

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockRepository struct {
	attendance       map[uuid.UUID]*Attendance
	excuses          map[uuid.UUID]*ExcuseRequest
	lessonExcuses    map[string]*ExcuseRequest
	lessonRecords    []AttendanceRecord
	summaries        []AttendanceSummary
	studentRecords   []StudentAttendance
	courseRecords    []CourseAttendance
	initializedCount int
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		attendance:    make(map[uuid.UUID]*Attendance),
		excuses:       make(map[uuid.UUID]*ExcuseRequest),
		lessonExcuses: make(map[string]*ExcuseRequest),
	}
}

func (m *mockRepository) InitializeAttendance(_ context.Context, lessonID uuid.UUID, studentIDs []uuid.UUID) (int, error) {
	m.initializedCount = len(studentIDs)
	for _, studentID := range studentIDs {
		a := &Attendance{
			ID:        uuid.New(),
			LessonID:  lessonID,
			StudentID: studentID,
			CreatedAt: time.Now(),
		}
		m.attendance[a.ID] = a
	}
	return len(studentIDs), nil
}

func (m *mockRepository) BulkUpdateAttendance(_ context.Context, _ uuid.UUID, markerID uuid.UUID, records []AttendanceUpdate) error {
	now := time.Now()
	for _, r := range records {
		if a, ok := m.attendance[r.ID]; ok {
			a.Percentage = r.Percentage
			a.MarkedBy = &markerID
			a.MarkedAt = &now
		}
	}
	return nil
}

func (m *mockRepository) UpdateAttendance(_ context.Context, a *Attendance) error {
	m.attendance[a.ID] = a
	return nil
}

func (m *mockRepository) GetAttendanceByID(_ context.Context, id uuid.UUID) (*Attendance, error) {
	if a, ok := m.attendance[id]; ok {
		return a, nil
	}
	return nil, ErrAttendanceNotFound
}

func (m *mockRepository) GetLessonAttendance(_ context.Context, _ uuid.UUID) ([]AttendanceRecord, error) {
	return m.lessonRecords, nil
}

func (m *mockRepository) GetOfferingAttendance(_ context.Context, _ uuid.UUID) ([]AttendanceRecord, error) {
	return m.lessonRecords, nil
}

func (m *mockRepository) GetAttendanceSummaries(_ context.Context, _ uuid.UUID) ([]AttendanceSummary, error) {
	return m.summaries, nil
}

func (m *mockRepository) GetStudentAttendance(_ context.Context, _, _ uuid.UUID) ([]StudentAttendance, error) {
	return m.studentRecords, nil
}

func (m *mockRepository) GetStudentCourseAttendances(_ context.Context, _ uuid.UUID) ([]CourseAttendance, error) {
	return m.courseRecords, nil
}

func (m *mockRepository) CreateExcuseRequest(_ context.Context, e *ExcuseRequest) error {
	e.ID = uuid.New()
	m.excuses[e.ID] = e
	key := e.LessonID.String() + "-" + e.StudentID.String()
	m.lessonExcuses[key] = e
	return nil
}

func (m *mockRepository) UpdateExcuseRequest(_ context.Context, e *ExcuseRequest) error {
	m.excuses[e.ID] = e
	return nil
}

func (m *mockRepository) GetExcuseRequestByID(_ context.Context, id uuid.UUID) (*ExcuseRequest, error) {
	if e, ok := m.excuses[id]; ok {
		return e, nil
	}
	return nil, ErrExcuseNotFound
}

func (m *mockRepository) GetExcuseByLessonAndStudent(_ context.Context, lessonID, studentID uuid.UUID) (*ExcuseRequest, error) {
	key := lessonID.String() + "-" + studentID.String()
	if e, ok := m.lessonExcuses[key]; ok {
		return e, nil
	}
	return nil, ErrExcuseNotFound
}

func (m *mockRepository) GetPendingExcuses(_ context.Context, _ uuid.UUID) ([]ExcuseRequest, error) {
	var pending []ExcuseRequest
	for _, e := range m.excuses {
		if e.Status == ExcuseStatusPending {
			pending = append(pending, *e)
		}
	}
	return pending, nil
}

type mockLessonChecker struct {
	lessons map[uuid.UUID]struct {
		offeringID uuid.UUID
		required   bool
	}
}

func newMockLessonChecker() *mockLessonChecker {
	return &mockLessonChecker{
		lessons: make(map[uuid.UUID]struct {
			offeringID uuid.UUID
			required   bool
		}),
	}
}

func (m *mockLessonChecker) GetLessonForAttendance(_ context.Context, lessonID uuid.UUID) (uuid.UUID, bool, error) {
	if l, ok := m.lessons[lessonID]; ok {
		return l.offeringID, l.required, nil
	}
	return uuid.Nil, false, ErrLessonNotFound
}

type mockEnrollmentChecker struct {
	enrollments map[string]bool
	studentIDs  []uuid.UUID
}

func newMockEnrollmentChecker() *mockEnrollmentChecker {
	return &mockEnrollmentChecker{
		enrollments: make(map[string]bool),
	}
}

func (m *mockEnrollmentChecker) IsStudentEnrolled(_ context.Context, studentID, offeringID uuid.UUID) (bool, error) {
	key := studentID.String() + "-" + offeringID.String()
	return m.enrollments[key], nil
}

func (m *mockEnrollmentChecker) GetEnrolledStudentIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return m.studentIDs, nil
}

func setupService() (*Service, *mockRepository, *mockLessonChecker, *mockEnrollmentChecker) {
	repo := newMockRepository()
	lessons := newMockLessonChecker()
	enrollment := newMockEnrollmentChecker()
	service := NewService(repo, lessons, enrollment, nil, nil)
	return service, repo, lessons, enrollment
}

func TestInitializeAttendance(t *testing.T) {
	ctx := context.Background()
	service, repo, lessons, enrollment := setupService()

	lessonID := uuid.New()
	offeringID := uuid.New()
	student1 := uuid.New()
	student2 := uuid.New()

	lessons.lessons[lessonID] = struct {
		offeringID uuid.UUID
		required   bool
	}{offeringID, true}

	enrollment.studentIDs = []uuid.UUID{student1, student2}

	count, err := service.InitializeAttendance(ctx, lessonID)
	if err != nil {
		t.Errorf("InitializeAttendance() error = %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 initialized, got %d", count)
	}
	if repo.initializedCount != 2 {
		t.Errorf("expected repo to have 2 initialized, got %d", repo.initializedCount)
	}
}

func TestInitializeAttendance_LessonNotFound(t *testing.T) {
	ctx := context.Background()
	service, _, _, _ := setupService()

	_, err := service.InitializeAttendance(ctx, uuid.New())
	if err != ErrLessonNotFound {
		t.Errorf("expected ErrLessonNotFound, got %v", err)
	}
}

func TestInitializeAttendance_NotRequired(t *testing.T) {
	ctx := context.Background()
	service, _, lessons, _ := setupService()

	lessonID := uuid.New()
	lessons.lessons[lessonID] = struct {
		offeringID uuid.UUID
		required   bool
	}{uuid.New(), false}

	_, err := service.InitializeAttendance(ctx, lessonID)
	if err != ErrAttendanceNotRequired {
		t.Errorf("expected ErrAttendanceNotRequired, got %v", err)
	}
}

func TestMarkAttendance(t *testing.T) {
	ctx := context.Background()
	service, repo, lessons, enrollment := setupService()

	lessonID := uuid.New()
	offeringID := uuid.New()
	studentID := uuid.New()
	markerID := uuid.New()

	lessons.lessons[lessonID] = struct {
		offeringID uuid.UUID
		required   bool
	}{offeringID, true}

	enrollment.studentIDs = []uuid.UUID{studentID}

	// First initialize
	_, _ = service.InitializeAttendance(ctx, lessonID)

	// Get the attendance ID
	var attendanceID uuid.UUID
	for id := range repo.attendance {
		attendanceID = id
		break
	}

	records := []AttendanceUpdate{
		{ID: attendanceID, Percentage: 100},
	}

	err := service.MarkAttendance(ctx, lessonID, markerID, records)
	if err != nil {
		t.Errorf("MarkAttendance() error = %v", err)
	}

	if repo.attendance[attendanceID].Percentage != 100 {
		t.Errorf("expected percentage 100, got %d", repo.attendance[attendanceID].Percentage)
	}
}

func TestMarkAttendance_InvalidPercentage(t *testing.T) {
	ctx := context.Background()
	service, _, lessons, _ := setupService()

	lessonID := uuid.New()
	offeringID := uuid.New()

	lessons.lessons[lessonID] = struct {
		offeringID uuid.UUID
		required   bool
	}{offeringID, true}

	records := []AttendanceUpdate{
		{ID: uuid.New(), Percentage: 33},
	}

	err := service.MarkAttendance(ctx, lessonID, uuid.New(), records)
	if err != ErrInvalidPercentage {
		t.Errorf("expected ErrInvalidPercentage, got %v", err)
	}
}

func TestRequestExcuse(t *testing.T) {
	ctx := context.Background()
	service, _, lessons, enrollment := setupService()

	lessonID := uuid.New()
	offeringID := uuid.New()
	studentID := uuid.New()

	lessons.lessons[lessonID] = struct {
		offeringID uuid.UUID
		required   bool
	}{offeringID, true}

	key := studentID.String() + "-" + offeringID.String()
	enrollment.enrollments[key] = true

	excuse, err := service.RequestExcuse(ctx, lessonID, studentID, "I was sick")
	if err != nil {
		t.Errorf("RequestExcuse() error = %v", err)
	}
	if excuse.Status != ExcuseStatusPending {
		t.Errorf("expected pending status, got %s", excuse.Status)
	}
}

func TestRequestExcuse_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	service, _, lessons, enrollment := setupService()

	lessonID := uuid.New()
	offeringID := uuid.New()
	studentID := uuid.New()

	lessons.lessons[lessonID] = struct {
		offeringID uuid.UUID
		required   bool
	}{offeringID, true}

	key := studentID.String() + "-" + offeringID.String()
	enrollment.enrollments[key] = true

	_, _ = service.RequestExcuse(ctx, lessonID, studentID, "First request")
	_, err := service.RequestExcuse(ctx, lessonID, studentID, "Second request")
	if err != ErrExcuseAlreadyExists {
		t.Errorf("expected ErrExcuseAlreadyExists, got %v", err)
	}
}

func TestReviewExcuse(t *testing.T) {
	ctx := context.Background()
	service, repo, _, _ := setupService()

	excuseID := uuid.New()
	studentID := uuid.New()
	reviewerID := uuid.New()

	repo.excuses[excuseID] = &ExcuseRequest{
		ID:        excuseID,
		StudentID: studentID,
		Status:    ExcuseStatusPending,
		CreatedAt: time.Now(),
	}

	note := "Medical certificate verified"
	err := service.ReviewExcuse(ctx, excuseID, reviewerID, ExcuseStatusApproved, &note)
	if err != nil {
		t.Errorf("ReviewExcuse() error = %v", err)
	}

	if repo.excuses[excuseID].Status != ExcuseStatusApproved {
		t.Errorf("expected approved status")
	}
}

func TestReviewExcuse_AlreadyReviewed(t *testing.T) {
	ctx := context.Background()
	service, repo, _, _ := setupService()

	excuseID := uuid.New()
	reviewerID := uuid.New()

	repo.excuses[excuseID] = &ExcuseRequest{
		ID:     excuseID,
		Status: ExcuseStatusApproved,
	}

	err := service.ReviewExcuse(ctx, excuseID, reviewerID, ExcuseStatusRejected, nil)
	if err != ErrExcuseAlreadyReviewed {
		t.Errorf("expected ErrExcuseAlreadyReviewed, got %v", err)
	}
}

func TestReviewExcuse_CannotReviewOwn(t *testing.T) {
	ctx := context.Background()
	service, repo, _, _ := setupService()

	excuseID := uuid.New()
	studentID := uuid.New()

	repo.excuses[excuseID] = &ExcuseRequest{
		ID:        excuseID,
		StudentID: studentID,
		Status:    ExcuseStatusPending,
	}

	err := service.ReviewExcuse(ctx, excuseID, studentID, ExcuseStatusApproved, nil)
	if err != ErrCannotExcuseOwnAttendance {
		t.Errorf("expected ErrCannotExcuseOwnAttendance, got %v", err)
	}
}

func TestGetAttendanceSummaries(t *testing.T) {
	ctx := context.Background()
	service, repo, _, _ := setupService()

	repo.summaries = []AttendanceSummary{
		{
			StudentID:     uuid.New(),
			StudentName:   "Test Student",
			TotalHours:    10,
			AttendedHours: 8,
			ExcusedHours:  2,
		},
	}

	summaries, err := service.GetAttendanceSummaries(ctx, uuid.New())
	if err != nil {
		t.Errorf("GetAttendanceSummaries() error = %v", err)
	}
	if len(summaries) != 1 {
		t.Errorf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].AttendanceRate != 100 {
		t.Errorf("expected 100%% rate, got %v", summaries[0].AttendanceRate)
	}
}
