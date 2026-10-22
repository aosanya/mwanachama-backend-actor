package models

type DefinitionRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DefinitionRoleSync struct {
	Renamed []RoleKind `json:"renamed"`
	Retired []RoleKind `json:"retired"`
}
