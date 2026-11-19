package models

type RoleKind struct {
	ID                 string   `json:"id"`
	Code               string   `json:"code"`
	Name               string   `json:"name"`
	Capabilities       []string `json:"capabilities"`
	Description        string   `json:"description,omitempty"`
	IsDelegate         bool     `json:"is_delegate,omitempty"`
	LeadsGroup         bool     `json:"leads_group,omitempty"`
	Staff              bool     `json:"staff,omitempty"`
	ApprovesAboveLimit bool     `json:"approves_above_limit,omitempty"`
	Outsider           bool     `json:"outsider,omitempty"`
	LevelID            string   `json:"level_id,omitempty"`
	DefinitionRoleID   string   `json:"definition_role_id,omitempty"`
	RetiredAt          string   `json:"retired_at,omitempty"`
	RetiredBy          string   `json:"retired_by,omitempty"`
}

func (k RoleKind) Retired() bool { return k.RetiredAt != "" }

type EndReason string

const (
	EndedByRevocation  EndReason = "revoked"
	EndedByResignation EndReason = "resigned"
	EndedByEviction    EndReason = "evicted"
	EndedByDeparture   EndReason = "deregistered"
	EndedByRoleRemoved EndReason = "role_removed"
)
