package authz

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
)

func postService(facts PostFacts, relations *fakeRelations, lineage *fakeLineage) *Service {
	if relations == nil {
		relations = &fakeRelations{}
	}
	if lineage == nil {
		lineage = &fakeLineage{}
	}
	policies := map[PolicyKey]Policy{
		{ResourcePost, ActionGet}:    {Staff: postStaff, Offering: teachingSeats},
		{ResourcePost, ActionUpdate}: {Staff: postStaff, Offering: teachingSeats},
		{ResourcePost, ActionPin}:    {Staff: postStaff, Offering: teachingSeats},
	}
	return NewService(&fakePolicies{policies: policies},
		fakeReaders{lineage, relations, &fakePosts{facts: facts}}, slog.Default())
}

func TestCheckPostOwnerArm(t *testing.T) {
	author := uuid.New()
	s := postService(PostFacts{AuthorID: author, Scope: PostScopeUniversity}, nil, nil)

	dec, err := s.CheckPost(context.Background(), Actor{ID: author},
		PolicyKey{ResourcePost, ActionUpdate}, uuid.New())
	if err != nil || !dec.Allowed {
		t.Fatalf("author must edit own post: Allowed=%v err=%v", dec.Allowed, err)
	}
	if dec.Matched == nil || dec.Matched.Type != TypeOwner {
		t.Fatalf("owner arm must be recorded, got %v", dec.Matched)
	}
}

func TestCheckPostScopeAuthority(t *testing.T) {
	deptID := uuid.New()
	otherDept := uuid.New()
	postID := uuid.New()
	facts := PostFacts{AuthorID: uuid.New(), Scope: PostScopeDepartment, ScopeID: &deptID}

	// Same-unit department admin moderates; another department's admin only reads.
	own := postService(facts, nil, &fakeLineage{lineage: Lineage{Department: &deptID}})
	dec, err := own.CheckPost(context.Background(), staffActor(LevelAdmin, ScopeDepartment, &deptID),
		PolicyKey{ResourcePost, ActionUpdate}, postID)
	if err != nil || !dec.Allowed {
		t.Fatalf("same-unit admin must moderate: Allowed=%v err=%v", dec.Allowed, err)
	}

	foreign := postService(facts, nil, &fakeLineage{lineage: Lineage{Department: &deptID}})
	dec, err = foreign.CheckPost(context.Background(), staffActor(LevelAdmin, ScopeDepartment, &otherDept),
		PolicyKey{ResourcePost, ActionUpdate}, postID)
	if err != nil || dec.Allowed {
		t.Fatalf("foreign-unit admin must not moderate: Allowed=%v err=%v", dec.Allowed, err)
	}
}

func TestCheckPostUniversityScopeIgnoresNarrowAdmins(t *testing.T) {
	programID := uuid.New()
	s := postService(PostFacts{AuthorID: uuid.New(), Scope: PostScopeUniversity}, nil, nil)

	// The pin policy admits program admins for program posts; a university
	// post must not fall to them through nil-target collection semantics.
	dec, err := s.CheckPost(context.Background(), staffActor(LevelAdmin, ScopeProgram, &programID),
		PolicyKey{ResourcePost, ActionPin}, uuid.New())
	if err != nil || dec.Allowed {
		t.Fatalf("program admin must not pin university posts: Allowed=%v err=%v", dec.Allowed, err)
	}

	dec, err = s.CheckPost(context.Background(), staffActor(LevelAdmin, ScopeUniversity, nil),
		PolicyKey{ResourcePost, ActionPin}, uuid.New())
	if err != nil || !dec.Allowed {
		t.Fatalf("university admin must pin: Allowed=%v err=%v", dec.Allowed, err)
	}
}

func TestCheckPostMemberRead(t *testing.T) {
	s := postService(PostFacts{AuthorID: uuid.New(), Scope: PostScopeUniversity}, nil, nil)

	// Any signed-in member reads staff-scoped feeds — allowed, no authority.
	dec, err := s.CheckPost(context.Background(), Actor{ID: uuid.New()},
		PolicyKey{ResourcePost, ActionGet}, uuid.New())
	if err != nil || !dec.Allowed {
		t.Fatalf("member must read: Allowed=%v err=%v", dec.Allowed, err)
	}
	if dec.Matched != nil {
		t.Fatalf("member read must carry no authority, got %v", dec.Matched)
	}
	// But may not moderate.
	dec, err = s.CheckPost(context.Background(), Actor{ID: uuid.New()},
		PolicyKey{ResourcePost, ActionUpdate}, uuid.New())
	if err != nil || dec.Allowed {
		t.Fatalf("member must not moderate: Allowed=%v err=%v", dec.Allowed, err)
	}
}

func TestCheckPostCourseScope(t *testing.T) {
	offeringID := uuid.New()
	facts := PostFacts{AuthorID: uuid.New(), Scope: PostScopeOffering, ScopeID: &offeringID}

	// A teaching seat has authority; a student seat reads without it;
	// no seat sees nothing.
	teacher := postService(facts, &fakeRelations{relation: OfferingRoleTeacher}, nil)
	dec, _ := teacher.CheckPost(context.Background(), Actor{ID: uuid.New()},
		PolicyKey{ResourcePost, ActionUpdate}, uuid.New())
	if !dec.Allowed || dec.Matched == nil {
		t.Fatalf("teaching seat must moderate, got %+v", dec)
	}

	student := postService(facts, &fakeRelations{relation: OfferingRoleStudent}, nil)
	dec, _ = student.CheckPost(context.Background(), Actor{ID: uuid.New()},
		PolicyKey{ResourcePost, ActionGet}, uuid.New())
	if !dec.Allowed || dec.Matched != nil {
		t.Fatalf("student seat must read without authority, got %+v", dec)
	}
	dec, _ = student.CheckPost(context.Background(), Actor{ID: uuid.New()},
		PolicyKey{ResourcePost, ActionUpdate}, uuid.New())
	if dec.Allowed {
		t.Fatal("student seat must not moderate")
	}

	outsider := postService(facts, &fakeRelations{relation: RelationNone}, nil)
	dec, _ = outsider.CheckPost(context.Background(), Actor{ID: uuid.New()},
		PolicyKey{ResourcePost, ActionGet}, uuid.New())
	if dec.Allowed {
		t.Fatal("a class feed is not readable outside the class")
	}
}

func TestCheckPostMissingPost(t *testing.T) {
	s := NewService(&fakePolicies{}, fakeReaders{&fakeLineage{}, &fakeRelations{},
		&fakePosts{err: ErrTargetNotFound}}, slog.Default())
	_, err := s.CheckPost(context.Background(), Actor{ID: uuid.New()},
		PolicyKey{ResourcePost, ActionGet}, uuid.New())
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("want ErrTargetNotFound, got %v", err)
	}
}
