package main

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/classroom"
	classroomhttp "github.com/randdotdev/e-campus-server/internal/classroom/http"
	classroompg "github.com/randdotdev/e-campus-server/internal/classroom/postgres"
	"github.com/randdotdev/e-campus-server/internal/files"
	"github.com/randdotdev/e-campus-server/internal/management"
)

// classroomSet is what the classroom context exports.
type classroomSet struct {
	handler *classroomhttp.Handler
}

// wireClassroom builds the classroom context. mgmt.cohorts needs no adapter:
// management's service satisfies classroom's port as-is.
func wireClassroom(infra *infra, fls filesSet, mgmt managementSet, comm communicationSet) classroomSet {
	readers := classroompg.NewReaders(infra.db)
	fileStore := classroomFileStore{fls.inode}
	grades := classroomGradeWriter{enrollment: mgmt.enrollment}

	contentRepo := classroompg.NewContentRepository(infra.db)
	assignmentRepo := classroompg.NewAssignmentRepository(infra.db)
	questionRepo := classroompg.NewQuestionRepository(infra.db)
	examRepo := classroompg.NewExamRepository(infra.db)
	attendanceRepo := classroompg.NewAttendanceRepository(infra.db)
	gradingRepo := classroompg.NewGradingRepository(infra.db)
	qaRepo := classroompg.NewQARepository(infra.db)
	teamRepo := classroompg.NewTeamRepository(infra.db)
	projectRepo := classroompg.NewProjectRepository(infra.db)

	contentService := classroom.NewContentService(contentRepo, fileStore, mgmt.cohorts, infra.slog)
	assignmentService := classroom.NewAssignmentService(assignmentRepo, fileStore, comm.notification, infra.slog)
	questionService := classroom.NewQuestionService(questionRepo, fileStore, readers, infra.slog)
	examService := classroom.NewExamService(examRepo, questionRepo, readers, comm.notification, infra.slog)
	attendanceService := classroom.NewAttendanceService(attendanceRepo, readers, comm.notification, infra.slog)
	gradingService := classroom.NewGradingService(gradingRepo, examRepo, attendanceRepo, readers, readers, grades, comm.notification, infra.slog)
	qaService := classroom.NewQAService(qaRepo, fileStore, comm.mute, comm.notification, infra.slog)
	teamService := classroom.NewTeamService(teamRepo, readers, readers)
	projectService := classroom.NewProjectService(projectRepo, teamRepo, fileStore, readers, comm.notification, infra.slog)

	return classroomSet{handler: classroomhttp.NewHandler(
		contentService, assignmentService, questionService, examService,
		attendanceService, gradingService, qaService, teamService,
		projectService, infra.log,
	)}
}

// classroomFileStore satisfies classroom.FileStore. The embedded inode
// service covers Link/Unlink/Presign; only ResolveUpload translates.
type classroomFileStore struct {
	*files.InodeService
}

func (a classroomFileStore) ResolveUpload(ctx context.Context, actorID, uploadID uuid.UUID) (classroom.StoredFile, error) {
	ct, err := a.InodeService.ResolveUpload(ctx, actorID, uploadID)
	if errors.Is(err, files.ErrUploadNotFound) {
		return classroom.StoredFile{}, classroom.ErrUploadNotFound
	}
	if err != nil {
		return classroom.StoredFile{}, err
	}
	return classroom.StoredFile{InodeID: ct.InodeID, Name: ct.Name, SizeBytes: ct.SizeBytes, MimeType: ct.MimeType}, nil
}

// classroomGradeWriter writes final grades through management's enrollment
// service — the one sanctioned classroom write into management.
type classroomGradeWriter struct {
	enrollment *management.EnrollmentService
}

func (a classroomGradeWriter) SetEnrollmentGrade(ctx context.Context, offeringID, studentID uuid.UUID, grade float64, status string) error {
	return a.enrollment.SetFinalGrade(ctx, offeringID, studentID, grade, status)
}

func (a classroomGradeWriter) ClearEnrollmentGrades(ctx context.Context, offeringID uuid.UUID) error {
	return a.enrollment.ClearFinalGrades(ctx, offeringID)
}

func (a classroomGradeWriter) OfferingFinalized(ctx context.Context, offeringID uuid.UUID) (bool, error) {
	return a.enrollment.IsOfferingFinalized(ctx, offeringID)
}

func (a classroomGradeWriter) StudentGrades(ctx context.Context, offeringID uuid.UUID) ([]classroom.StudentGrade, error) {
	rows, err := a.enrollment.FinalGrades(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	grades := make([]classroom.StudentGrade, len(rows))
	for i, r := range rows {
		grades[i] = classroom.StudentGrade{
			StudentID: r.StudentID, StudentName: r.StudentName,
			FinalGrade: r.FinalGrade, Status: r.Status,
		}
	}
	return grades, nil
}
