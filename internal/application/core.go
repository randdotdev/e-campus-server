package application

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

func isOwner(appUserID *uuid.UUID, userID uuid.UUID) bool {
	return appUserID != nil && *appUserID == userID
}

func canUpdate(status string) bool {
	return status == StatusNeedsRevision
}

func canWithdraw(status string) bool {
	return status == StatusPending || status == StatusNeedsRevision
}

func canReview(status string) bool {
	return status == StatusPending
}

func isValidReviewStatus(status string) bool {
	return status == StatusApproved || status == StatusRejected || status == StatusNeedsRevision
}

func marshalJSONB(data any, defaultVal []byte) ([]byte, error) {
	if data == nil {
		return defaultVal, nil
	}
	return json.Marshal(data)
}

func calculateAge(dateOfBirth string) (int, error) {
	dob, err := time.Parse("2006-01-02", dateOfBirth)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	age := now.Year() - dob.Year()

	if now.YearDay() < dob.YearDay() {
		age--
	}

	return age, nil
}
