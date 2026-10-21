package models

type Actor struct {
	ID          string         `json:"id"`
	Code        string         `json:"code"`
	PublicCode  string         `json:"public_code"`
	DisplayName string         `json:"display_name"`
	FirstName   string         `json:"first_name"`
	LastName    string         `json:"last_name"`
	CreatedAt   string         `json:"created_at"`
	LastUpdated string         `json:"last_updated"`
	Deleted     bool           `json:"deleted"`
	IsAgentic   bool           `json:"is_agentic"`
	Attributes  map[string]any `json:"attributes,omitempty"`
}

// DefaultActorProperties is the built-in property catalog validated against
// Actor.Attributes on every CreateActor.
//
// Neither phone nor email is Required: the gateway's registerDevice mints a
// member row before any phone number is given, and an agentic actor is
// forbidden a phone outright (migration 000058's
// member_agentic_has_no_phone) — this catalog does not reintroduce a
// requirement the gateway deliberately dropped. Both are Unique: two actors
// sharing a phone or email is exactly what the old plain-string columns
// never caught, and is this catalog's reason to exist.
func DefaultActorProperties() []Property {
	return []Property{
		{Name: "phone", Label: "Phone", Range: RangeText, Unique: true},
		{Name: "email", Label: "Email", Range: RangeText, Unique: true},
	}
}
