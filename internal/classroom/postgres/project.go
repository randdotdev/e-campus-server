package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/classroom"
)

// ProjectRepository is the SQL adapter for projects, registrations, frozen
// groups, submissions, and per-member grades.
type ProjectRepository struct {
	db *sqlx.DB
}

func NewProjectRepository(db *sqlx.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

var _ classroom.ProjectRepository = (*ProjectRepository)(nil)

func (r *ProjectRepository) CreateProject(ctx context.Context, p *classroom.Project) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO projects (id, offering_id, title, body, deadline, max_score,
			min_members, max_members, merge_target, registration_deadline, visibility,
			allow_late, publish_at, scores_public, created_by, created_at)
		VALUES (:id, :offering_id, :title, :body, :deadline, :max_score,
			:min_members, :max_members, :merge_target, :registration_deadline, :visibility,
			:allow_late, :publish_at, :scores_public, :created_by, :created_at)`, p)
	return err
}

func (r *ProjectRepository) GetProject(ctx context.Context, offeringID, id uuid.UUID) (*classroom.Project, error) {
	var p classroom.Project
	err := r.db.GetContext(ctx, &p,
		`SELECT * FROM projects WHERE id = $1 AND offering_id = $2`, id, offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrProjectNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepository) ListProjects(ctx context.Context, offeringID uuid.UUID, publishedOnly bool) ([]classroom.Project, error) {
	projects := []classroom.Project{}
	query := `SELECT * FROM projects WHERE offering_id = $1`
	if publishedOnly {
		query += ` AND (publish_at IS NULL OR publish_at <= NOW())`
	}
	query += ` ORDER BY deadline`
	err := r.db.SelectContext(ctx, &projects, query, offeringID)
	return projects, err
}

func (r *ProjectRepository) UpdateProject(ctx context.Context, p *classroom.Project, expectedVersion int64) (int64, error) {
	return scanVersion(r.db.QueryRowxContext(ctx, `
		UPDATE projects SET
			title = $1, body = $2, deadline = $3, max_score = $4, min_members = $5,
			max_members = $6, merge_target = $7, registration_deadline = $8,
			visibility = $9, allow_late = $10, publish_at = $11, scores_public = $12,
			version = version + 1
		WHERE id = $13 AND version = $14
		RETURNING version`,
		p.Title, p.Body, p.Deadline, p.MaxScore, p.MinMembers,
		p.MaxMembers, p.MergeTarget, p.RegistrationDeadline,
		p.Visibility, p.AllowLate, p.PublishAt, p.ScoresPublic,
		p.ID, expectedVersion))
}

func (r *ProjectRepository) DeleteProject(ctx context.Context, offeringID, id uuid.UUID) ([]uuid.UUID, error) {
	var inodeIDs []uuid.UUID
	err := inTx(ctx, r.db, func(tx *sqlx.Tx) error {
		if err := tx.SelectContext(ctx, &inodeIDs, `
			SELECT inode_id FROM project_attachments WHERE project_id = $1
			UNION ALL
			SELECT psf.inode_id
			FROM project_submission_files psf
			JOIN project_submissions ps ON ps.id = psf.submission_id
			WHERE ps.project_id = $1`, id); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx,
			`DELETE FROM projects WHERE id = $1 AND offering_id = $2`, id, offeringID)
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n == 0 {
			return classroom.ErrProjectNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return inodeIDs, nil
}

func (r *ProjectRepository) CreateAttachment(ctx context.Context, a *classroom.ProjectAttachment) error {
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO project_attachments (id, project_id, inode_id, display_name, order_index, added_by)
		VALUES ($1, $2, $3, $4,
			(SELECT COALESCE(MAX(order_index), -1) + 1 FROM project_attachments WHERE project_id = $2), $5)
		RETURNING order_index`,
		a.ID, a.ProjectID, a.InodeID, a.DisplayName, a.AddedBy,
	).Scan(&a.OrderIndex)
	return err
}

func (r *ProjectRepository) ListAttachments(ctx context.Context, projectID uuid.UUID) ([]classroom.ProjectAttachment, error) {
	attachments := []classroom.ProjectAttachment{}
	err := r.db.SelectContext(ctx, &attachments,
		`SELECT * FROM project_attachments WHERE project_id = $1 ORDER BY order_index`, projectID)
	return attachments, err
}

