package models

// ActorGroupAssignment links an Actor to a Group. Was a labeled entitygraph
// edge (registered_at/has_actor — DSN-1698 decision 4); now a real row (see
// the gormstore package's ActorGroupAssignmentRow), identified by the
// (ActorID, GroupID) pair rather than an edge label.
//
// **No IsHome or JoinedAt fields, 2026-09-04.** Both were dedicated
// columns; neither is one any more. CreatedAt now carries JoinedAt's old
// role — the moment this assignment first existed — and AssignGroup
// preserves it across a re-assign the same way it used to preserve
// JoinedAt (see AssignGroup's doc). A caller that still wants a "home"
// flag, or an explicit historical join date an import track needs to
// backdate, writes it into Attributes like any other organization-declared
// property — nothing here validates or interprets an "is_home" key
// specially, unlike Phone/Email, which are built into
// DefaultActorProperties. DSN-1698 decision 8's "no exclusivity" holds
// exactly as before: nothing in this package enforces at-most-one-home
// regardless of where the flag lives.
type ActorGroupAssignment struct {
	ActorID     string         `json:"actor_id"`
	GroupID     string         `json:"group_id"`
	Priority    string         `json:"priority"`
	CreatedAt   string         `json:"created_at"`
	LastUpdated string         `json:"last_updated"`
	Deleted     bool           `json:"deleted"`
	Attributes  map[string]any `json:"attributes,omitempty"`
}

// DefaultActorGroupAssignmentProperties is the built-in property catalog
// validated against ActorGroupAssignment.Attributes on every AssignGroup.
// Empty for now — no built-in assignment property has been decided yet —
// but the mechanism mirrors Actor's and Group's: an organization-declared
// property added here immediately gets Required/Unique/Range validation for
// free.
func DefaultActorGroupAssignmentProperties() []Property {
	return nil
}

const (
	PriorityVeryHigh = "very_high"
	PriorityHigh     = "high"
	PriorityNormal   = "normal"
	PriorityLow      = "low"
	PriorityVeryLow  = "very_low"
)

func ValidPriority(p string) bool {
	switch p {
	case PriorityVeryHigh, PriorityHigh, PriorityNormal, PriorityLow, PriorityVeryLow:
		return true
	}
	return false
}

func PriorityOrDefault(p string) string {
	if p == "" {
		return PriorityNormal
	}
	return p
}
