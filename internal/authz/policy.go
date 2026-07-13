package authz

import (
	"context"

	"github.com/google/uuid"
)

// A policy is the answer to one question: who may perform this action on
// this kind of resource? It is keyed (resource, action) — never a URL, so
// renaming routes cannot change who may do what — and its permissions OR
// together; no entry, no match, or no policy denies. Policies live as
// compiled-in defaults (policy_defaults.go) and, in DB mode, as the
// admin-tunable rows boot seeds from them.

// Entity names a kind of resource a policy can govern. Note the distinction
// the old system blurred: a course is a catalogue entry; an offering is that
// course taught in one semester, and it is where classrooms live.
type Entity string

const (
	ResourceCourse       Entity = "course"
	ResourceOffering     Entity = "offering"
	ResourceStudent      Entity = "student"
	ResourceExam         Entity = "exam"
	ResourceAssignment   Entity = "assignment"
	ResourceAcademicYear Entity = "academic_year"
	ResourceSemester     Entity = "semester"
	ResourceEnrollment   Entity = "enrollment"
	ResourceGrade        Entity = "grade"
	ResourceAttendance   Entity = "attendance"
	ResourceUser         Entity = "user"
	ResourceDepartment   Entity = "department"
	ResourceCollege      Entity = "college"
	ResourceProgram      Entity = "program"
	ResourceUniversity   Entity = "university"
	ResourcePlatform     Entity = "platform"
	ResourceActivity     Entity = "activity"
	ResourcePost         Entity = "post"
	ResourceQA           Entity = "qa"
	ResourceSettings     Entity = "settings"
	ResourceApplication  Entity = "application"
	ResourceCohortGroup  Entity = "cohort_group"
	ResourceSubscription Entity = "subscription"
	ResourceCurriculum   Entity = "curriculum"
	ResourceProject      Entity = "project"
	// ResourceContent is the course material tree: sections, lessons,
	// their attachments and schedules.
	ResourceContent Entity = "content"
	// ResourceMute is communication's moderation over an offering's
	// participants — silencing a member's posts. (Amended 2026-07-08: added
	// when the mute routes converted onto the gates; muting in an offering is
	// a teaching-seat capability the old staff-only inline check could not
	// express. University-wide mutes stay on ResourceUser.)
	ResourceMute Entity = "mute"
	// ResourceTeacher is an offering's teaching staff assignment
	// (course_teachers): who teaches, in what role. Scoped through the
	// offering it staffs.
	ResourceTeacher Entity = "teacher"
)

// Action names what is being done. The standard five derive from the HTTP
// verb; custom methods ride the URL's colon suffix
// (POST /students/:id:activate → "activate").
type Action string

const (
	ActionGet    Action = "get"
	ActionList   Action = "list"
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionSubmit Action = "submit"

	// Classroom custom methods (2026-07-06). Each is a distinct operation
	// with its own audience, so each is its own policy row — the verbs the
	// standard five cannot say. Student-side: start/save/submit an attempt
	// or a draft, discard a draft, register/unregister a team, request an
	// excuse. Teacher-side: attach material, schedule a cohort group,
	// publish/close an exam, record manual results, grade and review work,
	// answer/reject a question, form project groups, initialize and mark
	// attendance.
	ActionAttach     Action = "attach"
	ActionSchedule   Action = "schedule"
	ActionUnschedule Action = "unschedule"
	ActionSave       Action = "save"
	ActionDiscard    Action = "discard"
	ActionGrade      Action = "grade"
	ActionPublish    Action = "publish"
	ActionClose      Action = "close"
	ActionStart      Action = "start"
	ActionRecord     Action = "record"
	ActionReview     Action = "review"
	ActionAnswer     Action = "answer"
	ActionReject     Action = "reject"
	ActionRegister   Action = "register"
	ActionUnregister Action = "unregister"
	ActionFormGroups Action = "formGroups"
	ActionInitialize Action = "initialize"
	ActionMark       Action = "mark"
	ActionExcuse     Action = "excuse"

	// Management custom methods (2026-07-11: the management gate
	// conversion). Semester lifecycle: activate the term, open grading,
	// finalize grades, definalize grades, generate the term's offerings,
	// bulk-enroll cohorts, end the term. Admissions and
	// enrollment requests: review decides an application, approve/reject
	// decide a request. Student leaves: requestLeave opens one; approve
	// and end close its two transitions (reject reused from the shared
	// vocabulary).
	ActionActivate          Action = "activate"
	ActionStartGrading      Action = "startGrading"
	ActionFinalize          Action = "finalize"
	ActionDefinalize        Action = "definalize"
	ActionGenerateOfferings Action = "generateOfferings"
	ActionBulkEnroll        Action = "bulkEnroll"
	ActionEnd               Action = "end"
	ActionApprove           Action = "approve"
	ActionRequestLeave      Action = "requestLeave"

	// ActionPin fixes an announcements post to the top of its feed.
	ActionPin Action = "pin"
)

