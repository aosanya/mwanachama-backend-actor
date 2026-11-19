package models

type Actor struct {
	ID               string                     `json:"id"`
	Code             string                     `json:"code"`
	PublicCode       string                     `json:"public_code"`
	DisplayName      string                     `json:"display_name"`
	FirstName        string                     `json:"first_name"`
	LastName         string                     `json:"last_name"`
	CreatedAt        string                     `json:"created_at"`
	LastUpdated      string                     `json:"last_updated"`
	Deleted          bool                       `json:"deleted"`
	IsAgentic        bool                       `json:"is_agentic"`
	FormsWeekSplit   bool                       `json:"forms_week_split"`
	Attributes       map[string]any             `json:"attributes,omitempty"`
	AttributeSources map[string]AttributeSource `json:"attribute_sources,omitempty"`
}

func DefaultActorProperties() []Property {
	return []Property{
		{Name: "phone", Label: "Phone", Range: RangeText, Unique: true},
		{Name: "email", Label: "Email", Range: RangeText, Unique: true},
		{Name: "date_of_birth", Label: "Date of birth", Range: RangeDate, NotFuture: true},
	}
}
