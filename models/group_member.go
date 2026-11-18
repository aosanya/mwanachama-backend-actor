package models

type GroupMember struct {
	ActorID     string `json:"actor_id"`
	DisplayName string `json:"display_name"`
}

type GroupMemberPage struct {
	Items []GroupMember `json:"items"`
	Total int           `json:"total"`
}
