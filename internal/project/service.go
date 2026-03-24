package project

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ProjectRepository interface {
	Create(ctx context.Context, p *Project) error
	GetByID(ctx context.Context, id uuid.UUID) (*Project, error)
	Update(ctx context.Context, p *Project) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByOffering(ctx context.Context, offeringID uuid.UUID) ([]Project, error)
	ListPublishedByOffering(ctx context.Context, offeringID uuid.UUID, now time.Time) ([]Project, error)
}

type AttachmentRepository interface {
	Add(ctx context.Context, a *ProjectAttachment) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByProject(ctx context.Context, projectID uuid.UUID) ([]ProjectAttachment, error)
}

type RegistrationRepository interface {
	Register(ctx context.Context, r *Registration) error
	Unregister(ctx context.Context, projectID, teamID uuid.UUID) error
	GetByProject(ctx context.Context, projectID uuid.UUID) ([]RegistrationWithTeam, error)
	GetByTeam(ctx context.Context, projectID, teamID uuid.UUID) (*Registration, error)
	IsRegistered(ctx context.Context, projectID, teamID uuid.UUID) (bool, error)
}

type ProjectGroupRepository interface {
	Create(ctx context.Context, g *ProjectGroup) error
	GetByID(ctx context.Context, id uuid.UUID) (*ProjectGroup, error)
	Update(ctx context.Context, g *ProjectGroup) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByProject(ctx context.Context, projectID uuid.UUID) ([]ProjectGroupWithMembers, error)
	GetByStudent(ctx context.Context, projectID, studentID uuid.UUID) (*ProjectGroupWithMembers, error)
	AddMember(ctx context.Context, m *ProjectGroupMember) error
	GetMembers(ctx context.Context, groupID uuid.UUID) ([]GroupMemberInfo, error)
	Finalize(ctx context.Context, id uuid.UUID) error
}

type SubmissionRepository interface {
	Create(ctx context.Context, s *Submission) error
	GetByID(ctx context.Context, id uuid.UUID) (*Submission, error)
	Update(ctx context.Context, s *Submission) error
	GetByGroup(ctx context.Context, projectID, groupID uuid.UUID) (*Submission, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]Submission, error)
}

type SubmissionFileRepository interface {
	Add(ctx context.Context, f *SubmissionFile) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetBySubmission(ctx context.Context, submissionID uuid.UUID) ([]SubmissionFile, error)
	DeleteBySubmission(ctx context.Context, submissionID uuid.UUID) error
}

type GradeRepository interface {
	Upsert(ctx context.Context, g *Grade) error
	GetBySubmission(ctx context.Context, submissionID uuid.UUID) ([]GradeWithStudent, error)
	GetByStudent(ctx context.Context, submissionID, studentID uuid.UUID) (*Grade, error)
}

type EnrollmentChecker interface {
	IsEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error)
	AreAllEnrolled(ctx context.Context, offeringID uuid.UUID, studentIDs []uuid.UUID) (bool, error)
}

type TeacherChecker interface {
	IsTeacher(ctx context.Context, offeringID, userID uuid.UUID) (bool, error)
	IsTeacherOrAssistant(ctx context.Context, offeringID, userID uuid.UUID) (bool, error)
}

type TeamProvider interface {
	GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]uuid.UUID, error)
	GetTeamLeader(ctx context.Context, teamID uuid.UUID) (uuid.UUID, error)
	CountTeamMembers(ctx context.Context, teamID uuid.UUID) (int, error)
}

type FileOwnershipChecker interface {
	IsFileOwnedBy(ctx context.Context, fileID, userID uuid.UUID) (bool, error)
}

type Service struct {
	projects        ProjectRepository
	attachments     AttachmentRepository
	registrations   RegistrationRepository
	groups          ProjectGroupRepository
	submissions     SubmissionRepository
	submissionFiles SubmissionFileRepository
	grades          GradeRepository
	enrollment      EnrollmentChecker
	teachers        TeacherChecker
	teams           TeamProvider
	files           FileOwnershipChecker
}

func NewService(
	projects ProjectRepository,
	attachments AttachmentRepository,
	registrations RegistrationRepository,
	groups ProjectGroupRepository,
	submissions SubmissionRepository,
	submissionFiles SubmissionFileRepository,
	grades GradeRepository,
	enrollment EnrollmentChecker,
	teachers TeacherChecker,
	teams TeamProvider,
	files FileOwnershipChecker,
) *Service {
	return &Service{
		projects:        projects,
		attachments:     attachments,
		registrations:   registrations,
		groups:          groups,
		submissions:     submissions,
		submissionFiles: submissionFiles,
		grades:          grades,
		enrollment:      enrollment,
		teachers:        teachers,
		teams:           teams,
		files:           files,
	}
}

