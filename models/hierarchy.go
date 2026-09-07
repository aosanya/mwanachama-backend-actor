package models

// Hierarchy is a named ladder of levels for one organization. Ported from
// mwanachama-backend-api-gateway's internal/domain/chapter.Hierarchy
// field-for-field — DSN-1699 gap 1, resolved: Hierarchy/Level move here
// alongside Group rather than staying behind in the gateway's own tables.
// See this repo's CLAUDE.md for the decision record.
type Hierarchy struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Level is a rung on a hierarchy (e.g. "Ward"), ordered from grassroots
// (Depth 0) up to the national council.
//
// IsDefaultAnchor marks the hierarchy's anchor level: the coarsest level at
// which a self-service member registers. At most one per hierarchy — see
// gormstore.Migrate's partial unique index, the DB-level half of the
// exclusivity CreateLevel and SetDefaultAnchor enforce in Go.
type Level struct {
	ID              string `json:"id"`
	HierarchyID     string `json:"hierarchy_id"`
	Name            string `json:"name"`
	Depth           int    `json:"depth"`
	IsDefaultAnchor bool   `json:"is_default_anchor,omitempty"`
}
