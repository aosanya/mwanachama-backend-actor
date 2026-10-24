package models

type DefinitionRole struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Staff bool   `json:"staff,omitempty"`
}

type DefinitionRoleSync struct {
	Renamed []RoleKind `json:"renamed"`
	Retired []RoleKind `json:"retired"`
}
