package project

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// ProjectRepository

func (r *Repository) Create(ctx context.Context, p *Project) error {
	query := `INSERT INTO projects (id, offering_id, title, body, deadline, max_score, min_members, max_members, merge_target, registration_deadline, visibility, allow_late, publish_at, scores_public, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`
	_, err := r.db.ExecContext(ctx, query,
		p.ID, p.OfferingID, p.Title, p.Body, p.Deadline, p.MaxScore,
		p.MinMembers, p.MaxMembers, p.MergeTarget, p.RegistrationDeadline,
		p.Visibility, p.AllowLate, p.PublishAt, p.ScoresPublic, p.CreatedBy, p.CreatedAt)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	var p Project
	query := `SELECT id, offering_id, title, body, deadline, max_score, min_members, max_members, merge_target, registration_deadline, visibility, allow_late, publish_at, scores_public, created_by, created_at
		FROM projects WHERE id = $1`
	err := r.db.GetContext(ctx, &p, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

func (r *Repository) Update(ctx context.Context, p *Project) error {
	query := `UPDATE projects SET title = $1, body = $2, deadline = $3, max_score = $4, min_members = $5, max_members = $6, merge_target = $7, registration_deadline = $8, visibility = $9, allow_late = $10, publish_at = $11, scores_public = $12
		WHERE id = $13`
	_, err := r.db.ExecContext(ctx, query,
		p.Title, p.Body, p.Deadline, p.MaxScore, p.MinMembers, p.MaxMembers,
		p.MergeTarget, p.RegistrationDeadline, p.Visibility, p.AllowLate,
		p.PublishAt, p.ScoresPublic, p.ID)
	return err
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id)
	return err
}

func (r *Repository) ListByOffering(ctx context.Context, offeringID uuid.UUID) ([]Project, error) {
	var projects []Project
	query := `SELECT id, offering_id, title, body, deadline, max_score, min_members, max_members, merge_target, registration_deadline, visibility, allow_late, publish_at, scores_public, created_by, created_at
		FROM projects WHERE offering_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &projects, query, offeringID)
	return projects, err
}

func (r *Repository) ListPublishedByOffering(ctx context.Context, offeringID uuid.UUID, now time.Time) ([]Project, error) {
	var projects []Project
	query := `SELECT id, offering_id, title, body, deadline, max_score, min_members, max_members, merge_target, registration_deadline, visibility, allow_late, publish_at, scores_public, created_by, created_at
		FROM projects WHERE offering_id = $1 AND (publish_at IS NULL OR publish_at <= $2) ORDER BY deadline`
	err := r.db.SelectContext(ctx, &projects, query, offeringID, now)
	return projects, err
}

// AttachmentRepository

func (r *Repository) Add(ctx context.Context, a *ProjectAttachment) error {
	query := `INSERT INTO project_attachments (id, project_id, stored_file_id, display_name, order_index, added_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query, a.ID, a.ProjectID, a.StoredFileID, a.DisplayName, a.OrderIndex, a.AddedBy, a.CreatedAt)
	return err
}

func (r *Repository) DeleteAttachment(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_attachments WHERE id = $1`, id)
	return err
}

func (r *Repository) GetByProject(ctx context.Context, projectID uuid.UUID) ([]ProjectAttachment, error) {
	var attachments []ProjectAttachment
	query := `SELECT id, project_id, stored_file_id, display_name, order_index, added_by, created_at
		FROM project_attachments WHERE project_id = $1 ORDER BY order_index`
	err := r.db.SelectContext(ctx, &attachments, query, projectID)
	return attachments, err
}

// RegistrationRepository

func (r *Repository) Register(ctx context.Context, reg *Registration) error {
	query := `INSERT INTO project_registrations (id, project_id, team_id, project_title, registered_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query, reg.ID, reg.ProjectID, reg.TeamID, reg.ProjectTitle, reg.RegisteredAt)
	return err
}

func (r *Repository) Unregister(ctx context.Context, projectID, teamID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_registrations WHERE project_id = $1 AND team_id = $2`, projectID, teamID)
	return err
}

func (r *Repository) GetByProjectRegistrations(ctx context.Context, projectID uuid.UUID) ([]RegistrationWithTeam, error) {
	var registrations []RegistrationWithTeam
	query := `SELECT pr.id, pr.project_id, pr.team_id, pr.project_title, pr.registered_at,
			t.name as team_name, t.leader_id, u.full_name_en as leader_name,
			(SELECT COUNT(*) FROM team_members WHERE team_id = t.id) as member_count
		FROM project_registrations pr
		JOIN teams t ON t.id = pr.team_id
		JOIN users u ON u.id = t.leader_id
		WHERE pr.project_id = $1
		ORDER BY pr.registered_at`
	err := r.db.SelectContext(ctx, &registrations, query, projectID)
	return registrations, err
}

func (r *Repository) GetByTeam(ctx context.Context, projectID, teamID uuid.UUID) (*Registration, error) {
	var reg Registration
	query := `SELECT id, project_id, team_id, project_title, registered_at
		FROM project_registrations WHERE project_id = $1 AND team_id = $2`
	err := r.db.GetContext(ctx, &reg, query, projectID, teamID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &reg, err
}

func (r *Repository) IsRegistered(ctx context.Context, projectID, teamID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM project_registrations WHERE project_id = $1 AND team_id = $2)`
	err := r.db.GetContext(ctx, &exists, query, projectID, teamID)
	return exists, err
}

// ProjectGroupRepository

func (r *Repository) CreateGroup(ctx context.Context, g *ProjectGroup) error {
	query := `INSERT INTO project_groups (id, project_id, name, project_title, leader_id, finalized, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query, g.ID, g.ProjectID, g.Name, g.ProjectTitle, g.LeaderID, g.Finalized, g.CreatedAt)
	return err
}

func (r *Repository) GetGroupByID(ctx context.Context, id uuid.UUID) (*ProjectGroup, error) {
	var g ProjectGroup
	query := `SELECT id, project_id, name, project_title, leader_id, finalized, created_at
		FROM project_groups WHERE id = $1`
	err := r.db.GetContext(ctx, &g, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &g, err
}

func (r *Repository) UpdateGroup(ctx context.Context, g *ProjectGroup) error {
	query := `UPDATE project_groups SET name = $1, project_title = $2, leader_id = $3, finalized = $4 WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query, g.Name, g.ProjectTitle, g.LeaderID, g.Finalized, g.ID)
	return err
}

