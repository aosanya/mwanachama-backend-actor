package models

// Actor is a person enrolled in the network. Mirrors
// mwanachama-backend-api-gateway's internal/domain/member.Member field for
// field, with one deliberate divergence: Phone and Email are NOT dedicated
// columns here the way they are on member.Member. They are Attributes
// entries declared by DefaultActorProperties (2026-09-04), so the built-in
// pair and anything an organization adds later all go through the same
// Required/Unique/Range validation — see property.go. A caller porting a
// gateway member.Member writes .Phone/.Email into
// Attributes["phone"]/Attributes["email"] instead of a struct field.
type Actor struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
	// LastUpdated is stamped on every write (create and every subsequent
	// edit) — see gormstore's ActorRow.UpdatedAt, which this mirrors.
	LastUpdated string `json:"last_updated"`
	// Deleted marks a soft-deleted actor. Not `omitempty`: false is a real,
	// meaningful answer, not an absent one. Nothing in this package sets it
	// true yet — no delete method exists — but every read that lists rather
	// than names one by id (ListActors) filters it out, so the column and
	// the filter land together rather than the column sitting unused.
	Deleted bool `json:"deleted"`

	// IsAgentic marks an actor driven by a model rather than a person —
	// DSN-1663. Not `omitempty` — see the gateway's member.Member doc for why.
	IsAgentic bool `json:"is_agentic"`

	// Attributes is the persona blob — DSN-1664's A-Box, a flat prop:value
	// map keyed by a Property.Name, built-in (DefaultActorProperties) or
	// organization-declared. `omitempty`: an absent map and an empty one
	// mean the same thing (no persona).
	Attributes map[string]any `json:"attributes,omitempty"`
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