// PolicyKey is what policies are looked up by, and the unit seeding reasons
// about: one (resource, action) pair.
type PolicyKey struct {
	Resource Entity
	Action   Action
}

// PermissionType tells the permission shapes apart. Staff and offering
// are the two stored shapes; TypeOwner appears only in Decisions — the
// row's own author acted on it, a structural fact, never a policy row.
type PermissionType string

const (
	TypeStaff    PermissionType = "staff"
	TypeOffering PermissionType = "offering"
	TypeOwner    PermissionType = "owner"
)

// StaffPermission is a region in the staff-authority space: at least
// MinLevel, at a scope that reaches Scope, and — when Domain is set —
// within that functional domain.
type StaffPermission struct {
	MinLevel Level
	Scope    Scope
	Domain   Domain // empty = any domain
}

// Policy is everything permitted for one PolicyKey: the staff regions and
// the offering seats, OR'd together. An empty Policy denies everyone —
// deny by default.
type Policy struct {
	Staff    []StaffPermission
	Offering []OfferingRole
}

// Permission is one stored policy row — the unit administration edits.
// The course_role column keeps its legacy name until classroom's own
// migration renames the underlying tables.
type Permission struct {
	ID           uuid.UUID      `db:"id"`
	Resource     Entity         `db:"resource"`
	Action       Action         `db:"verb"`
	Type         PermissionType `db:"type"`
	MinLevel     *Level         `db:"min_level"`
	Scope        *Scope         `db:"scope_type"`
	Domain       *Domain        `db:"domain"`
	OfferingRole *OfferingRole  `db:"course_role"`
	Active       bool           `db:"is_active"`
}

func ValidEntity(e Entity) bool { return validEntities[e] }

func ValidAction(a Action) bool { return validActions[a] }

// ValidPermission reports whether the input is a well-shaped permission of
// its type: staff permissions carry level+scope and no seat, offering
// permissions carry a seat and nothing else, and every value is in its
// vocabulary.
func ValidPermission(in PermissionInput) bool {
	if !ValidEntity(in.Resource) || !ValidAction(in.Action) {
		return false
	}
	switch in.Type {
	case TypeStaff:
		return in.OfferingRole == "" && ValidLevel(in.MinLevel) && ValidScope(in.Scope) &&
			(in.Domain == "" || ValidDomain(in.Domain))
	case TypeOffering:
		return ValidOfferingRole(in.OfferingRole) && in.MinLevel == "" && in.Scope == "" && in.Domain == ""
	default:
		// Owner permissions are implicit in the engine, never policy rows.
		return false
	}
}