func (s *Service) CreateProject(ctx context.Context, p *Project) error {
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	return s.projects.Create(ctx, p)
}

func (s *Service) GetProject(ctx context.Context, id uuid.UUID) (*Project, error) {
	p, err := s.projects.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProjectNotFound
	}
	return p, nil
}

func (s *Service) UpdateProject(ctx context.Context, id uuid.UUID, updates ProjectUpdates) (*Project, error) {
	p, err := s.projects.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProjectNotFound
	}

	ApplyProjectUpdates(p, updates)

	if err := s.projects.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) DeleteProject(ctx context.Context, id uuid.UUID) error {
	p, err := s.projects.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrProjectNotFound
	}
	return s.projects.Delete(ctx, id)
}

func (s *Service) ListProjects(ctx context.Context, offeringID uuid.UUID) ([]Project, error) {
	return s.projects.ListByOffering(ctx, offeringID)
}

func (s *Service) ListPublishedProjects(ctx context.Context, offeringID uuid.UUID) ([]Project, error) {
	return s.projects.ListPublishedByOffering(ctx, offeringID, time.Now())
}

func (s *Service) GetAttachments(ctx context.Context, projectID uuid.UUID) ([]ProjectAttachment, error) {
	return s.attachments.GetByProject(ctx, projectID)
}

func (s *Service) AddAttachment(ctx context.Context, a *ProjectAttachment) error {
	a.ID = uuid.New()
	a.CreatedAt = time.Now()
	return s.attachments.Add(ctx, a)
}

func (s *Service) DeleteAttachment(ctx context.Context, id uuid.UUID) error {
	return s.attachments.Delete(ctx, id)
}

func (s *Service) Register(ctx context.Context, projectID, teamID uuid.UUID, projectTitle string) error {
	p, err := s.projects.GetByID(ctx, projectID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrProjectNotFound
	}

	now := time.Now()
	if !IsPublished(p.PublishAt, now) {
		return ErrNotPublished
	}

	if IsRegistrationClosed(p.RegistrationDeadline, now) {
		return ErrRegistrationClosed
	}

	registered, err := s.registrations.IsRegistered(ctx, projectID, teamID)
	if err != nil {
		return err
	}
	if registered {
		return ErrAlreadyRegistered
	}

	memberCount, err := s.teams.CountTeamMembers(ctx, teamID)
	if err != nil {
		return err
	}

	if memberCount < p.MinMembers {
		return ErrTeamTooSmall
	}
	if memberCount > p.MaxMembers {
		return ErrTeamTooLarge
	}

	members, err := s.teams.GetTeamMembers(ctx, teamID)
	if err != nil {
		return err
	}

	allEnrolled, err := s.enrollment.AreAllEnrolled(ctx, p.OfferingID, members)
	if err != nil {
		return err
	}
	if !allEnrolled {
		return ErrMembersNotEnrolled
	}

	r := &Registration{
		ID:           uuid.New(),
		ProjectID:    projectID,
		TeamID:       teamID,
		ProjectTitle: projectTitle,
		RegisteredAt: now,
	}

	return s.registrations.Register(ctx, r)
}

func (s *Service) Unregister(ctx context.Context, projectID, teamID uuid.UUID) error {
	p, err := s.projects.GetByID(ctx, projectID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrProjectNotFound
	}

	registered, err := s.registrations.IsRegistered(ctx, projectID, teamID)
	if err != nil {
		return err
	}
	if !registered {
		return ErrNotRegistered
	}

	return s.registrations.Unregister(ctx, projectID, teamID)
}

func (s *Service) GetRegistrations(ctx context.Context, projectID uuid.UUID) ([]RegistrationWithTeam, error) {
	return s.registrations.GetByProject(ctx, projectID)
}

func (s *Service) GetMyRegistration(ctx context.Context, projectID, teamID uuid.UUID) (*Registration, error) {
	return s.registrations.GetByTeam(ctx, projectID, teamID)
}

