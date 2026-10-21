package models

const (
	BatchRowReady         = "ready"
	BatchRowMissingName   = "missing_name"
	BatchRowBadEmail      = "bad_email"
	BatchRowUnknownGroup  = "unknown_group"
	BatchRowAlreadyMember = "already_member"
	BatchRowDuplicate     = "duplicate"
)

type InvitationBatchRow struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	GroupCode string `json:"group_code"`
}

type InvitationBatchCheck struct {
	Row     int                `json:"row"`
	Input   InvitationBatchRow `json:"input"`
	Status  string             `json:"status"`
	GroupID string             `json:"group_id,omitempty"`
	ActorID string             `json:"actor_id,omitempty"`
}
