package models

// ActorRoleAssignment binds an Actor to a RoleKind at a Group — mwanachama-
// backend-api-gateway's internal/domain/role.Assignment, renamed the same
// way ActorGroupAssignment renamed Registration: MemberID -> ActorID,
// ChapterID -> GroupID.
//
// Active is false once the member steps down or is revoked. EndedAt and
// EndedReason are set together, and only when the assignment is deactivated
// — both carry `omitempty` so an **active** assignment serialises exactly as
// it did before either was ever set.
type ActorRoleAssignment struct {
	ID      string `json:"id"`
	ActorID string `json:"actor_id"`
	GroupID string `json:"group_id"`
	KindID  string `json:"kind_id"`
	Active  bool   `json:"active"`
	// GrantedAt/EndedAt are RFC3339Nano strings, TimeLayout-formatted — see
	// models/time.go's doc for why plain string comparison has to sort the
	// same as chronological order here: ListForGroup/ListForActor both order
	// by (GrantedAt, ID).
	GrantedAt string `json:"granted_at"`
	// GrantedBy is the actor id of whoever granted this seat, stamped
	// server-side by the mounting process from its caller's own session —
	// this package never trusts a caller-supplied value here. Empty on the
	// rows a Structure creates at first run, before any operator exists to
	// grant one.
	GrantedBy   string    `json:"granted_by,omitempty"`
	EndedAt     string    `json:"ended_at,omitempty"`
	EndedReason EndReason `json:"ended_reason,omitempty"`
}
