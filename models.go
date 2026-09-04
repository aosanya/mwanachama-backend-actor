// Package mwanachamaactor provides Member/Group lifecycle management,
// extracted from mwanachama-backend-api-gateway's internal/domain/member and
// internal/domain/chapter packages onto the mwanachama-backend-taskmanager
// pattern: domain logic AND storage both delegated to a
// [github.com/aosanya/mwanachama-backend-shared/entitygraph.DataManager],
// imported directly into the gateway process — no separate service, no gRPC,
// no proto. Decided in a dev-research session, 2026-09-03 (DSN-1698).
//
// Implementation is split across focused files:
//   - models.go            — domain types
//   - schema.go             — DefaultUserSchema
//   - member.go             — UserManager interface
//   - member_impl.go        — Member CRUD
//   - group_impl.go         — Group CRUD, tree operations
//   - registration_impl.go  — Registration (registered_at edge) operations
//   - converters.go         — entity <-> domain converters
//
// Ported from mwanachama-backend-api-gateway's internal/domain/member and
// internal/domain/chapter packages. See this repo's CLAUDE.md for what
// changed along the way.
package mwanachamaactor

import "time"

// RelRegisteredAt is the edge label for a Member's enrolment at a Group.
// Was the `registration` table in the gateway; now a relationship
// (DSN-1698 decision 4).
const RelRegisteredAt = "registered_at"

// RelHasMember is the inverse of RelRegisteredAt, declared on Group.
const RelHasMember = "has_member"

// Member is a person enrolled in the network. Mirrors
// mwanachama-backend-api-gateway's internal/domain/member.Member field for
// field.
type Member struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
	CreatedAt   string `json:"created_at"`

	// IsAgentic marks a member driven by a model rather than a person —
	// DSN-1663. Not `omitempty` — see the gateway's member.Member doc for why.
	IsAgentic bool `json:"is_agentic"`

	// Attributes is the persona blob — DSN-1664's A-Box, a flat prop:value
	// map keyed by member_data_property.name. `omitempty`: an absent map and
	// an empty one mean the same thing (no persona).
	Attributes map[string]any `json:"attributes,omitempty"`
}

// Group is a node in the org structure — the generic type the gateway's
// domain calls "Chapter" (DSN-1698 decision 3) and presents under an
// org-configurable label. ParentID is empty for the root.
//
// HierarchyID, LevelID, and AnchorLevelOverrideID name rows that live in the
// gateway's OWN Postgres tables (hierarchy, chapter_level) — see the
// DSN-1699 gap 1 default recorded in schema.go's package doc. This package
// carries them as opaque strings and does not validate them against
// anything; the gateway adapter is responsible for whatever validation it
// still wants to do against its own hierarchy/chapter_level tables.
type Group struct {
	ID                    string `json:"id"`
	HierarchyID           string `json:"hierarchy_id"`
	LevelID               string `json:"level_id"`
	ParentID              string `json:"parent_id,omitempty"`
	Name                  string `json:"name"`
	Discoverable          bool   `json:"discoverable"`
	AnchorLevelOverrideID string `json:"anchor_level_override,omitempty"`
	NodeType              string `json:"node_type,omitempty"`
	CreatedAt             string `json:"created_at"`
}

// GroupEdit is the whole of what EditGroup may write. Mirrors the gateway's
// chapter.Edit. Empty clears — every field is written on every call.
type GroupEdit struct {
	Name                  string
	NodeType              string
	AnchorLevelOverrideID string
}

// Registration links a Member to a Group via the registered_at edge.
//
// IsHome is a plain boolean property with NO exclusivity semantics —
// DSN-1698 decision 8 drops "one home chapter per member" outright, not
// relocated. Nothing in this package enforces at-most-one-home.
type Registration struct {
	MemberID string `json:"member_id"`
	GroupID  string `json:"group_id"`
	IsHome   bool   `json:"is_home"`
	JoinedAt string `json:"joined_at"`
}

// timeLayout is the fixed-width, nanosecond-precision RFC 3339 layout every
// timestamp property in this package's schema is written and read with —
// including sort keys such as Group.CreatedAt and Registration.JoinedAt.
//
// Plain time.RFC3339 (second precision) is NOT enough: two registrations
// created microseconds apart in the same second would compare equal on
// JoinedAt and the sort would fall through to the tie-break (member/group
// id), which for randomly-minted UUIDs does not track real join order — see
// the gateway's TestRegistrationRosterOrderIsNotCallerWritable, the parity
// test that caught exactly this. time.RFC3339Nano is not safe either: Go
// trims trailing fractional zeros, so two timestamps with different digit
// counts stop comparing correctly as plain strings. A fixed nine-digit
// fractional part keeps every stored timestamp both parseable AND
// lexicographically sortable in the same order as chronologically.
const timeLayout = "2006-01-02T15:04:05.000000000Z07:00"

// nowRFC3339 returns the current UTC time formatted per [timeLayout].
func nowRFC3339() string {
	return time.Now().UTC().Format(timeLayout)
}