func (s *Service) CreateProjectGroups(ctx context.Context, projectID uuid.UUID) error {
	p, err := s.projects.GetByID(ctx, projectID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrProjectNotFound
	}

	registrations, err := s.registrations.GetByProject(ctx, projectID)
	if err != nil {
		return err
	}

	for _, reg := range registrations {
		if reg.MemberCount >= p.MinMembers {
			members, err := s.teams.GetTeamMembers(ctx, reg.TeamID)
			if err != nil {
				return err
			}

			group := &ProjectGroup{
				ID:           uuid.New(),
				ProjectID:    projectID,
				Name:         reg.TeamName,
				ProjectTitle: &reg.ProjectTitle,
				LeaderID:     reg.LeaderID,
				Finalized:    false,
				CreatedAt:    time.Now(),
			}

			if err := s.groups.Create(ctx, group); err != nil {
				return err
			}

			for _, studentID := range members {
				member := &ProjectGroupMember{
					ID:             uuid.New(),
					ProjectGroupID: group.ID,
					StudentID:      studentID,
					FromTeamID:     &reg.TeamID,
				}
				if err := s.groups.AddMember(ctx, member); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *Service) GetProjectGroups(ctx context.Context, projectID uuid.UUID) ([]ProjectGroupWithMembers, error) {
	return s.groups.GetByProject(ctx, projectID)
}

func (s *Service) GetMyProjectGroup(ctx context.Context, projectID, studentID uuid.UUID) (*ProjectGroupWithMembers, error) {
	return s.groups.GetByStudent(ctx, projectID, studentID)
}

func (s *Service) FinalizeProjectGroup(ctx context.Context, groupID uuid.UUID) error {
	g, err := s.groups.GetByID(ctx, groupID)
	if err != nil {
		return err
	}
	if g == nil {
		return ErrGroupNotFound
	}
	return s.groups.Finalize(ctx, groupID)
}

func (s *Service) CreateSubmission(ctx context.Context, projectID, groupID, userID uuid.UUID, content *string, files []FileInput) (*Submission, error) {
	p, err := s.projects.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProjectNotFound
	}

	g, err := s.groups.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, ErrGroupNotFound
	}

	if g.LeaderID != userID {
		return nil, ErrNotGroupLeader
	}

	existing, err := s.submissions.GetByGroup(ctx, projectID, groupID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAlreadySubmitted
	}

	for _, f := range files {
		owned, err := s.files.IsFileOwnedBy(ctx, f.StoredFileID, userID)
		if err != nil {
			return nil, err
		}
		if !owned {
			return nil, ErrFileNotOwned
		}
	}

	sub := &Submission{
		ID:             uuid.New(),
		ProjectID:      projectID,
		ProjectGroupID: groupID,
		Content:        content,
		CreatedAt:      time.Now(),
	}

	if err := s.submissions.Create(ctx, sub); err != nil {
		return nil, err
	}

	for i, f := range files {
		sf := &SubmissionFile{
			ID:           uuid.New(),
			SubmissionID: sub.ID,
			StoredFileID: f.StoredFileID,
			DisplayName:  f.DisplayName,
			OrderIndex:   i,
			CreatedAt:    time.Now(),
		}
		if err := s.submissionFiles.Add(ctx, sf); err != nil {
			return nil, err
		}
	}

	return sub, nil
}

func (s *Service) UpdateSubmission(ctx context.Context, submissionID, userID uuid.UUID, content *string, files []FileInput) (*Submission, error) {
	sub, err := s.submissions.GetByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubmissionNotFound
	}

	if sub.SubmittedAt != nil {
		return nil, ErrAlreadySubmitted
	}

	g, err := s.groups.GetByID(ctx, sub.ProjectGroupID)
	if err != nil {
		return nil, err
	}
	if g.LeaderID != userID {
		return nil, ErrNotGroupLeader
	}

	for _, f := range files {
		owned, err := s.files.IsFileOwnedBy(ctx, f.StoredFileID, userID)
		if err != nil {
			return nil, err
		}
		if !owned {
			return nil, ErrFileNotOwned
		}
	}

	sub.Content = content
	now := time.Now()
	sub.UpdatedAt = &now

	if err := s.submissions.Update(ctx, sub); err != nil {
		return nil, err
	}

	if err := s.submissionFiles.DeleteBySubmission(ctx, submissionID); err != nil {
		return nil, err
	}

	for i, f := range files {
		sf := &SubmissionFile{
			ID:           uuid.New(),
			SubmissionID: sub.ID,
			StoredFileID: f.StoredFileID,
			DisplayName:  f.DisplayName,
			OrderIndex:   i,
			CreatedAt:    time.Now(),
		}
		if err := s.submissionFiles.Add(ctx, sf); err != nil {
			return nil, err
		}
	}

	return sub, nil
}

