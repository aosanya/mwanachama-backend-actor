package models

type RoleKind struct {
	ID               string   `json:"id"`
	Code             string   `json:"code"`
	Name             string   `json:"name"`
	Capabilities     []string `json:"capabilities"`
	Description      string   `json:"description,omitempty"`
	IsDelegate       bool     `json:"is_delegate,omitempty"`
	LeadsGroup       bool     `json:"leads_group,omitempty"`
	LevelID          string   `json:"level_id,omitempty"`
	DefinitionRoleID string   `json:"definition_role_id,omitempty"`
	RetiredAt        string   `json:"retired_at,omitempty"`
	RetiredBy        string   `json:"retired_by,omitempty"`
}

func (k RoleKind) Retired() bool { return k.RetiredAt != "" }

// EndReason records which of the four acts ended an ActorRoleAssignment —
// mirrors mwanachama-backend-api-gateway's internal/domain/role.EndReason.
// Revoke and StepDown deactivate the same row for different acts, open to
// different callers; EndOnEviction/EndOnDeparture are the two further ways a
// seat falls when the membership under it ends.
type EndReason string

const (
	EndedByRevocation  EndReason = "revoked"
	EndedByResignation EndReason = "resigned"
	EndedByEviction    EndReason = "evicted"
	EndedByDeparture   EndReason = "deregistered"
)
