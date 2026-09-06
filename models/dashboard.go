package models

// RoleWithMembers is one role kind at a group and the actors holding it.
// Ported from mwanachama-backend-api-gateway's
// internal/domain/dashboard.RoleWithMembers, MemberID -> ActorID's rename
// carried into the JSON key the way ActorGroupAssignment carried it before.
type RoleWithMembers struct {
	KindID   string   `json:"kind_id"`
	KindName string   `json:"kind_name"`
	Members  []string `json:"actor_ids"`
}

// GroupDashboard is the composed view returned for a group — mwanachama-
// backend-api-gateway's internal/domain/dashboard.ChapterDashboard, ChapterID
// renamed the way every other type in this package renamed Chapter to Group.
type GroupDashboard struct {
	GroupID     string            `json:"group_id"`
	Roles       []RoleWithMembers `json:"roles"`
	MemberCount int               `json:"member_count"`
	ChildCount  int               `json:"child_count"`
	ActiveRoles int               `json:"active_role_count"`
}
