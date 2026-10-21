package models

type RoleHolder struct {
	ActorRoleAssignment
	DisplayName string `json:"display_name"`
}
