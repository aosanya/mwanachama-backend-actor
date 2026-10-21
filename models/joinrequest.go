package models

const (
	JoinRequestPending  = "pending"
	JoinRequestApproved = "approved"
	JoinRequestDeclined = "declined"
)

const MaxJoinMessageLength = 500

const MaxDeclineReasonLength = 500

const (
	JoinedNow      = "joined"
	JoinRequested  = "requested"
	AlreadyAMember = "member"
)

type GroupJoinRequest struct {
	ID            string `json:"id"`
	GroupID       string `json:"group_id"`
	ActorID       string `json:"actor_id"`
	State         string `json:"state"`
	Appeal        bool   `json:"appeal"`
	Message       string `json:"message"`
	Silent        bool   `json:"silent"`
	RequestedAt   string `json:"requested_at"`
	AskedAt       string `json:"asked_at"`
	DecidedBy     string `json:"decided_by,omitempty"`
	DecidedAt     string `json:"decided_at,omitempty"`
	DeclineReason string `json:"decline_reason,omitempty"`
	LastUpdated   string `json:"last_updated"`
}

func (r GroupJoinRequest) Pending() bool { return r.State == JoinRequestPending }

type JoinOutcome struct {
	Status     string                `json:"status"`
	Request    *GroupJoinRequest     `json:"request,omitempty"`
	Assignment *ActorGroupAssignment `json:"assignment,omitempty"`
}
