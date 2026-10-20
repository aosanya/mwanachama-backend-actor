package models

const (
	InvitationPending  = "pending"
	InvitationAccepted = "accepted"
	InvitationDeclined = "declined"
)

type GroupInvitation struct {
	ID          string `json:"id"`
	GroupID     string `json:"group_id"`
	ActorID     string `json:"actor_id"`
	State       string `json:"state"`
	InvitedBy   string `json:"invited_by"`
	InvitedAt   string `json:"invited_at"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	AnsweredAt  string `json:"answered_at,omitempty"`
	LastUpdated string `json:"last_updated"`
}

func (i GroupInvitation) Lapsed(now string) bool {
	return i.State == InvitationPending && i.ExpiresAt != "" && i.ExpiresAt <= now
}

func (i GroupInvitation) Open(now string) bool {
	return i.State == InvitationPending && !i.Lapsed(now)
}
