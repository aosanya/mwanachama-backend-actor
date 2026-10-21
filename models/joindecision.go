package models

type JoinDecision struct {
	ID        string `json:"id"`
	RequestID string `json:"request_id"`
	GroupID   string `json:"group_id"`
	ActorID   string `json:"actor_id"`
	State     string `json:"state"`
	Silent    bool   `json:"silent"`
	DecidedBy string `json:"decided_by,omitempty"`
	DecidedAt string `json:"decided_at"`
	Reason    string `json:"reason,omitempty"`
}
