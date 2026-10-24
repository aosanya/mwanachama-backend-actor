package models

import "fmt"

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

func ParseCodeGroups(value any) (int, error) {
	var n int
	switch v := value.(type) {
	case int:
		n = v
	case int64:
		n = int(v)
	case float64:
		if v != float64(int(v)) {
			return 0, fmt.Errorf("code groups: %v is not a whole number", v)
		}
		n = int(v)
	default:
		return 0, fmt.Errorf("code groups: %v is not a whole number", value)
	}
	if n < 1 || n > MaxCodeGroups {
		return 0, fmt.Errorf("code groups: a code has between 1 and %d groups, not %d", MaxCodeGroups, n)
	}
	return n, nil
}
