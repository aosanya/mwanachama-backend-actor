package models

const (
	CodeFormatID      = "codes"
	DefaultCodeGroups = 3
	MaxCodeGroups     = 12
)

type CodeFormat struct {
	ID          string `json:"id"`
	GroupCount  int    `json:"group_count"`
	LastUpdated string `json:"last_updated"`
}
