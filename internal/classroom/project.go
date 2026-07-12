package classroom

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"
)

// A project is team-based graded work in one offering. Teams register
// while the window is open; when the teacher forms groups, each valid
// registration is frozen into a project group — a snapshot of the team at
// that moment, so later team churn cannot rewrite who did the work. The
// group leader submits once per project; grades land per member, because
// contribution is judged per person even when work is shared.

// ── Value objects ───────────────────────────────────────────────────────────

// ProjectVisibility is who may see the registrations list.
type ProjectVisibility string

const (
	ProjectHidden     ProjectVisibility = "hidden"     // teacher only
	ProjectRegistered ProjectVisibility = "registered" // registered teams
	ProjectAll        ProjectVisibility = "all"        // whole offering
)

func ValidProjectVisibility(v ProjectVisibility) bool {
	return v == ProjectHidden || v == ProjectRegistered || v == ProjectAll
}

// ── Entities ────────────────────────────────────────────────────────────────

type Project struct {
	ID         uuid.UUID `db:"id"`
	OfferingID uuid.UUID `db:"offering_id"`
	Title      string    `db:"title"`
	Body       *string   `db:"body"`
	Deadline   time.Time `db:"deadline"`
	MaxScore   float64   `db:"max_score"`
	MinMembers int       `db:"min_members"`
	MaxMembers int       `db:"max_members"`
	// MergeTarget is the group size FormGroups aims for when packing
	// under-sized teams together; nil means aim for MinMembers.
	MergeTarget          *int              `db:"merge_target"`
	RegistrationDeadline *time.Time        `db:"registration_deadline"`
	Visibility           ProjectVisibility `db:"visibility"`
	AllowLate            bool              `db:"allow_late"`
	PublishAt            *time.Time        `db:"publish_at"`
	ScoresPublic         bool              `db:"scores_public"`
	CreatedBy            *uuid.UUID        `db:"created_by"`
	Version              int64             `db:"version"`
	CreatedAt            time.Time         `db:"created_at"`
}

type ProjectAttachment struct {
	ID          uuid.UUID  `db:"id"`
	ProjectID   uuid.UUID  `db:"project_id"`
	InodeID     uuid.UUID  `db:"inode_id"`
	DisplayName string     `db:"display_name"`
	OrderIndex  int        `db:"order_index"`
	AddedBy     *uuid.UUID `db:"added_by"`
	CreatedAt   time.Time  `db:"created_at"`
}

// Registration is a team's claim on a project, with the title the team
// chose for its own take on it.
type Registration struct {
	ID           uuid.UUID `db:"id"`
	ProjectID    uuid.UUID `db:"project_id"`
	TeamID       uuid.UUID `db:"team_id"`
	ProjectTitle string    `db:"project_title"`
	RegisteredAt time.Time `db:"registered_at"`
}

// ProjectGroup is a frozen registration: the roster that will actually be
// graded.
type ProjectGroup struct {
	ID           uuid.UUID `db:"id"`
	ProjectID    uuid.UUID `db:"project_id"`
	Name         *string   `db:"name"`
	ProjectTitle *string   `db:"project_title"`
	LeaderID     uuid.UUID `db:"leader_id"`
	Finalized    bool      `db:"finalized"`
	CreatedAt    time.Time `db:"created_at"`
}

