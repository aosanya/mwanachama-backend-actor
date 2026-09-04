// Package mwanachamaactor — pre-delivered schema definition.
//
// This file exposes [DefaultUserSchema], which returns the
// [schema.Schema] for one mounted instance of mwanachama-backend-actor.
// Wiring code in mwanachama-backend-api-gateway seeds this schema at startup
// via SchemaManager.SetSchema — one deployment per agency (single-tenant),
// so no agency scoping is threaded through the schema itself. The gateway
// may mount this package more than once — e.g. "member" today,
// with others possible later — each as its own instance with its own
// Postgres table prefix; DefaultUserSchema takes that instance name so its
// StorageCollection labels stay in step with the table prefix the gateway
// wires alongside it (see the Storage paragraph below).
//
// The schema declares two TypeDefinitions:
//   - Member — a person enrolled in the network (mutable)
//   - Group  — a node in the org structure; the generic name for what the
//     gateway's own domain calls "Chapter" and presents under an
//     org-configurable label (DSN-1698/DSN-1699 gap 2, unresolved — the
//     gateway is responsible for the label, not this package)
//
// Graph topology:
//
//	Member ──registered_at──► Group   (properties: is_home, joined_at)
//	Group  ──has_member──────► Member (inverse of registered_at)
//
// DSN-1699 gap 1 (hierarchy / chapter_level fate) is resolved for this pass
// with the DEFAULT the design doc names: hierarchy and chapter_level stay
// physically in the gateway's own Postgres tables. Group carries
// hierarchy_id / level_id / parent_id as plain JSONB properties with NO
// foreign key back to those gateway tables — same "no worse than today"
// posture already accepted for the 16 chapter_id-referencing tables. This
// means Level.IsDefaultAnchor / Chapter.AnchorLevelOverrideID's cross-hierarchy
// composite-FK guarantee (migration 000034 in the gateway) has NO equivalent
// here: an anchor_level_override value that names a level from a different
// hierarchy is not rejected by this package or its storage. See this
// repo's documentation/2. design/todo.md for the note recording this as a
// default, not a settled decision.
//
// Storage: every entity lives in whatever Postgres tables the caller's
// mwanachama-backend-shared/postgres.Backend is configured with — the
// gateway wires each instance to postgres.DefaultTableNames(instance+"_"),
// e.g. "member" gives member_entities / member_relationships /
// member_schemas_draft / member_schemas_published, mirroring
// mwanachama-backend-taskmanager's work_* convention exactly.
// TypeDefinition.StorageCollection below is carried over purely as a label
// (see schema.TypeDefinition's doc), derived from the same instance name for
// consistency, and has no functional effect here.
package mwanachamaactor

import "github.com/aosanya/mwanachama-backend-shared/schema"

// DefaultUserSchema returns the pre-delivered [schema.Schema] for one
// mounted instance of this package, seeded by mwanachama-backend-api-gateway
// on startup via SchemaManager.SetSchema. instance names the mount (e.g.
// "member") and is used only to derive the two TypeDefinitions'
// StorageCollection labels below — it has no bearing on schema.Schema.ID/Tag,
// which stay fixed so every instance runs the same schema version. The
// operation is idempotent — calling it multiple times for the same instance
// is safe.
func DefaultUserSchema(instance string) schema.Schema {
	return schema.Schema{
		ID:      "user-schema-v1",
		Version: 1,
		Tag:     "v1",
		Types: []schema.TypeDefinition{
			{
				Name:              "Member",
				DisplayName:       "Member",
				StorageCollection: instance + "_members",
				Properties: []schema.PropertyDefinition{
					// display_name is the name the member gave for themselves.
					{Name: "display_name", Type: schema.PropertyTypeString},
					// phone is the member's phone number, empty when not given.
					{Name: "phone", Type: schema.PropertyTypeString},
					// email is the member's email address, empty when not given.
					{Name: "email", Type: schema.PropertyTypeString},
					// is_agentic marks a member driven by a model rather than a
					// person — DSN-1663. Not omitted when false: a real member must
					// serialise this explicitly so a caller cannot mistake an
					// absent field for a false one.
					{Name: "is_agentic", Type: schema.PropertyTypeBoolean},
					// attributes is the persona blob (DSN-1664's A-Box), stored as a
					// JSON-encoded object string — flat prop:value keyed by
					// member_data_property.name. Empty ("" or "{}") for every real
					// member. JSON-encoded rather than a native map because
					// schema.PropertyType has no object/map type (see schema.go).
					{Name: "attributes", Type: schema.PropertyTypeString},
					{Name: "created_at", Type: schema.PropertyTypeString},
					{Name: "updated_at", Type: schema.PropertyTypeString},
				},
				Relationships: []schema.RelationshipDefinition{
					{
						Name:    RelRegisteredAt,
						Label:   "Registered at",
						ToType:  "Group",
						ToMany:  true,
						Inverse: RelHasMember,
						Properties: []schema.PropertyDefinition{
							// is_home marks the member's home group. Plain boolean
							// property with NO exclusivity semantics — decision 8 of
							// DSN-1698 drops "one home chapter per member" outright,
							// not relocated. Multiple registrations may carry
							// is_home=true for the same member; nothing here or in the
							// gateway adapter enforces otherwise.
							{Name: "is_home", Type: schema.PropertyTypeBoolean},
							{Name: "joined_at", Type: schema.PropertyTypeString},
						},
					},
				},
			},
			{
				Name:              "Group",
				DisplayName:       "Group",
				StorageCollection: instance + "_groups",
				Properties: []schema.PropertyDefinition{
					// name is the short human-readable label.
					{Name: "name", Type: schema.PropertyTypeString, Required: true},
					// hierarchy_id names the hierarchy this group's level belongs to.
					// Plain property, no FK — see the package doc for why.
					{Name: "hierarchy_id", Type: schema.PropertyTypeString},
					// level_id names the rung on that hierarchy. Plain property, no FK.
					{Name: "level_id", Type: schema.PropertyTypeString},
					// parent_id names the parent Group entity ID, empty for a root.
					// Deliberately a plain property rather than a graph edge — the
					// gateway's own tree-walk queries (MoveChapter's cycle check) are
					// reproduced in Go over ListGroups rather than as a recursive CTE,
					// since entitygraph exposes no arbitrary-SQL escape hatch.
					{Name: "parent_id", Type: schema.PropertyTypeString},
					// discoverable is the visibility fence — see chapter.Chapter.Discoverable
					// in the gateway for the policy; this package stores the bit only.
					{Name: "discoverable", Type: schema.PropertyTypeBoolean},
					// anchor_level_override overrides the hierarchy's default anchor
					// level for this group's subtree. Plain property, no composite FK
					// to (level_id, hierarchy_id) — the guarantee migration 000034 gave
					// in the gateway has no equivalent here. See package doc.
					{Name: "anchor_level_override", Type: schema.PropertyTypeString},
					// node_type is a party-editable descriptive tag. Empty means untyped.
					{Name: "node_type", Type: schema.PropertyTypeString},
					{Name: "created_at", Type: schema.PropertyTypeString},
					{Name: "updated_at", Type: schema.PropertyTypeString},
				},
				Relationships: []schema.RelationshipDefinition{
					{
						Name:    RelHasMember,
						Label:   "Has member",
						ToType:  "Member",
						ToMany:  true,
						Inverse: RelRegisteredAt,
					},
				},
			},
		},
	}
}
