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
	ActorID string `json:"actor_id"`
	GroupID string `json:"group_id"`
	// CreatedAt is set once, on the assignment's first AssignGroup call,
	// and preserved across every subsequent re-assign of the same pair.
	CreatedAt string `json:"created_at"`
	// LastUpdated is stamped on every AssignGroup call, create or update.
	LastUpdated string `json:"last_updated"`
	// Deleted marks a soft-deleted assignment. Not `omitempty` — see
	// Actor's identical field for why. Unused today: Deregister hard-deletes
	// the row rather than setting this.
	Deleted bool `json:"deleted"`

	// Attributes is an open prop:value map for organization-declared
	// assignment properties, validated against
	// DefaultActorGroupAssignmentProperties the same way Actor.Attributes is
	// validated against DefaultActorProperties — see property.go. Added
	// 2026-09-04. `omitempty`: an absent map and an empty one mean the same
	// thing.
	Attributes map[string]any `json:"attributes,omitempty"`
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
