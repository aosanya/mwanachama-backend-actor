package mwanachamaactor

import (
	"encoding/json"

	"github.com/aosanya/mwanachama-backend-shared/entitygraph"
)

// ── Member ───────────────────────────────────────────────────────────────

func memberToProperties(m Member) map[string]any {
	attrs := ""
	if len(m.Attributes) > 0 {
		if b, err := json.Marshal(m.Attributes); err == nil {
			attrs = string(b)
		}
	}
	return map[string]any{
		"display_name": m.DisplayName,
		"phone":        m.Phone,
		"email":        m.Email,
		"is_agentic":   m.IsAgentic,
		"attributes":   attrs,
		"created_at":   m.CreatedAt,
		"updated_at":   nowRFC3339(),
	}
}

func memberFromEntity(e entitygraph.Entity) Member {
	m := Member{
		ID:          e.ID,
		DisplayName: entitygraph.StringProp(e.Properties, "display_name"),
		Phone:       entitygraph.StringProp(e.Properties, "phone"),
		Email:       entitygraph.StringProp(e.Properties, "email"),
		IsAgentic:   entitygraph.BoolProp(e.Properties, "is_agentic"),
		CreatedAt:   entitygraph.StringProp(e.Properties, "created_at"),
	}
	if raw := entitygraph.StringProp(e.Properties, "attributes"); raw != "" && raw != "{}" {
		var attrs map[string]any
		if err := json.Unmarshal([]byte(raw), &attrs); err == nil && len(attrs) > 0 {
			m.Attributes = attrs
		}
	}
	if m.CreatedAt == "" {
		m.CreatedAt = e.CreatedAt.UTC().Format(timeLayout)
	}
	return m
}

// ── Group ────────────────────────────────────────────────────────────────

func groupToProperties(g Group) map[string]any {
	return map[string]any{
		"name":                  g.Name,
		"hierarchy_id":          g.HierarchyID,
		"level_id":              g.LevelID,
		"parent_id":             g.ParentID,
		"discoverable":          g.Discoverable,
		"anchor_level_override": g.AnchorLevelOverrideID,
		"node_type":             g.NodeType,
		"created_at":            g.CreatedAt,
		"updated_at":            nowRFC3339(),
	}
}

func groupFromEntity(e entitygraph.Entity) Group {
	g := Group{
		ID:                    e.ID,
		Name:                  entitygraph.StringProp(e.Properties, "name"),
		HierarchyID:           entitygraph.StringProp(e.Properties, "hierarchy_id"),
		LevelID:               entitygraph.StringProp(e.Properties, "level_id"),
		ParentID:              entitygraph.StringProp(e.Properties, "parent_id"),
		Discoverable:          entitygraph.BoolProp(e.Properties, "discoverable"),
		AnchorLevelOverrideID: entitygraph.StringProp(e.Properties, "anchor_level_override"),
		NodeType:              entitygraph.StringProp(e.Properties, "node_type"),
		CreatedAt:             entitygraph.StringProp(e.Properties, "created_at"),
	}
	if g.CreatedAt == "" {
		g.CreatedAt = e.CreatedAt.UTC().Format(timeLayout)
	}
	return g
}
