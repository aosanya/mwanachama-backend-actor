package models

type DefinitionRole struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Staff              bool   `json:"staff,omitempty"`
	ApprovesAboveLimit bool   `json:"approves_above_limit,omitempty"`
}

type DefinitionRoleSync struct {
	Renamed []RoleKind `json:"renamed"`
	Retired []RoleKind `json:"retired"`
}
