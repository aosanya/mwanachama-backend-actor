package models

type GroupNameTranslation struct {
	ID          string `json:"id"`
	GroupID     string `json:"group_id"`
	Language    string `json:"language"`
	Name        string `json:"name"`
	Reviewed    bool   `json:"reviewed"`
	ReviewedBy  string `json:"reviewed_by,omitempty"`
	ReviewedAt  string `json:"reviewed_at,omitempty"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
}