func (r *Repository) DeleteGroup(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_groups WHERE id = $1`, id)
	return err
}

func (r *Repository) GetGroupsByProject(ctx context.Context, projectID uuid.UUID) ([]ProjectGroupWithMembers, error) {
	var groups []ProjectGroup
	query := `SELECT id, project_id, name, project_title, leader_id, finalized, created_at
		FROM project_groups WHERE project_id = $1 ORDER BY created_at`
	err := r.db.SelectContext(ctx, &groups, query, projectID)
	if err != nil {
		return nil, err
	}

	result := make([]ProjectGroupWithMembers, len(groups))
	for i, g := range groups {
		members, err := r.GetGroupMembers(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		result[i] = ProjectGroupWithMembers{
			ProjectGroup: g,
			Members:      members,
			MemberCount:  len(members),
		}
	}

	return result, nil
}

func (r *Repository) GetGroupByStudent(ctx context.Context, projectID, studentID uuid.UUID) (*ProjectGroupWithMembers, error) {
	var g ProjectGroup
	query := `SELECT pg.id, pg.project_id, pg.name, pg.project_title, pg.leader_id, pg.finalized, pg.created_at
		FROM project_groups pg
		JOIN project_group_members pgm ON pgm.project_group_id = pg.id
		WHERE pg.project_id = $1 AND pgm.student_id = $2`
	err := r.db.GetContext(ctx, &g, query, projectID, studentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	members, err := r.GetGroupMembers(ctx, g.ID)
	if err != nil {
		return nil, err
	}

	return &ProjectGroupWithMembers{
		ProjectGroup: g,
		Members:      members,
		MemberCount:  len(members),
	}, nil
}

func (r *Repository) AddGroupMember(ctx context.Context, m *ProjectGroupMember) error {
	query := `INSERT INTO project_group_members (id, project_group_id, student_id, from_team_id)
		VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, m.ID, m.ProjectGroupID, m.StudentID, m.FromTeamID)
	return err
}

