package team

import "github.com/google/uuid"

func IsLeader(team *Team, userID uuid.UUID) bool {
	return team.LeaderID == userID
}

func IsActive(status string) bool {
	return status == StatusActive
}

func IsValidStatus(status string) bool {
	return status == StatusActive || status == StatusArchived
}

func GetDefaultTeamName(leaderName string) string {
	return leaderName + "'s Team"
}

func CanModifyMembers(status string, hasSubmissions bool) bool {
	return status == StatusActive && !hasSubmissions
}

func IsValidMemberCount(count int) bool {
	return count >= 1 && count <= MaxMembers
}
