package models

type ActorRoleAssignment struct {
	ID          string    `json:"id"`
	ActorID     string    `json:"actor_id"`
	GroupID     string    `json:"group_id"`
	KindID      string    `json:"kind_id"`
	Active      bool      `json:"active"`
	GrantedAt   string    `json:"granted_at"`
	GrantedBy   string    `json:"granted_by,omitempty"`
	EndedAt     string    `json:"ended_at,omitempty"`
	EndedReason EndReason `json:"ended_reason,omitempty"`
}
