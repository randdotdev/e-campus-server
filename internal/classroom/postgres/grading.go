package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/classroom"
)

// GradingRepository is the SQL adapter for the rule set and the
// assignment-average read.
type GradingRepository struct {
	db *sqlx.DB
}

func NewGradingRepository(db *sqlx.DB) *GradingRepository {
	return &GradingRepository{db: db}
}

var _ classroom.GradingRepository = (*GradingRepository)(nil)

func (r *GradingRepository) SaveRules(ctx context.Context, gr *classroom.GradingRules) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO grading_rules (offering_id, rules, created_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (offering_id)
		DO UPDATE SET rules = EXCLUDED.rules, updated_at = NOW()`,
		gr.OfferingID, gr.Rules, gr.CreatedBy)
	return err
}

func (r *GradingRepository) GetRules(ctx context.Context, offeringID uuid.UUID) (*classroom.GradingRules, error) {
	var gr classroom.GradingRules
	err := r.db.GetContext(ctx, &gr, `
		SELECT offering_id, rules, created_by, created_at, updated_at
		FROM grading_rules WHERE offering_id = $1`, offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrRulesNotFound
	}
	if err != nil {
		return nil, err
	}
	return &gr, nil
}

func (r *GradingRepository) StudentAssignmentAverage(ctx context.Context, offeringID, studentID uuid.UUID) (float64, bool, error) {
	var result struct {
		Avg *float64 `db:"avg"`
	}
	err := r.db.GetContext(ctx, &result, `
		SELECT AVG(asub.score / a.max_score * 100) AS avg
		FROM assignments a
		JOIN assignment_submissions asub
			ON asub.assignment_id = a.id AND asub.student_id = $2 AND asub.score IS NOT NULL
		WHERE a.offering_id = $1`, offeringID, studentID)
	if err != nil {
		return 0, false, err
	}
	if result.Avg == nil {
		return 0, false, nil
	}
	return *result.Avg, true, nil
}

func (r *GradingRepository) HasUngradedSubmissions(ctx context.Context, offeringID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `
		SELECT EXISTS(
			SELECT 1 FROM assignment_submissions asub
			JOIN assignments a ON a.id = asub.assignment_id
			WHERE a.offering_id = $1 AND asub.submitted_at IS NOT NULL AND asub.score IS NULL
		)`, offeringID)
	return exists, err
}
