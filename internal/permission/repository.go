package permission

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetTeacherRole(ctx context.Context, offeringID, userID uuid.UUID) (string, error) {
	var role string
	query := `SELECT role FROM course_teachers WHERE offering_id = $1 AND user_id = $2`
	if err := r.db.GetContext(ctx, &role, query, offeringID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return role, nil
}

func (r *Repository) IsEnrolled(ctx context.Context, offeringID, userID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM course_enrollments WHERE offering_id = $1 AND student_id = $2 AND status = 'enrolled')`
	if err := r.db.GetContext(ctx, &exists, query, offeringID, userID); err != nil {
		return false, err
	}
	return exists, nil
}