func (s *Service) SubmitSubmission(ctx context.Context, submissionID, userID uuid.UUID) (*Submission, error) {
	sub, err := s.submissions.GetByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubmissionNotFound
	}

	if sub.SubmittedAt != nil {
		return nil, ErrAlreadySubmitted
	}

	g, err := s.groups.GetByID(ctx, sub.ProjectGroupID)
	if err != nil {
		return nil, err
	}
	if g.LeaderID != userID {
		return nil, ErrNotGroupLeader
	}

	p, err := s.projects.GetByID(ctx, sub.ProjectID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if !CanSubmit(p.Deadline, p.AllowLate, now) {
		return nil, ErrSubmissionsClosed
	}

	files, err := s.submissionFiles.GetBySubmission(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	if !HasContent(sub.Content, files) {
		return nil, ErrNoContent
	}

	sub.SubmittedAt = &now
	sub.SubmittedBy = &userID
	sub.UpdatedAt = &now

	if err := s.submissions.Update(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

func (s *Service) GetSubmission(ctx context.Context, id uuid.UUID) (*Submission, error) {
	return s.submissions.GetByID(ctx, id)
}

func (s *Service) GetSubmissionFiles(ctx context.Context, submissionID uuid.UUID) ([]SubmissionFile, error) {
	return s.submissionFiles.GetBySubmission(ctx, submissionID)
}

func (s *Service) ListSubmissions(ctx context.Context, projectID uuid.UUID) ([]Submission, error) {
	return s.submissions.ListByProject(ctx, projectID)
}

func (s *Service) GetMySubmission(ctx context.Context, projectID, studentID uuid.UUID) (*Submission, error) {
	g, err := s.groups.GetByStudent(ctx, projectID, studentID)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, ErrNotGroupMember
	}

	sub, err := s.submissions.GetByGroup(ctx, projectID, g.ID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubmissionNotFound
	}

	return sub, nil
}

func (s *Service) GradeSubmission(ctx context.Context, submissionID, studentID, graderID uuid.UUID, score float64, feedback *string) error {
	sub, err := s.submissions.GetByID(ctx, submissionID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrSubmissionNotFound
	}

	p, err := s.projects.GetByID(ctx, sub.ProjectID)
	if err != nil {
		return err
	}

	if !IsValidScore(score, p.MaxScore) {
		return ErrInvalidScore
	}

	now := time.Now()
	grade := &Grade{
		ID:           uuid.New(),
		SubmissionID: submissionID,
		StudentID:    studentID,
		Score:        &score,
		Feedback:     feedback,
		GradedBy:     &graderID,
		GradedAt:     &now,
	}

	return s.grades.Upsert(ctx, grade)
}

func (s *Service) GetGrades(ctx context.Context, submissionID uuid.UUID) ([]GradeWithStudent, error) {
	return s.grades.GetBySubmission(ctx, submissionID)
}

func (s *Service) GetMyGrade(ctx context.Context, submissionID, studentID uuid.UUID) (*Grade, error) {
	return s.grades.GetByStudent(ctx, submissionID, studentID)
}

func (s *Service) PublishScores(ctx context.Context, projectID uuid.UUID) error {
	p, err := s.projects.GetByID(ctx, projectID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrProjectNotFound
	}

	p.ScoresPublic = true
	return s.projects.Update(ctx, p)
}

func (s *Service) IsTeacher(ctx context.Context, offeringID, userID uuid.UUID) (bool, error) {
	return s.teachers.IsTeacher(ctx, offeringID, userID)
}

func (s *Service) IsTeacherOrAssistant(ctx context.Context, offeringID, userID uuid.UUID) (bool, error) {
	return s.teachers.IsTeacherOrAssistant(ctx, offeringID, userID)
}

func (s *Service) IsEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error) {
	return s.enrollment.IsEnrolled(ctx, offeringID, studentID)
}

type ProjectUpdates struct {
	Title                *string
	Body                 *string
	Deadline             *time.Time
	MaxScore             *float64
	MinMembers           *int
	MaxMembers           *int
	MergeTarget          *int
	RegistrationDeadline *time.Time
	Visibility           *string
	AllowLate            *bool
	PublishAt            *time.Time
}

type FileInput struct {
	StoredFileID uuid.UUID
	DisplayName  string
}