var validEntities = map[Entity]bool{
	ResourceCourse: true, ResourceOffering: true, ResourceStudent: true,
	ResourceExam: true, ResourceAssignment: true, ResourceAcademicYear: true,
	ResourceSemester: true, ResourceEnrollment: true, ResourceGrade: true,
	ResourceAttendance: true, ResourceUser: true, ResourceDepartment: true,
	ResourceCollege: true, ResourceProgram: true, ResourceUniversity: true,
	ResourcePlatform: true, ResourceActivity: true, ResourcePost: true,
	ResourceQA: true, ResourceSettings: true, ResourceApplication: true,
	ResourceCohortGroup: true, ResourceSubscription: true,
	ResourceCurriculum: true, ResourceProject: true, ResourceContent: true,
	ResourceMute: true, ResourceTeacher: true,
}

var validActions = map[Action]bool{
	ActionGet: true, ActionList: true, ActionCreate: true,
	ActionUpdate: true, ActionDelete: true, ActionSubmit: true,
	ActionAttach: true, ActionSchedule: true, ActionUnschedule: true,
	ActionSave: true, ActionDiscard: true, ActionGrade: true,
	ActionPublish: true, ActionClose: true, ActionStart: true,
	ActionRecord: true, ActionReview: true, ActionAnswer: true,
	ActionReject: true, ActionRegister: true, ActionUnregister: true,
	ActionFormGroups: true, ActionInitialize: true, ActionMark: true,
	ActionExcuse:   true,
	ActionActivate: true, ActionStartGrading: true, ActionFinalize: true,
	ActionDefinalize: true, ActionGenerateOfferings: true, ActionBulkEnroll: true,
	ActionEnd: true, ActionApprove: true, ActionRequestLeave: true,
	ActionPin: true,
}

// PolicyStore is the persistence port for policies. PolicyFor returns
// only active permissions (none = empty Policy = deny). Seed installs
// defaults for pairs with no rows at all — admin edits survive boots —
// and must be safe under concurrent start-up. Reset re-flashes wholesale.
type PolicyStore interface {
	PolicyFor(ctx context.Context, key PolicyKey) (Policy, error)
	ListPermissions(ctx context.Context) ([]Permission, error)
	CreatePermission(ctx context.Context, in PermissionInput) (*Permission, error)
	DeactivatePermission(ctx context.Context, id uuid.UUID) error
	Seed(ctx context.Context) error
	Reset(ctx context.Context) error
}

// PermissionInput describes a permission to create. Type decides which
// fields apply.
type PermissionInput struct {
	Resource     Entity
	Action       Action
	Type         PermissionType
	MinLevel     Level        // staff type
	Scope        Scope        // staff type
	Domain       Domain       // staff type, optional
	OfferingRole OfferingRole // offering type
}

// ListPermissions returns every stored permission, active and inactive,
// for the administration UI.
func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	return s.policies.ListPermissions(ctx)
}

// CreatePermission adds one permission. Duplicates are decided by the
// unique index on the permission tuple (ErrPermissionExists).
func (s *Service) CreatePermission(ctx context.Context, in PermissionInput) (*Permission, error) {
	if !ValidPermission(in) {
		return nil, ErrInvalidPermission
	}
	return s.policies.CreatePermission(ctx, in)
}

// DeactivatePermission soft-deletes one permission. Administration never
// hard-deletes: a pair whose rows are all inactive means "nobody may do
// this", and seeding respects that decision across boots.
func (s *Service) DeactivatePermission(ctx context.Context, id uuid.UUID) error {
	return s.policies.DeactivatePermission(ctx, id)
}

// SeedPolicies installs the defaults for pairs the table has never heard
// of. Run at boot, before serving; idempotent, concurrent-boot safe.
func (s *Service) SeedPolicies(ctx context.Context) error {
	return s.policies.Seed(ctx)
}

// ResetPolicies discards every stored permission and reinstalls the
// defaults — the recovery path when policy edits have gone wrong.
func (s *Service) ResetPolicies(ctx context.Context, actorID uuid.UUID) error {
	if err := s.policies.Reset(ctx); err != nil {
		return err
	}
	s.log.InfoContext(ctx, "authz policies reset to defaults", "audit", true, "actor_id", actorID)
	return nil
}
