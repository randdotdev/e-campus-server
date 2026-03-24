package team

import "errors"

var (
	ErrTeamNotFound       = errors.New("team not found")
	ErrNotLeader          = errors.New("only leader can perform this action")
	ErrNotMember          = errors.New("user is not a team member")
	ErrAlreadyMember      = errors.New("user is already a member")
	ErrTeamLocked         = errors.New("team has submissions and cannot be modified")
	ErrTeamArchived       = errors.New("team is archived")
	ErrLeaderCannotLeave  = errors.New("leader must transfer leadership first")
	ErrCannotRemoveLeader = errors.New("cannot remove leader, transfer leadership first")
	ErrMaxMembers         = errors.New("team has maximum members")
	ErrMemberNotFound     = errors.New("member not found")
	ErrInvalidStatus      = errors.New("invalid team status")
)
