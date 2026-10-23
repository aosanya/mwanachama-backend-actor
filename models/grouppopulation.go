package models

type GroupPopulation struct {
	ID         string `json:"id"`
	GroupID    string `json:"group_id"`
	Population int    `json:"population"`
	AsOf       string `json:"as_of"`
	Source     string `json:"source"`
	RecordedBy string `json:"recorded_by,omitempty"`
	CreatedAt  string `json:"created_at"`
}