// ProjectSubmission is the group's single submission.
type ProjectSubmission struct {
	ID             uuid.UUID  `db:"id"`
	ProjectID      uuid.UUID  `db:"project_id"`
	ProjectGroupID uuid.UUID  `db:"project_group_id"`
	Content        *string    `db:"content"`
	SubmittedAt    *time.Time `db:"submitted_at"`
	SubmittedBy    *uuid.UUID `db:"submitted_by"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      *time.Time `db:"updated_at"`
}

type ProjectSubmissionFile struct {
	ID           uuid.UUID `db:"id"`
	SubmissionID uuid.UUID `db:"submission_id"`
	InodeID      uuid.UUID `db:"inode_id"`
	DisplayName  string    `db:"display_name"`
	OrderIndex   int       `db:"order_index"`
	CreatedAt    time.Time `db:"created_at"`
}

// ProjectGrade is one member's grade on the group's submission.
type ProjectGrade struct {
	ID           uuid.UUID  `db:"id"`
	SubmissionID uuid.UUID  `db:"submission_id"`
	StudentID    uuid.UUID  `db:"student_id"`
	Score        *float64   `db:"score"`
	Feedback     *string    `db:"feedback"`
	GradedBy     *uuid.UUID `db:"graded_by"`
	GradedAt     *time.Time `db:"graded_at"`
}

// ── Derived read models ─────────────────────────────────────────────────────

// RegistrationWithTeam joins the team's name, leader, and size
// (project_registrations ⋈ teams ⋈ team_members).
type RegistrationWithTeam struct {
	Registration
	TeamName    *string   `db:"team_name"`
	LeaderID    uuid.UUID `db:"leader_id"`
	MemberCount int       `db:"member_count"`
}

// GroupMemberInfo joins member display columns
// (course_group_members ⋈ users).
type GroupMemberInfo struct {
	UserID   uuid.UUID `db:"user_id"`
	Name     string    `db:"name"`
	Username string    `db:"username"`
}

// ProjectGroupWithMembers is the full group view.
type ProjectGroupWithMembers struct {
	ProjectGroup
	Members []GroupMemberInfo
}

// ProjectGradeWithStudent joins the graded member's display columns
// (project_grades ⋈ users).
type ProjectGradeWithStudent struct {
	ProjectGrade
	StudentName     string `db:"student_name"`
	StudentUsername string `db:"student_username"`
}

// ── Rules ───────────────────────────────────────────────────────────────────

// RegistrationOpen reports whether teams may still register.
func RegistrationOpen(deadline *time.Time, now time.Time) bool {
	return deadline == nil || !now.After(*deadline)
}

// TeamSizeFits reports whether a team's size is inside the project bounds.
func TeamSizeFits(size, minMembers, maxMembers int) bool {
	return size >= minMembers && size <= maxMembers
}

// CanViewRegistrations applies the project's visibility to a reader.
func CanViewRegistrations(v ProjectVisibility, isRegistered, isStaff bool) bool {
	switch v {
	case ProjectAll:
		return true
	case ProjectRegistered:
		return isRegistered || isStaff
	case ProjectHidden:
		return isStaff
	}
	return isStaff
}

// MergeSeed is one under-sized registered team entering the merge.
type MergeSeed struct {
	TeamID uuid.UUID
	Size   int
}

// PlanMerge packs under-sized teams into merged groups of [min, max]
// members, aiming at target (0 means min). First-fit-decreasing: larger
// teams seed groups, smaller ones fill toward the target; a leftover group
// below min is folded into whichever earlier group still has room, and
// whatever cannot reach min stays unmerged for the teacher to settle. The
// order of seeds breaks ties, so callers pass a deterministic order.
func PlanMerge(seeds []MergeSeed, minMembers, maxMembers, target int) (groups [][]MergeSeed, unmerged []MergeSeed) {
	if target < minMembers || target > maxMembers {
		target = minMembers
	}
	sorted := make([]MergeSeed, len(seeds))
	copy(sorted, seeds)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Size > sorted[j].Size })

	var sizes []int
	for _, seed := range sorted {
		// Fill the group this seed brings closest to target without
		// bursting max; below-target groups take priority.
		best := -1
		for gi, size := range sizes {
			if size+seed.Size > maxMembers {
				continue
			}
			if best == -1 || betterFill(size+seed.Size, sizes[best]+seed.Size, target) {
				best = gi
			}
		}
		if best >= 0 && sizes[best] < target {
			groups[best] = append(groups[best], seed)
			sizes[best] += seed.Size
			continue
		}
		groups = append(groups, []MergeSeed{seed})
		sizes = append(sizes, seed.Size)
	}

	// Fold still-undersized groups into earlier groups with room; what
	// cannot reach min comes back unmerged.
	result := groups[:0]
	var resultSizes []int
	for gi, group := range groups {
		if sizes[gi] >= minMembers {
			result = append(result, group)
			resultSizes = append(resultSizes, sizes[gi])
			continue
		}
		folded := false
		for ri := range result {
			if resultSizes[ri]+sizes[gi] <= maxMembers {
				result[ri] = append(result[ri], group...)
				resultSizes[ri] += sizes[gi]
				folded = true
				break
			}
		}
		if !folded {
			unmerged = append(unmerged, group...)
		}
	}
	return result, unmerged
}

// betterFill prefers the fill closer to target; among equally distant,
// the smaller (leaving more room elsewhere).
func betterFill(candidate, current, target int) bool {
	dc, dr := absInt(candidate-target), absInt(current-target)
	if dc != dr {
		return dc < dr
	}
	return candidate < current
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// ── Ports ───────────────────────────────────────────────────────────────────

// ProjectRepository persists projects, registrations, groups, submissions,
// and per-member grades. Project gets are offering-scoped.
//
// UpdateProject is a version compare-and-swap. DeleteProject removes the
// whole subtree in one transaction and returns every attachment and
// submission-file inode for unlinking. Register inserts under the
// registration window and the (project, team) unique pair — the window
// guard is in the statement, so a race at the deadline cannot slip in.
// FormGroups freezes every fitting registration into groups atomically and
// reports how many appeared; rerunning skips teams already frozen.
// SaveGroupSubmission creates or replaces the group's draft atomically,
// guarded by "not submitted", returning replaced file inodes.
// SubmitGroupSubmission flips submitted_at while NULL. UpsertGrade writes
// one member's grade keyed on the (submission, student) unique pair.
type ProjectRepository interface {
	CreateProject(ctx context.Context, p *Project) error
	GetProject(ctx context.Context, offeringID, id uuid.UUID) (*Project, error)
	ListProjects(ctx context.Context, offeringID uuid.UUID, publishedOnly bool) ([]Project, error)
	UpdateProject(ctx context.Context, p *Project, expectedVersion int64) (int64, error)
	DeleteProject(ctx context.Context, offeringID, id uuid.UUID) (inodeIDs []uuid.UUID, err error)

	CreateAttachment(ctx context.Context, a *ProjectAttachment) error
	ListAttachments(ctx context.Context, projectID uuid.UUID) ([]ProjectAttachment, error)
	GetAttachmentByName(ctx context.Context, projectID uuid.UUID, displayName string) (*ProjectAttachment, error)
	DeleteAttachment(ctx context.Context, projectID, id uuid.UUID) (inodeID uuid.UUID, err error)

	Register(ctx context.Context, r *Registration, minMembers, maxMembers int) error
	Unregister(ctx context.Context, projectID, teamID uuid.UUID) error
	ListRegistrations(ctx context.Context, projectID uuid.UUID) ([]RegistrationWithTeam, error)
	IsTeamRegistered(ctx context.Context, projectID, teamID uuid.UUID) (bool, error)

	// FormGroups freezes fitting registrations, then merges the under-
	// sized rest per PlanMerge; returns (groups formed, teams unmerged).
	FormGroups(ctx context.Context, p *Project) (formed, unmerged int, err error)
	ListGroups(ctx context.Context, projectID uuid.UUID) ([]ProjectGroupWithMembers, error)
	GetGroup(ctx context.Context, projectID, groupID uuid.UUID) (*ProjectGroup, error)
	GetMemberGroup(ctx context.Context, projectID, userID uuid.UUID) (*ProjectGroupWithMembers, error)

	SaveGroupSubmission(ctx context.Context, sub *ProjectSubmission, files []ProjectSubmissionFile) (replaced []uuid.UUID, err error)
	SubmitGroupSubmission(ctx context.Context, id uuid.UUID, submittedBy uuid.UUID, at time.Time) error
	GetGroupSubmission(ctx context.Context, projectID, groupID uuid.UUID) (*ProjectSubmission, error)
	GetSubmission(ctx context.Context, projectID, id uuid.UUID) (*ProjectSubmission, error)
	ListSubmissions(ctx context.Context, projectID uuid.UUID) ([]ProjectSubmission, error)
	ListSubmissionFiles(ctx context.Context, submissionID uuid.UUID) ([]ProjectSubmissionFile, error)
	GetSubmissionFileByName(ctx context.Context, submissionID uuid.UUID, displayName string) (*ProjectSubmissionFile, error)

	UpsertGrade(ctx context.Context, g *ProjectGrade) error
	ListGrades(ctx context.Context, submissionID uuid.UUID) ([]ProjectGradeWithStudent, error)
	GetMemberGrade(ctx context.Context, submissionID, userID uuid.UUID) (*ProjectGrade, error)
}

// ── Service input types ─────────────────────────────────────────────────────

type CreateProjectInput struct {
	OfferingID           uuid.UUID
	CreatedBy            uuid.UUID
	Title                string
	Body                 *string
	Deadline             time.Time
	MaxScore             float64
	MinMembers           int
	MaxMembers           int
	MergeTarget          *int
	RegistrationDeadline *time.Time
	Visibility           ProjectVisibility
	AllowLate            bool
	PublishAt            *time.Time
}

// UpdateProjectInput is a partial edit; nil leaves a field alone.
type UpdateProjectInput struct {
	Title                *string
	Body                 *string
	Deadline             *time.Time
	MaxScore             *float64
	MinMembers           *int
	MaxMembers           *int
	MergeTarget          *int
	RegistrationDeadline *time.Time
	Visibility           *ProjectVisibility
	AllowLate            *bool
	PublishAt            *time.Time
	ScoresPublic         *bool
}

// ── Service ─────────────────────────────────────────────────────────────────

// ProjectService manages the project lifecycle end to end.
type ProjectService struct {
	repo        ProjectRepository
	teams       TeamRepository
	files       FileStore
	enrollments EnrollmentReader
	notifier    Notifier
	log         *slog.Logger
}

func NewProjectService(repo ProjectRepository, teams TeamRepository, files FileStore, enrollments EnrollmentReader, notifier Notifier, log *slog.Logger) *ProjectService {
	return &ProjectService{repo: repo, teams: teams, files: files, enrollments: enrollments, notifier: notifier, log: log}
}

func (s *ProjectService) Create(ctx context.Context, in CreateProjectInput) (*Project, error) {
	if in.MaxScore <= 0 || in.MinMembers < 1 || in.MinMembers > in.MaxMembers {
		return nil, ErrInvalidInput
	}
	if in.MergeTarget != nil && (*in.MergeTarget < in.MinMembers || *in.MergeTarget > in.MaxMembers) {
		return nil, ErrInvalidInput
	}
	visibility := in.Visibility
	if visibility == "" {
		visibility = ProjectHidden
	}
	if !ValidProjectVisibility(visibility) {
		return nil, ErrInvalidInput
	}
	p := &Project{
		ID:                   uuid.New(),
		OfferingID:           in.OfferingID,
		Title:                in.Title,
		Body:                 in.Body,
		Deadline:             in.Deadline,
		MaxScore:             in.MaxScore,
		MinMembers:           in.MinMembers,
		MaxMembers:           in.MaxMembers,
		MergeTarget:          in.MergeTarget,
		RegistrationDeadline: in.RegistrationDeadline,
		Visibility:           visibility,
		AllowLate:            in.AllowLate,
		PublishAt:            in.PublishAt,
		CreatedBy:            &in.CreatedBy,
		CreatedAt:            time.Now(),
	}
	if err := s.repo.CreateProject(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Get hides unpublished projects from students.
func (s *ProjectService) Get(ctx context.Context, offeringID, id uuid.UUID, forStudent bool) (*Project, []ProjectAttachment, error) {
	p, err := s.repo.GetProject(ctx, offeringID, id)
	if err != nil {
		return nil, nil, err
	}
	if forStudent && !Published(p.PublishAt, time.Now()) {
		return nil, nil, ErrProjectNotFound
	}
	attachments, err := s.repo.ListAttachments(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return p, attachments, nil
}

func (s *ProjectService) List(ctx context.Context, offeringID uuid.UUID, forStudent bool) ([]Project, error) {
	return s.repo.ListProjects(ctx, offeringID, forStudent)
}

func (s *ProjectService) Update(ctx context.Context, offeringID, id uuid.UUID, in UpdateProjectInput) (*Project, error) {
	p, err := s.repo.GetProject(ctx, offeringID, id)
	if err != nil {
		return nil, err
	}
	if err := applyProjectUpdate(p, in); err != nil {
		return nil, err
	}
	newVersion, err := s.repo.UpdateProject(ctx, p, p.Version)
	if err != nil {
		return nil, err
	}
	p.Version = newVersion
	return p, nil
}

func (s *ProjectService) Delete(ctx context.Context, offeringID, id uuid.UUID) error {
	inodeIDs, err := s.repo.DeleteProject(ctx, offeringID, id)
	if err != nil {
		return err
	}
	for _, inodeID := range inodeIDs {
		unlinkLogged(ctx, s.files, s.log, inodeID)
	}
	return nil
}

func (s *ProjectService) Attach(ctx context.Context, offeringID, projectID, actorID uuid.UUID, ref FileRef) (*ProjectAttachment, error) {
	if _, err := s.repo.GetProject(ctx, offeringID, projectID); err != nil {
		return nil, err
	}
	file, err := s.files.ResolveUpload(ctx, actorID, ref.UploadID)
	if err != nil {
		return nil, err
	}
	name := ref.DisplayName
	if name == "" {
		name = file.Name
	}
	if err := s.files.Link(ctx, file.InodeID); err != nil {
		return nil, err
	}
	att := &ProjectAttachment{
		ID:          uuid.New(),
		ProjectID:   projectID,
		InodeID:     file.InodeID,
		DisplayName: name,
		AddedBy:     &actorID,
		CreatedAt:   time.Now(),
	}
	if err := s.repo.CreateAttachment(ctx, att); err != nil {
		unlinkLogged(ctx, s.files, s.log, file.InodeID)
		return nil, err
	}
	return att, nil
}

func (s *ProjectService) Detach(ctx context.Context, offeringID, projectID, attachmentID uuid.UUID) error {
	if _, err := s.repo.GetProject(ctx, offeringID, projectID); err != nil {
		return err
	}
	inodeID, err := s.repo.DeleteAttachment(ctx, projectID, attachmentID)
	if err != nil {
		return err
	}
	unlinkLogged(ctx, s.files, s.log, inodeID)
	return nil
}

// PresignAttachment mints a download URL for a project attachment;
// students reach it only once the project is published.
func (s *ProjectService) PresignAttachment(ctx context.Context, offeringID, projectID uuid.UUID, displayName string, forStudent bool) (string, error) {
	p, err := s.repo.GetProject(ctx, offeringID, projectID)
	if err != nil {
		return "", err
	}
	if forStudent && !Published(p.PublishAt, time.Now()) {
		return "", ErrProjectNotFound
	}
	att, err := s.repo.GetAttachmentByName(ctx, projectID, displayName)
	if err != nil {
		return "", err
	}
	return s.files.Presign(ctx, att.InodeID, att.DisplayName)
}

// Register signs the caller's team up. Only the team leader registers;
// size and enrollment are validated here as UX, the window and duplicate
// guards live in the insert.
func (s *ProjectService) Register(ctx context.Context, offeringID, projectID, teamID, actorID uuid.UUID, projectTitle string) error {
	p, err := s.repo.GetProject(ctx, offeringID, projectID)
	if err != nil {
		return err
	}
	now := time.Now()
	if !Published(p.PublishAt, now) {
		return ErrNotPublished
	}
	if !RegistrationOpen(p.RegistrationDeadline, now) {
		return ErrRegistrationClosed
	}

	team, err := s.teams.GetTeam(ctx, teamID)
	if err != nil {
		return err
	}
	if team.LeaderID != actorID {
		return ErrNotLeader
	}
	members, err := s.teams.TeamMemberIDs(ctx, teamID)
	if err != nil {
		return err
	}
	if !TeamSizeFits(len(members), p.MinMembers, p.MaxMembers) {
		return ErrTeamSizeInvalid
	}
	allEnrolled, err := s.enrollments.AllEnrolled(ctx, offeringID, members)
	if err != nil {
		return err
	}
	if !allEnrolled {
		return ErrMembersNotEnrolled
	}

	return s.repo.Register(ctx, &Registration{
		ID:           uuid.New(),
		ProjectID:    projectID,
		TeamID:       teamID,
		ProjectTitle: projectTitle,
		RegisteredAt: now,
	}, p.MinMembers, p.MaxMembers)
}

// Unregister withdraws the caller's team while registration is open.
func (s *ProjectService) Unregister(ctx context.Context, offeringID, projectID, teamID, actorID uuid.UUID) error {
	p, err := s.repo.GetProject(ctx, offeringID, projectID)
	if err != nil {
		return err
	}
	if !RegistrationOpen(p.RegistrationDeadline, time.Now()) {
		return ErrRegistrationClosed
	}
	team, err := s.teams.GetTeam(ctx, teamID)
	if err != nil {
		return err
	}
	if team.LeaderID != actorID {
		return ErrNotLeader
	}
	return s.repo.Unregister(ctx, projectID, teamID)
}

// Registrations applies the project's visibility to the reader.
func (s *ProjectService) Registrations(ctx context.Context, offeringID, projectID, readerID uuid.UUID, isStaff bool) ([]RegistrationWithTeam, error) {
	p, err := s.repo.GetProject(ctx, offeringID, projectID)
	if err != nil {
		return nil, err
	}
	isRegistered := false
	if !isStaff {
		group, err := s.repo.GetMemberGroup(ctx, projectID, readerID)
		if err == nil && group != nil {
			isRegistered = true
		} else {
			regs, err := s.repo.ListRegistrations(ctx, projectID)
			if err != nil {
				return nil, err
			}
			for _, reg := range regs {
				member, err := s.teams.IsMember(ctx, reg.TeamID, readerID)
				if err != nil {
					return nil, err
				}
				if member {
					isRegistered = true
					break
				}
			}
		}
	}
	if !CanViewRegistrations(p.Visibility, isRegistered, isStaff) {
		return nil, ErrNotAuthorized
	}
	return s.repo.ListRegistrations(ctx, projectID)
}

// FormGroups freezes fitting registrations into graded groups and merges
// the under-sized rest into combined groups; teams that cannot reach the
// minimum even merged are reported back for the teacher to settle.
func (s *ProjectService) FormGroups(ctx context.Context, offeringID, projectID uuid.UUID) (int, int, error) {
	p, err := s.repo.GetProject(ctx, offeringID, projectID)
	if err != nil {
		return 0, 0, err
	}
	return s.repo.FormGroups(ctx, p)
}

func (s *ProjectService) Groups(ctx context.Context, offeringID, projectID uuid.UUID) ([]ProjectGroupWithMembers, error) {
	if _, err := s.repo.GetProject(ctx, offeringID, projectID); err != nil {
		return nil, err
	}
	return s.repo.ListGroups(ctx, projectID)
}

// MyGroup is the caller's frozen group on this project.
func (s *ProjectService) MyGroup(ctx context.Context, offeringID, projectID, userID uuid.UUID) (*ProjectGroupWithMembers, error) {
	if _, err := s.repo.GetProject(ctx, offeringID, projectID); err != nil {
		return nil, err
	}
	group, err := s.repo.GetMemberGroup(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrGroupNotFound
	}
	return group, nil
}

// SaveSubmission creates or replaces the group's draft; group-leader only.
func (s *ProjectService) SaveSubmission(ctx context.Context, offeringID, projectID, actorID uuid.UUID, content *string, refs []FileRef) (*ProjectSubmission, error) {
	if _, err := s.repo.GetProject(ctx, offeringID, projectID); err != nil {
		return nil, err
	}
	group, err := s.repo.GetMemberGroup(ctx, projectID, actorID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrGroupNotFound
	}
	if group.LeaderID != actorID {
		return nil, ErrNotGroupLeader
	}

	files, err := resolveUploads(ctx, s.files, actorID, refs)
	if err != nil {
		return nil, err
	}
	if err := linkAll(ctx, s.files, s.log, files); err != nil {
		return nil, err
	}
	now := time.Now()
	sub := &ProjectSubmission{
		ID:             uuid.New(),
		ProjectID:      projectID,
		ProjectGroupID: group.ID,
		Content:        content,
		CreatedAt:      now,
	}
	subFiles := make([]ProjectSubmissionFile, len(files))
	for i, f := range files {
		subFiles[i] = ProjectSubmissionFile{
			ID:           uuid.New(),
			SubmissionID: sub.ID,
			InodeID:      f.InodeID,
			DisplayName:  f.Name,
			OrderIndex:   i,
			CreatedAt:    now,
		}
	}
	replaced, err := s.repo.SaveGroupSubmission(ctx, sub, subFiles)
	if err != nil {
		for _, f := range files {
			unlinkLogged(ctx, s.files, s.log, f.InodeID)
		}
		return nil, err
	}
	for _, inodeID := range replaced {
		unlinkLogged(ctx, s.files, s.log, inodeID)
	}
	return s.repo.GetGroupSubmission(ctx, projectID, group.ID)
}

// Submit turns the group's draft in; group-leader only.
func (s *ProjectService) Submit(ctx context.Context, offeringID, projectID, actorID uuid.UUID) (*ProjectSubmission, error) {
	p, err := s.repo.GetProject(ctx, offeringID, projectID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if !CanSubmitWork(p.Deadline, p.AllowLate, now) {
		return nil, ErrSubmissionsClosed
	}
	group, err := s.repo.GetMemberGroup(ctx, projectID, actorID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrGroupNotFound
	}
	if group.LeaderID != actorID {
		return nil, ErrNotGroupLeader
	}
	sub, err := s.repo.GetGroupSubmission(ctx, projectID, group.ID)
	if err != nil {
		return nil, err
	}
	files, err := s.repo.ListSubmissionFiles(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	if !HasWork(sub.Content, len(files)) {
		return nil, ErrNoContent
	}
	if err := s.repo.SubmitGroupSubmission(ctx, sub.ID, actorID, now); err != nil {
		return nil, err
	}
	return s.repo.GetGroupSubmission(ctx, projectID, group.ID)
}

// MySubmission is the caller's group's submission with files.
func (s *ProjectService) MySubmission(ctx context.Context, offeringID, projectID, userID uuid.UUID) (*ProjectSubmission, []ProjectSubmissionFile, error) {
	if _, err := s.repo.GetProject(ctx, offeringID, projectID); err != nil {
		return nil, nil, err
	}
	group, err := s.repo.GetMemberGroup(ctx, projectID, userID)
	if err != nil {
		return nil, nil, err
	}
	if group == nil {
		return nil, nil, ErrGroupNotFound
	}
	sub, err := s.repo.GetGroupSubmission(ctx, projectID, group.ID)
	if err != nil {
		return nil, nil, err
	}
	files, err := s.repo.ListSubmissionFiles(ctx, sub.ID)
	if err != nil {
		return nil, nil, err
	}
	return sub, files, nil
}

// Submissions is the teacher's list.
func (s *ProjectService) Submissions(ctx context.Context, offeringID, projectID uuid.UUID) ([]ProjectSubmission, error) {
	if _, err := s.repo.GetProject(ctx, offeringID, projectID); err != nil {
		return nil, err
	}
	return s.repo.ListSubmissions(ctx, projectID)
}

// Grade writes one member's grade on a submitted work and tells them.
func (s *ProjectService) Grade(ctx context.Context, offeringID, projectID, submissionID, studentID, graderID uuid.UUID, score float64, feedback *string) error {
	p, err := s.repo.GetProject(ctx, offeringID, projectID)
	if err != nil {
		return err
	}
	if !ValidScore(score, p.MaxScore) {
		return ErrInvalidScore
	}
	sub, err := s.repo.GetSubmission(ctx, projectID, submissionID)
	if err != nil {
		return err
	}
	if sub.SubmittedAt == nil {
		return ErrSubmissionNotFound
	}
	now := time.Now()
	if err := s.repo.UpsertGrade(ctx, &ProjectGrade{
		ID:           uuid.New(),
		SubmissionID: submissionID,
		StudentID:    studentID,
		Score:        &score,
		Feedback:     feedback,
		GradedBy:     &graderID,
		GradedAt:     &now,
	}); err != nil {
		return err
	}
	body := "Your project \"" + p.Title + "\" has been graded."
	notify(ctx, s.notifier, s.log, studentID, "project_graded", "Project graded", &body, map[string]any{
		"project_id": p.ID, "submission_id": submissionID,
	})
	return nil
}

// Grades is the teacher's per-member grade list for a submission.
func (s *ProjectService) Grades(ctx context.Context, offeringID, projectID, submissionID uuid.UUID) ([]ProjectGradeWithStudent, error) {
	if _, err := s.repo.GetProject(ctx, offeringID, projectID); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetSubmission(ctx, projectID, submissionID); err != nil {
		return nil, err
	}
	return s.repo.ListGrades(ctx, submissionID)
}

// MyGrade is the caller's own grade, visible once scores are public.
func (s *ProjectService) MyGrade(ctx context.Context, offeringID, projectID, userID uuid.UUID) (*ProjectGrade, error) {
	p, err := s.repo.GetProject(ctx, offeringID, projectID)
	if err != nil {
		return nil, err
	}
	group, err := s.repo.GetMemberGroup(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrGroupNotFound
	}
	sub, err := s.repo.GetGroupSubmission(ctx, projectID, group.ID)
	if err != nil {
		return nil, err
	}
	if !p.ScoresPublic {
		return nil, ErrNotAuthorized
	}
	grade, err := s.repo.GetMemberGrade(ctx, sub.ID, userID)
	if err != nil {
		return nil, err
	}
	return grade, nil
}

// PresignSubmissionFile mints a download URL for one submission file;
// group members and staff only — the edge routes accordingly.
func (s *ProjectService) PresignSubmissionFile(ctx context.Context, offeringID, projectID, submissionID uuid.UUID, displayName string) (string, error) {
	if _, err := s.repo.GetProject(ctx, offeringID, projectID); err != nil {
		return "", err
	}
	if _, err := s.repo.GetSubmission(ctx, projectID, submissionID); err != nil {
		return "", err
	}
	file, err := s.repo.GetSubmissionFileByName(ctx, submissionID, displayName)
	if err != nil {
		return "", err
	}
	return s.files.Presign(ctx, file.InodeID, file.DisplayName)
}

func applyProjectUpdate(p *Project, in UpdateProjectInput) error {
	if in.Title != nil {
		p.Title = *in.Title
	}
	if in.Body != nil {
		p.Body = in.Body
	}
	if in.Deadline != nil {
		p.Deadline = *in.Deadline
	}
	if in.MaxScore != nil {
		if *in.MaxScore <= 0 {
			return ErrInvalidInput
		}
		p.MaxScore = *in.MaxScore
	}
	if in.MinMembers != nil {
		p.MinMembers = *in.MinMembers
	}
	if in.MaxMembers != nil {
		p.MaxMembers = *in.MaxMembers
	}
	if in.MergeTarget != nil {
		p.MergeTarget = in.MergeTarget
	}
	if p.MinMembers < 1 || p.MinMembers > p.MaxMembers {
		return ErrInvalidInput
	}
	if p.MergeTarget != nil && (*p.MergeTarget < p.MinMembers || *p.MergeTarget > p.MaxMembers) {
		return ErrInvalidInput
	}
	if in.RegistrationDeadline != nil {
		p.RegistrationDeadline = in.RegistrationDeadline
	}
	if in.Visibility != nil {
		if !ValidProjectVisibility(*in.Visibility) {
			return ErrInvalidInput
		}
		p.Visibility = *in.Visibility
	}
	if in.AllowLate != nil {
		p.AllowLate = *in.AllowLate
	}
	if in.PublishAt != nil {
		p.PublishAt = in.PublishAt
	}
	if in.ScoresPublic != nil {
		p.ScoresPublic = *in.ScoresPublic
	}
	return nil
}
