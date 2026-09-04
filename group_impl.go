// group_impl.go — Group CRUD and tree-operation implementation for
// [userManager]. Ported from mwanachama-backend-api-gateway's
// internal/store/postgres/chapter_store.go and chapter_store_edit.go (the
// Chapter-CRUD half only — Hierarchy/Level stay behind in the gateway, per
// the DSN-1699 gap 1 default recorded in gormstore's GroupRow doc).
package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-actor/gormstore"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

// CreateGroup creates a new Group entity. Attributes is validated against
// [models.DefaultGroupProperties] the same way CreateActor validates
// against [models.DefaultActorProperties].
func (m *userManager) CreateGroup(ctx context.Context, g models.Group) (models.Group, error) {
	if g.Name == "" {
		return models.Group{}, fmt.Errorf("%w: Group.Name is required", ErrInvalidGroup)
	}
	properties := models.DefaultGroupProperties()
	if err := models.ValidateAttributes(properties, g.Attributes); err != nil {
		return models.Group{}, fmt.Errorf("%w: %v", ErrInvalidGroup, err)
	}
	if err := m.checkUniqueAttributes(ctx, m.tables.Groups, properties, g.Attributes); err != nil {
		return models.Group{}, err
	}
	if g.CreatedAt == "" {
		g.CreatedAt = models.NowRFC3339()
	}
	row := gormstore.GroupToRow(g)
	if err := m.db.WithContext(ctx).Table(m.tables.Groups).Create(&row).Error; err != nil {
		return models.Group{}, fmt.Errorf("CreateGroup: %w", err)
	}
	return gormstore.GroupFromRow(row), nil
}

// GetGroup reads a single Group entity.
func (m *userManager) GetGroup(ctx context.Context, id string) (models.Group, error) {
	var row gormstore.GroupRow
	err := m.db.WithContext(ctx).Table(m.tables.Groups).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Group{}, ErrGroupNotFound
		}
		return models.Group{}, fmt.Errorf("GetGroup: %w", err)
	}
	return gormstore.GroupFromRow(row), nil
}

// EditGroup writes name/node_type/anchor_level_override. Empty clears —
// every field is written on every call, mirroring the gateway's
// chapter.Repository.EditChapter contract exactly.
func (m *userManager) EditGroup(ctx context.Context, id string, e models.GroupEdit) (models.Group, error) {
	current, err := m.GetGroup(ctx, id)
	if err != nil {
		return models.Group{}, err
	}
	err = m.db.WithContext(ctx).Table(m.tables.Groups).Where("id = ?", id).
		Updates(map[string]any{
			"name":                  e.Name,
			"node_type":             e.NodeType,
			"anchor_level_override": e.AnchorLevelOverrideID,
			"updated_at":            models.NowRFC3339(),
		}).Error
	if err != nil {
		return models.Group{}, fmt.Errorf("EditGroup: %w", err)
	}
	current.Name = e.Name
	current.NodeType = e.NodeType
	current.AnchorLevelOverrideID = e.AnchorLevelOverrideID
	return current, nil
}

// MoveGroup re-parents a group, refusing the three moves that cannot mean
// anything — mirrors the gateway's chapter.Repository.MoveChapter.
//
// The subtree/cycle check is a Go walk over ListGroups rather than a
// recursive CTE — see gormstore's GroupRow doc for why ParentID stays a
// plain indexed column rather than a GORM-managed association; a
// hand-written recursive CTE would work fine against the real FK-less
// column, but is out of scope for this storage swap.
func (m *userManager) MoveGroup(ctx context.Context, id, newParentID string) (models.Group, error) {
	if newParentID == id {
		return models.Group{}, ErrParentIsSelf
	}
	current, err := m.GetGroup(ctx, id)
	if err != nil {
		return models.Group{}, err
	}
	if current.ParentID == "" {
		return models.Group{}, ErrRootCannotMove
	}
	if newParentID != "" {
		if _, err := m.GetGroup(ctx, newParentID); err != nil {
			if errors.Is(err, ErrGroupNotFound) {
				return models.Group{}, ErrParentNotFound
			}
			return models.Group{}, err
		}
	}

	all, err := m.ListGroups(ctx, "")
	if err != nil {
		return models.Group{}, fmt.Errorf("MoveGroup: %w", err)
	}
	byID := make(map[string]models.Group, len(all))
	for _, g := range all {
		byID[g.ID] = g
	}
	// Walk UP from the proposed parent. If id appears on that path, the
	// parent is one of id's own descendants.
	seen := map[string]bool{}
	for cur, ok := byID[newParentID]; ok && !seen[cur.ID]; cur, ok = byID[cur.ParentID] {
		if cur.ID == id {
			return models.Group{}, ErrParentInSubtree
		}
		seen[cur.ID] = true
	}

	err = m.db.WithContext(ctx).Table(m.tables.Groups).Where("id = ?", id).
		Updates(map[string]any{"parent_id": gormstore.StringToNullable(newParentID), "updated_at": models.NowRFC3339()}).Error
	if err != nil {
		return models.Group{}, fmt.Errorf("MoveGroup: %w", err)
	}
	current.ParentID = newParentID
	return current, nil
}

// ListGroupChildren returns the direct children of a group (empty parentID
// returns the roots), id order.
func (m *userManager) ListGroupChildren(ctx context.Context, parentID string) ([]models.Group, error) {
	all, err := m.ListGroups(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("ListGroupChildren: %w", err)
	}
	out := []models.Group{}
	for _, g := range all {
		if g.ParentID == parentID {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// ListGroups returns every group, optionally filtered to one hierarchy,
// created_at-then-id order (matching the gateway Postgres store's ORDER BY
// created_at, id).
func (m *userManager) ListGroups(ctx context.Context, hierarchyID string) ([]models.Group, error) {
	q := m.db.WithContext(ctx).Table(m.tables.Groups)
	if hierarchyID != "" {
		q = q.Where("hierarchy_id = ?", hierarchyID)
	}
	var rows []gormstore.GroupRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListGroups: %w", err)
	}
	out := make([]models.Group, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.GroupFromRow(r))
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
func (m *userManager) ListDiscoverableGroups(ctx context.Context, query string) ([]models.Group, error) {
	all, err := m.ListGroups(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("ListDiscoverableGroups: %w", err)
	}
	q := strings.ToLower(query)
	out := []models.Group{}
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
