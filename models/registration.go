package models

// ActorGroupAssignment links an Actor to a Group. Was a labeled entitygraph
// edge (registered_at/has_actor — DSN-1698 decision 4); now a real row (see
// the gormstore package's ActorGroupAssignmentRow), identified by the
// (ActorID, GroupID) pair rather than an edge label.
//
// IsHome is a plain boolean field with NO exclusivity semantics —
// DSN-1698 decision 8 drops "one home chapter per member" outright, not
// relocated. Nothing in this package enforces at-most-one-home.
type ActorGroupAssignment struct {
	ActorID  string `json:"actor_id"`
	GroupID  string `json:"group_id"`
	IsHome   bool   `json:"is_home"`
	JoinedAt string `json:"joined_at"`

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