func (r *Repository) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]GroupMemberInfo, error) {
	var members []GroupMemberInfo
	query := `SELECT pgm.student_id, u.full_name_en as student_name, pgm.from_team_id
		FROM project_group_members pgm
		JOIN users u ON u.id = pgm.student_id
		WHERE pgm.project_group_id = $1`
	err := r.db.SelectContext(ctx, &members, query, groupID)
	return members, err
}

func (r *Repository) FinalizeGroup(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE project_groups SET finalized = true WHERE id = $1`, id)
	return err
}

// SubmissionRepository

func (r *Repository) CreateSubmission(ctx context.Context, s *Submission) error {
	query := `INSERT INTO project_submissions (id, project_id, project_group_id, content, submitted_at, submitted_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query, s.ID, s.ProjectID, s.ProjectGroupID, s.Content, s.SubmittedAt, s.SubmittedBy, s.CreatedAt, s.UpdatedAt)
	return err
}

func (r *Repository) GetSubmissionByID(ctx context.Context, id uuid.UUID) (*Submission, error) {
	var s Submission
	query := `SELECT id, project_id, project_group_id, content, submitted_at, submitted_by, created_at, updated_at
		FROM project_submissions WHERE id = $1`
	err := r.db.GetContext(ctx, &s, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &s, err
}

func (r *Repository) UpdateSubmission(ctx context.Context, s *Submission) error {
	query := `UPDATE project_submissions SET content = $1, submitted_at = $2, submitted_by = $3, updated_at = $4 WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query, s.Content, s.SubmittedAt, s.SubmittedBy, s.UpdatedAt, s.ID)
	return err
}

func (r *Repository) GetSubmissionByGroup(ctx context.Context, projectID, groupID uuid.UUID) (*Submission, error) {
	var s Submission
	query := `SELECT id, project_id, project_group_id, content, submitted_at, submitted_by, created_at, updated_at
		FROM project_submissions WHERE project_id = $1 AND project_group_id = $2`
	err := r.db.GetContext(ctx, &s, query, projectID, groupID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &s, err
}

func (r *Repository) ListSubmissionsByProject(ctx context.Context, projectID uuid.UUID) ([]Submission, error) {
	var submissions []Submission
	query := `SELECT id, project_id, project_group_id, content, submitted_at, submitted_by, created_at, updated_at
		FROM project_submissions WHERE project_id = $1 ORDER BY created_at`
	err := r.db.SelectContext(ctx, &submissions, query, projectID)
	return submissions, err
}

// SubmissionFileRepository

func (r *Repository) AddSubmissionFile(ctx context.Context, f *SubmissionFile) error {
	query := `INSERT INTO project_submission_files (id, submission_id, stored_file_id, display_name, order_index, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, f.ID, f.SubmissionID, f.StoredFileID, f.DisplayName, f.OrderIndex, f.CreatedAt)
	return err
}

func (r *Repository) DeleteSubmissionFile(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_submission_files WHERE id = $1`, id)
	return err
}

func (r *Repository) GetSubmissionFiles(ctx context.Context, submissionID uuid.UUID) ([]SubmissionFile, error) {
	var files []SubmissionFile
	query := `SELECT id, submission_id, stored_file_id, display_name, order_index, created_at
		FROM project_submission_files WHERE submission_id = $1 ORDER BY order_index`
	err := r.db.SelectContext(ctx, &files, query, submissionID)
	return files, err
}

func (r *Repository) DeleteSubmissionFilesBySubmission(ctx context.Context, submissionID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_submission_files WHERE submission_id = $1`, submissionID)
	return err
}

// GradeRepository

func (r *Repository) UpsertGrade(ctx context.Context, g *Grade) error {
	query := `INSERT INTO project_grades (id, submission_id, student_id, score, feedback, graded_by, graded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (submission_id, student_id) DO UPDATE SET score = $4, feedback = $5, graded_by = $6, graded_at = $7`
	_, err := r.db.ExecContext(ctx, query, g.ID, g.SubmissionID, g.StudentID, g.Score, g.Feedback, g.GradedBy, g.GradedAt)
	return err
}

