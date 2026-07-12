package management

import (
	"testing"

	"github.com/google/uuid"
)

func TestPickCohortGroups(t *testing.T) {
	group := func(gt CohortGroupType, count int) CohortGroupWithCount {
		return CohortGroupWithCount{
			CohortGroup: CohortGroup{ID: uuid.New(), Type: gt},
			MemberCount: count,
		}
	}

	t.Run("picks least populated per type", func(t *testing.T) {
		groups := []CohortGroupWithCount{
			group(CohortGroupTheory, 30),
			group(CohortGroupTheory, 12),
			group(CohortGroupPractice, 8),
			group(CohortGroupPractice, 25),
		}
		theory, practice := PickCohortGroups(groups)
		if theory == nil || theory.MemberCount != 12 {
			t.Errorf("expected theory group with 12 members, got %+v", theory)
		}
		if practice == nil || practice.MemberCount != 8 {
			t.Errorf("expected practice group with 8 members, got %+v", practice)
		}
	})

	t.Run("missing type yields nil", func(t *testing.T) {
		theory, practice := PickCohortGroups([]CohortGroupWithCount{group(CohortGroupTheory, 5)})
		if theory == nil {
			t.Error("expected a theory group")
		}
		if practice != nil {
			t.Errorf("expected no practice group, got %+v", practice)
		}
	})

	t.Run("empty input yields nils", func(t *testing.T) {
		theory, practice := PickCohortGroups(nil)
		if theory != nil || practice != nil {
			t.Errorf("expected nils, got %+v / %+v", theory, practice)
		}
	})
}

func TestValidCohortGroupType(t *testing.T) {
	if !ValidCohortGroupType(CohortGroupTheory) || !ValidCohortGroupType(CohortGroupPractice) {
		t.Error("theory and practice must be valid cohort group types")
	}
	for _, invalid := range []CohortGroupType{"", "lab", "THEORY"} {
		if ValidCohortGroupType(invalid) {
			t.Errorf("ValidCohortGroupType(%q) = true, want false", invalid)
		}
	}
}