func (r *ProjectRepository) GetAttachmentByName(ctx context.Context, projectID uuid.UUID, displayName string) (*classroom.ProjectAttachment, error) {
	var a classroom.ProjectAttachment
	err := r.db.GetContext(ctx, &a,
		`SELECT * FROM project_attachments WHERE project_id = $1 AND display_name = $2
		 ORDER BY order_index LIMIT 1`, projectID, displayName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *ProjectRepository) DeleteAttachment(ctx context.Context, projectID, id uuid.UUID) (uuid.UUID, error) {
	var inodeID uuid.UUID
	err := r.db.QueryRowxContext(ctx, `
		DELETE FROM project_attachments WHERE id = $1 AND project_id = $2
		RETURNING inode_id`, id, projectID,
	).Scan(&inodeID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, classroom.ErrAttachmentNotFound
	}
	return inodeID, err
}

// Register inserts under the registration window; the (project, team)
// unique pair settles a double registration.
func (r *ProjectRepository) Register(ctx context.Context, reg *classroom.Registration, minMembers, maxMembers int) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO project_registrations (id, project_id, team_id, project_title, registered_at)
		SELECT $1, $2, $3, $4, $5
		WHERE EXISTS (
			SELECT 1 FROM projects
			WHERE id = $2 AND (registration_deadline IS NULL OR registration_deadline >= $5))
		  AND (SELECT COUNT(*) FROM team_members WHERE team_id = $3) BETWEEN $6 AND $7`,
		reg.ID, reg.ProjectID, reg.TeamID, reg.ProjectTitle, reg.RegisteredAt,
		minMembers, maxMembers)
	if isUniqueViolation(err) {
		return classroom.ErrAlreadyRegistered
	}
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return classroom.ErrRegistrationClosed
	}
	return nil
}

func (r *ProjectRepository) Unregister(ctx context.Context, projectID, teamID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM project_registrations WHERE project_id = $1 AND team_id = $2`,
		projectID, teamID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return classroom.ErrNotRegistered
	}
	return nil
}

func (r *ProjectRepository) ListRegistrations(ctx context.Context, projectID uuid.UUID) ([]classroom.RegistrationWithTeam, error) {
	regs := []classroom.RegistrationWithTeam{}
	err := r.db.SelectContext(ctx, &regs, `
		SELECT pr.*, t.name AS team_name, t.leader_id,
		       (SELECT COUNT(*) FROM team_members WHERE team_id = t.id) AS member_count
		FROM project_registrations pr
		JOIN teams t ON t.id = pr.team_id
		WHERE pr.project_id = $1
		ORDER BY pr.registered_at`, projectID)
	return regs, err
}

func (r *ProjectRepository) IsTeamRegistered(ctx context.Context, projectID, teamID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM project_registrations WHERE project_id = $1 AND team_id = $2)`,
		projectID, teamID)
	return exists, err
}

// FormGroups freezes registrations into graded groups, one transaction,
// rerun-safe. Phase 1: every fitting team becomes its own group. Phase 2:
// under-sized teams are packed together per classroom.PlanMerge; a merged
// group's leader and title come from its largest constituent team. Teams
// that cannot reach the minimum even merged stay ungrouped and are
// counted back.
func (r *ProjectRepository) FormGroups(ctx context.Context, p *classroom.Project) (int, int, error) {
	formed, unmergedCount := 0, 0
	err := inTx(ctx, r.db, func(tx *sqlx.Tx) error {
		type reg struct {
			TeamID       uuid.UUID `db:"team_id"`
			ProjectTitle string    `db:"project_title"`
			TeamName     *string   `db:"team_name"`
			LeaderID     uuid.UUID `db:"leader_id"`
			Size         int       `db:"size"`
		}
		var regs []reg
		if err := tx.SelectContext(ctx, &regs, `
			SELECT pr.team_id, pr.project_title, t.name AS team_name, t.leader_id,
			       (SELECT COUNT(*) FROM team_members WHERE team_id = pr.team_id) AS size
			FROM project_registrations pr
			JOIN teams t ON t.id = pr.team_id
			WHERE pr.project_id = $1
			  AND NOT EXISTS (
				SELECT 1 FROM project_group_members pgm
				JOIN project_groups pg ON pg.id = pgm.project_group_id
				WHERE pg.project_id = $1 AND pgm.from_team_id = pr.team_id)
			ORDER BY pr.registered_at
			FOR UPDATE OF pr`, p.ID); err != nil {
			return err
		}

		byTeam := make(map[uuid.UUID]reg, len(regs))
		var undersized []classroom.MergeSeed
		freeze := func(teams []uuid.UUID, lead reg) error {
			groupID := uuid.New()
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO project_groups (id, project_id, name, project_title, leader_id, finalized)
				VALUES ($1, $2, $3, $4, $5, FALSE)`,
				groupID, p.ID, lead.TeamName, lead.ProjectTitle, lead.LeaderID); err != nil {
				return err
			}
			for _, teamID := range teams {
				if _, err := tx.ExecContext(ctx, `
					INSERT INTO project_group_members (id, project_group_id, student_id, from_team_id)
					SELECT gen_random_uuid(), $1, student_id, $2
					FROM team_members WHERE team_id = $2`,
					groupID, teamID); err != nil {
					return err
				}
			}
			formed++
			return nil
		}

		for _, rg := range regs {
			byTeam[rg.TeamID] = rg
			if rg.Size >= p.MinMembers && rg.Size <= p.MaxMembers {
				if err := freeze([]uuid.UUID{rg.TeamID}, rg); err != nil {
					return err
				}
				continue
			}
			if rg.Size < p.MinMembers {
				undersized = append(undersized, classroom.MergeSeed{TeamID: rg.TeamID, Size: rg.Size})
			}
			// Over-sized teams are the leader's problem: the size gate at
			// registration should have refused them; skip silently.
		}

		target := p.MinMembers
		if p.MergeTarget != nil {
			target = *p.MergeTarget
		}
		merged, unmerged := classroom.PlanMerge(undersized, p.MinMembers, p.MaxMembers, target)
		for _, group := range merged {
			largest := group[0]
			for _, seed := range group[1:] {
				if seed.Size > largest.Size {
					largest = seed
				}
			}
			teamIDs := make([]uuid.UUID, len(group))
			for i, seed := range group {
				teamIDs[i] = seed.TeamID
			}
			if err := freeze(teamIDs, byTeam[largest.TeamID]); err != nil {
				return err
			}
		}
		unmergedCount = len(unmerged)
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	return formed, unmergedCount, nil
}