func (r *Repository) GetGradesBySubmission(ctx context.Context, submissionID uuid.UUID) ([]GradeWithStudent, error) {
	var grades []GradeWithStudent
	query := `SELECT pg.id, pg.submission_id, pg.student_id, pg.score, pg.feedback, pg.graded_by, pg.graded_at, u.full_name_en as student_name
		FROM project_grades pg
		JOIN users u ON u.id = pg.student_id
		WHERE pg.submission_id = $1`
	err := r.db.SelectContext(ctx, &grades, query, submissionID)
	return grades, err
}

func (r *Repository) GetGradeByStudent(ctx context.Context, submissionID, studentID uuid.UUID) (*Grade, error) {
	var g Grade
	query := `SELECT id, submission_id, student_id, score, feedback, graded_by, graded_at
		FROM project_grades WHERE submission_id = $1 AND student_id = $2`
	err := r.db.GetContext(ctx, &g, query, submissionID, studentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &g, err
}

// Adapter methods to match interface names

type ProjectRepo struct {
	*Repository
}

func (r *ProjectRepo) Create(ctx context.Context, p *Project) error {
	return r.Repository.Create(ctx, p)
}
func (r *ProjectRepo) GetByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	return r.Repository.GetByID(ctx, id)
}
func (r *ProjectRepo) Update(ctx context.Context, p *Project) error {
	return r.Repository.Update(ctx, p)
}
func (r *ProjectRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.Repository.Delete(ctx, id)
}
func (r *ProjectRepo) ListByOffering(ctx context.Context, offeringID uuid.UUID) ([]Project, error) {
	return r.Repository.ListByOffering(ctx, offeringID)
}
func (r *ProjectRepo) ListPublishedByOffering(ctx context.Context, offeringID uuid.UUID, now time.Time) ([]Project, error) {
	return r.Repository.ListPublishedByOffering(ctx, offeringID, now)
}

type AttachmentRepo struct {
	*Repository
}

func (r *AttachmentRepo) Add(ctx context.Context, a *ProjectAttachment) error {
	return r.Repository.Add(ctx, a)
}
func (r *AttachmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.DeleteAttachment(ctx, id)
}
func (r *AttachmentRepo) GetByProject(ctx context.Context, projectID uuid.UUID) ([]ProjectAttachment, error) {
	return r.Repository.GetByProject(ctx, projectID)
}

type RegistrationRepo struct {
	*Repository
}

func (r *RegistrationRepo) Register(ctx context.Context, reg *Registration) error {
	return r.Repository.Register(ctx, reg)
}
func (r *RegistrationRepo) Unregister(ctx context.Context, projectID, teamID uuid.UUID) error {
	return r.Repository.Unregister(ctx, projectID, teamID)
}
func (r *RegistrationRepo) GetByProject(ctx context.Context, projectID uuid.UUID) ([]RegistrationWithTeam, error) {
	return r.GetByProjectRegistrations(ctx, projectID)
}
func (r *RegistrationRepo) GetByTeam(ctx context.Context, projectID, teamID uuid.UUID) (*Registration, error) {
	return r.Repository.GetByTeam(ctx, projectID, teamID)
}
func (r *RegistrationRepo) IsRegistered(ctx context.Context, projectID, teamID uuid.UUID) (bool, error) {
	return r.Repository.IsRegistered(ctx, projectID, teamID)
}

type GroupRepo struct {
	*Repository
}

