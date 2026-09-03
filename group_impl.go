// group_impl.go — Group CRUD and tree-operation implementation for
// [userManager]. Ported from mwanachama-backend-api-gateway's
// internal/store/postgres/chapter_store.go and chapter_store_edit.go (the
// Chapter-CRUD half only — Hierarchy/Level stay behind in the gateway, per
// the DSN-1699 gap 1 default recorded in schema.go's package doc).
package mwanachamauser

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/aosanya/mwanachama-backend-shared/entitygraph"
)

// CreateGroup creates a new Group entity.
func (m *userManager) CreateGroup(ctx context.Context, agencyID string, g Group) (Group, error) {
	if g.Name == "" {
		return Group{}, fmt.Errorf("%w: Group.Name is required", ErrInvalidGroup)
	}
	if g.CreatedAt == "" {
		g.CreatedAt = nowRFC3339()
	}
	created, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		AgencyID:   agencyID,
		TypeID:     groupTypeID,
		Properties: groupToProperties(g),
	})
	if err != nil {
		return Group{}, fmt.Errorf("CreateGroup: %w", err)
	}
	return groupFromEntity(created), nil
}

// GetGroup reads a single Group entity.
func (m *userManager) GetGroup(ctx context.Context, agencyID, id string) (Group, error) {
	e, err := m.dm.GetEntity(ctx, agencyID, id)
	if err != nil {
		if errors.Is(err, entitygraph.ErrEntityNotFound) {
			return Group{}, ErrGroupNotFound
		}
		return Group{}, fmt.Errorf("GetGroup: %w", err)
	}
	if e.TypeID != groupTypeID {
		return Group{}, ErrGroupNotFound
	}
	return groupFromEntity(e), nil
}

// EditGroup writes name/node_type/anchor_level_override. Empty clears —
// every field is written on every call, mirroring the gateway's
// chapter.Repository.EditChapter contract exactly.
func (m *userManager) EditGroup(ctx context.Context, agencyID, id string, e GroupEdit) (Group, error) {
	current, err := m.GetGroup(ctx, agencyID, id)
	if err != nil {
		return Group{}, err
	}
	updated, err := m.dm.UpdateEntity(ctx, agencyID, id, entitygraph.UpdateEntityRequest{
		Properties: map[string]any{
			"name":                  e.Name,
			"node_type":             e.NodeType,
			"anchor_level_override": e.AnchorLevelOverrideID,
		},
	})
	if err != nil {
		if errors.Is(err, entitygraph.ErrEntityNotFound) {
			return Group{}, ErrGroupNotFound
		}
		return Group{}, fmt.Errorf("EditGroup: %w", err)
	}
	out := groupFromEntity(updated)
	out.CreatedAt = current.CreatedAt
	out.HierarchyID = current.HierarchyID
	out.LevelID = current.LevelID
	out.ParentID = current.ParentID
	out.Discoverable = current.Discoverable
	return out, nil
}

// MoveGroup re-parents a group, refusing the three moves that cannot mean
// anything — mirrors the gateway's chapter.Repository.MoveChapter.
//
// The subtree/cycle check is a Go walk over ListGroups rather than a
// recursive CTE: entitygraph.DataManager exposes no arbitrary-SQL escape
// hatch, so the Postgres store's WITH RECURSIVE query has no equivalent
// here. For an org-sized tree (hundreds, not millions, of groups) an O(n)
// walk per move is not a performance concern; if it ever becomes one, the
// gateway adapter is a more natural place to add a materialized-path
// property than this package.
func (m *userManager) MoveGroup(ctx context.Context, agencyID, id, newParentID string) (Group, error) {
	if newParentID == id {
		return Group{}, ErrParentIsSelf
	}
	current, err := m.GetGroup(ctx, agencyID, id)
	if err != nil {
		return Group{}, err
	}
	if current.ParentID == "" {
		return Group{}, ErrRootCannotMove
	}
	if newParentID != "" {
		if _, err := m.GetGroup(ctx, agencyID, newParentID); err != nil {
			if errors.Is(err, ErrGroupNotFound) {
				return Group{}, ErrParentNotFound
			}
			return Group{}, err
		}
	}

	all, err := m.ListGroups(ctx, agencyID, "")
	if err != nil {
		return Group{}, fmt.Errorf("MoveGroup: %w", err)
	}
	byID := make(map[string]Group, len(all))
	for _, g := range all {
		byID[g.ID] = g
	}
	// Walk UP from the proposed parent. If id appears on that path, the
	// parent is one of id's own descendants.
	seen := map[string]bool{}
	for cur, ok := byID[newParentID]; ok && !seen[cur.ID]; cur, ok = byID[cur.ParentID] {
		if cur.ID == id {
			return Group{}, ErrParentInSubtree
		}
		seen[cur.ID] = true
	}

	updated, err := m.dm.UpdateEntity(ctx, agencyID, id, entitygraph.UpdateEntityRequest{
		Properties: map[string]any{"parent_id": newParentID},
	})
	if err != nil {
		if errors.Is(err, entitygraph.ErrEntityNotFound) {
			return Group{}, ErrGroupNotFound
		}
		return Group{}, fmt.Errorf("MoveGroup: %w", err)
	}
	return groupFromEntity(updated), nil
}

// ListGroupChildren returns the direct children of a group (empty parentID
// returns the roots), id order.
func (m *userManager) ListGroupChildren(ctx context.Context, agencyID, parentID string) ([]Group, error) {
	all, err := m.ListGroups(ctx, agencyID, "")
	if err != nil {
		return nil, fmt.Errorf("ListGroupChildren: %w", err)
	}
	out := []Group{}
	for _, g := range all {
		if g.ParentID == parentID {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// ListGroups returns every group in the agency, optionally filtered to one
// hierarchy, created_at-then-id order (matching the gateway Postgres store's
// ORDER BY created_at, id).
func (m *userManager) ListGroups(ctx context.Context, agencyID, hierarchyID string) ([]Group, error) {
	props := map[string]any{}
	if hierarchyID != "" {
		props["hierarchy_id"] = hierarchyID
	}
	entities, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID:   agencyID,
		TypeID:     groupTypeID,
		Properties: props,
	})
	if err != nil {
		return nil, fmt.Errorf("ListGroups: %w", err)
	}
	out := make([]Group, 0, len(entities))
	for _, e := range entities {
		out = append(out, groupFromEntity(e))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// ListDiscoverableGroups returns groups flagged discoverable, optionally
// name-filtered, id order.
func (m *userManager) ListDiscoverableGroups(ctx context.Context, agencyID, query string) ([]Group, error) {
	all, err := m.ListGroups(ctx, agencyID, "")
	if err != nil {
		return nil, fmt.Errorf("ListDiscoverableGroups: %w", err)
	}
	q := strings.ToLower(query)
	out := []Group{}
	for _, g := range all {
		if !g.Discoverable {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(g.Name), q) {
			continue
		}
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
