package models

type StandingRegistration struct {
	GroupID   string `json:"group_id"`
	GroupName string `json:"group_name"`
	Since     string `json:"since"`
}

type StandingSeat struct {
	ID               string    `json:"id"`
	GroupID          string    `json:"group_id"`
	GroupName        string    `json:"group_name"`
	KindID           string    `json:"kind_id"`
	RoleName         string    `json:"role_name"`
	DefinitionRoleID string    `json:"definition_role_id,omitempty"`
	LeadsGroup       bool      `json:"leads_group"`
	Staff            bool      `json:"staff"`
	GrantedAt        string    `json:"granted_at"`
	GrantedBy        string    `json:"granted_by,omitempty"`
	GrantedByName    string    `json:"granted_by_name,omitempty"`
	EndedAt          string    `json:"ended_at,omitempty"`
	EndedReason      EndReason `json:"ended_reason,omitempty"`
}

type ActorStanding struct {
	ActorID       string                 `json:"actor_id"`
	DisplayName   string                 `json:"display_name"`
	Registrations []StandingRegistration `json:"registrations"`
	Seats         []StandingSeat         `json:"seats"`
}