func (r *GroupRepo) Create(ctx context.Context, g *ProjectGroup) error {
	return r.CreateGroup(ctx, g)
}
func (r *GroupRepo) GetByID(ctx context.Context, id uuid.UUID) (*ProjectGroup, error) {
	return r.GetGroupByID(ctx, id)
}
func (r *GroupRepo) Update(ctx context.Context, g *ProjectGroup) error {
	return r.UpdateGroup(ctx, g)
}
func (r *GroupRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.DeleteGroup(ctx, id)
}
func (r *GroupRepo) GetByProject(ctx context.Context, projectID uuid.UUID) ([]ProjectGroupWithMembers, error) {
	return r.GetGroupsByProject(ctx, projectID)
}
func (r *GroupRepo) GetByStudent(ctx context.Context, projectID, studentID uuid.UUID) (*ProjectGroupWithMembers, error) {
	return r.GetGroupByStudent(ctx, projectID, studentID)
}
func (r *GroupRepo) AddMember(ctx context.Context, m *ProjectGroupMember) error {
	return r.AddGroupMember(ctx, m)
}
func (r *GroupRepo) GetMembers(ctx context.Context, groupID uuid.UUID) ([]GroupMemberInfo, error) {
	return r.GetGroupMembers(ctx, groupID)
}
func (r *GroupRepo) Finalize(ctx context.Context, id uuid.UUID) error {
	return r.FinalizeGroup(ctx, id)
}

type SubmissionRepo struct {
	*Repository
}

func (r *SubmissionRepo) Create(ctx context.Context, s *Submission) error {
	return r.CreateSubmission(ctx, s)
}
func (r *SubmissionRepo) GetByID(ctx context.Context, id uuid.UUID) (*Submission, error) {
	return r.GetSubmissionByID(ctx, id)
}
func (r *SubmissionRepo) Update(ctx context.Context, s *Submission) error {
	return r.UpdateSubmission(ctx, s)
}
func (r *SubmissionRepo) GetByGroup(ctx context.Context, projectID, groupID uuid.UUID) (*Submission, error) {
	return r.GetSubmissionByGroup(ctx, projectID, groupID)
}
func (r *SubmissionRepo) ListByProject(ctx context.Context, projectID uuid.UUID) ([]Submission, error) {
	return r.ListSubmissionsByProject(ctx, projectID)
}

type SubmissionFileRepo struct {
	*Repository
}

func (r *SubmissionFileRepo) Add(ctx context.Context, f *SubmissionFile) error {
	return r.AddSubmissionFile(ctx, f)
}
func (r *SubmissionFileRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.DeleteSubmissionFile(ctx, id)
}
func (r *SubmissionFileRepo) GetBySubmission(ctx context.Context, submissionID uuid.UUID) ([]SubmissionFile, error) {
	return r.GetSubmissionFiles(ctx, submissionID)
}
func (r *SubmissionFileRepo) DeleteBySubmission(ctx context.Context, submissionID uuid.UUID) error {
	return r.DeleteSubmissionFilesBySubmission(ctx, submissionID)
}

type GradeRepo struct {
	*Repository
}

func (r *GradeRepo) Upsert(ctx context.Context, g *Grade) error {
	return r.UpsertGrade(ctx, g)
}
func (r *GradeRepo) GetBySubmission(ctx context.Context, submissionID uuid.UUID) ([]GradeWithStudent, error) {
	return r.GetGradesBySubmission(ctx, submissionID)
}
func (r *GradeRepo) GetByStudent(ctx context.Context, submissionID, studentID uuid.UUID) (*Grade, error) {
	return r.GetGradeByStudent(ctx, submissionID, studentID)
}

func NewProjectRepo(db *sqlx.DB) *ProjectRepo {
	return &ProjectRepo{NewRepository(db)}
}
func NewAttachmentRepo(db *sqlx.DB) *AttachmentRepo {
	return &AttachmentRepo{NewRepository(db)}
}
func NewRegistrationRepo(db *sqlx.DB) *RegistrationRepo {
	return &RegistrationRepo{NewRepository(db)}
}
func NewGroupRepo(db *sqlx.DB) *GroupRepo {
	return &GroupRepo{NewRepository(db)}
}
func NewSubmissionRepo(db *sqlx.DB) *SubmissionRepo {
	return &SubmissionRepo{NewRepository(db)}
}
func NewSubmissionFileRepo(db *sqlx.DB) *SubmissionFileRepo {
	return &SubmissionFileRepo{NewRepository(db)}
}
func NewGradeRepo(db *sqlx.DB) *GradeRepo {
	return &GradeRepo{NewRepository(db)}
}