func (r *ProjectRepository) ListGroups(ctx context.Context, projectID uuid.UUID) ([]classroom.ProjectGroupWithMembers, error) {
	groups := []classroom.ProjectGroup{}
	if err := r.db.SelectContext(ctx, &groups,
		`SELECT * FROM project_groups WHERE project_id = $1 ORDER BY created_at`, projectID); err != nil {
		return nil, err
	}
	result := make([]classroom.ProjectGroupWithMembers, len(groups))
	for i, g := range groups {
		members, err := r.groupMembers(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		result[i] = classroom.ProjectGroupWithMembers{ProjectGroup: g, Members: members}
	}
	return result, nil
}

func (r *ProjectRepository) GetGroup(ctx context.Context, projectID, groupID uuid.UUID) (*classroom.ProjectGroup, error) {
	var g classroom.ProjectGroup
	err := r.db.GetContext(ctx, &g,
		`SELECT * FROM project_groups WHERE id = $1 AND project_id = $2`, groupID, projectID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrGroupNotFound
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// GetMemberGroup returns nil (no error) when the user has no group here.
func (r *ProjectRepository) GetMemberGroup(ctx context.Context, projectID, userID uuid.UUID) (*classroom.ProjectGroupWithMembers, error) {
	var g classroom.ProjectGroup
	err := r.db.GetContext(ctx, &g, `
		SELECT pg.* FROM project_groups pg
		JOIN project_group_members pgm ON pgm.project_group_id = pg.id
		WHERE pg.project_id = $1 AND pgm.student_id = $2`, projectID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	members, err := r.groupMembers(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	return &classroom.ProjectGroupWithMembers{ProjectGroup: g, Members: members}, nil
}

func (r *ProjectRepository) groupMembers(ctx context.Context, groupID uuid.UUID) ([]classroom.GroupMemberInfo, error) {
	members := []classroom.GroupMemberInfo{}
	err := r.db.SelectContext(ctx, &members, `
		SELECT pgm.student_id AS user_id, u.full_name_en AS name, u.username
		FROM project_group_members pgm
		JOIN users u ON u.id = pgm.student_id
		WHERE pgm.project_group_id = $1
		ORDER BY u.full_name_en`, groupID)
	return members, err
}

// SaveGroupSubmission upserts the group's draft and replaces its files in
// one transaction; a submitted row refuses the write.
func (r *ProjectRepository) SaveGroupSubmission(ctx context.Context, sub *classroom.ProjectSubmission, files []classroom.ProjectSubmissionFile) ([]uuid.UUID, error) {
	var replaced []uuid.UUID
	err := inTx(ctx, r.db, func(tx *sqlx.Tx) error {
		var subID uuid.UUID
		err := tx.QueryRowxContext(ctx, `
			INSERT INTO project_submissions (id, project_id, project_group_id, content, created_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (project_id, project_group_id) DO UPDATE
				SET content = EXCLUDED.content, updated_at = NOW()
				WHERE project_submissions.submitted_at IS NULL
			RETURNING id`,
			sub.ID, sub.ProjectID, sub.ProjectGroupID, sub.Content, sub.CreatedAt,
		).Scan(&subID)
		if errors.Is(err, sql.ErrNoRows) {
			return classroom.ErrAlreadySubmitted
		}
		if err != nil {
			return err
		}
		if err := tx.SelectContext(ctx, &replaced,
			`DELETE FROM project_submission_files WHERE submission_id = $1 RETURNING inode_id`,
			subID); err != nil {
			return err
		}
		for i := range files {
			files[i].SubmissionID = subID
			if _, err := tx.NamedExecContext(ctx, `
				INSERT INTO project_submission_files (id, submission_id, inode_id, display_name, order_index, created_at)
				VALUES (:id, :submission_id, :inode_id, :display_name, :order_index, :created_at)`,
				files[i]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return replaced, nil
}

func (r *ProjectRepository) SubmitGroupSubmission(ctx context.Context, id uuid.UUID, submittedBy uuid.UUID, at time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE project_submissions
		SET submitted_at = $1, submitted_by = $2, updated_at = $1
		WHERE id = $3 AND submitted_at IS NULL`, at, submittedBy, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return classroom.ErrAlreadySubmitted
	}
	return nil
}

func (r *ProjectRepository) GetGroupSubmission(ctx context.Context, projectID, groupID uuid.UUID) (*classroom.ProjectSubmission, error) {
	var sub classroom.ProjectSubmission
	err := r.db.GetContext(ctx, &sub,
		`SELECT * FROM project_submissions WHERE project_id = $1 AND project_group_id = $2`,
		projectID, groupID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrSubmissionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *ProjectRepository) GetSubmission(ctx context.Context, projectID, id uuid.UUID) (*classroom.ProjectSubmission, error) {
	var sub classroom.ProjectSubmission
	err := r.db.GetContext(ctx, &sub,
		`SELECT * FROM project_submissions WHERE id = $1 AND project_id = $2`, id, projectID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrSubmissionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *ProjectRepository) ListSubmissions(ctx context.Context, projectID uuid.UUID) ([]classroom.ProjectSubmission, error) {
	subs := []classroom.ProjectSubmission{}
	err := r.db.SelectContext(ctx, &subs,
		`SELECT * FROM project_submissions WHERE project_id = $1 ORDER BY submitted_at NULLS LAST`,
		projectID)
	return subs, err
}

func (r *ProjectRepository) ListSubmissionFiles(ctx context.Context, submissionID uuid.UUID) ([]classroom.ProjectSubmissionFile, error) {
	files := []classroom.ProjectSubmissionFile{}
	err := r.db.SelectContext(ctx, &files,
		`SELECT * FROM project_submission_files WHERE submission_id = $1 ORDER BY order_index`,
		submissionID)
	return files, err
}

func (r *ProjectRepository) GetSubmissionFileByName(ctx context.Context, submissionID uuid.UUID, displayName string) (*classroom.ProjectSubmissionFile, error) {
	var f classroom.ProjectSubmissionFile
	err := r.db.GetContext(ctx, &f,
		`SELECT * FROM project_submission_files WHERE submission_id = $1 AND display_name = $2
		 ORDER BY order_index LIMIT 1`, submissionID, displayName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *ProjectRepository) UpsertGrade(ctx context.Context, g *classroom.ProjectGrade) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO project_grades (id, submission_id, student_id, score, feedback, graded_by, graded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (submission_id, student_id) DO UPDATE
			SET score = EXCLUDED.score, feedback = EXCLUDED.feedback,
			    graded_by = EXCLUDED.graded_by, graded_at = EXCLUDED.graded_at`,
		g.ID, g.SubmissionID, g.StudentID, g.Score, g.Feedback, g.GradedBy, g.GradedAt)
	if isForeignKeyViolation(err) {
		return classroom.ErrSubmissionNotFound
	}
	return err
}

func (r *ProjectRepository) ListGrades(ctx context.Context, submissionID uuid.UUID) ([]classroom.ProjectGradeWithStudent, error) {
	grades := []classroom.ProjectGradeWithStudent{}
	err := r.db.SelectContext(ctx, &grades, `
		SELECT pg.*, u.full_name_en AS student_name, u.username AS student_username
		FROM project_grades pg
		JOIN users u ON u.id = pg.student_id
		WHERE pg.submission_id = $1
		ORDER BY u.full_name_en`, submissionID)
	return grades, err
}

func (r *ProjectRepository) GetMemberGrade(ctx context.Context, submissionID, userID uuid.UUID) (*classroom.ProjectGrade, error) {
	var g classroom.ProjectGrade
	err := r.db.GetContext(ctx, &g,
		`SELECT * FROM project_grades WHERE submission_id = $1 AND student_id = $2`,
		submissionID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrSubmissionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}
