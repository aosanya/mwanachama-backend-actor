package models

const (
	AttributeSetBySelf  = "self"
	AttributeSetByStaff = "staff"
)

type AttributeSource struct {
	SetBy   string `json:"set_by"`
	SetByID string `json:"set_by_id"`
	SetAt   string `json:"set_at"`
}

type AttributeCorrection struct {
	Attributes map[string]any `json:"attributes"`
}
