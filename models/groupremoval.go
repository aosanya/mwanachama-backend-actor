package models

type GroupRemoval struct {
	ID        string `json:"id"`
	ActorID   string `json:"actor_id"`
	GroupID   string `json:"group_id"`
	RemovedBy string `json:"removed_by"`
	RemovedAt string `json:"removed_at"`
}
