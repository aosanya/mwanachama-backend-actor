package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func (m *userManager) CreateGroup(ctx context.Context, g Group) (Group, error) {
	if g.Name == "" {
		return Group{}, fmt.Errorf("%w: Group.Name is required", ErrInvalidGroup)
	}
	properties := models.DefaultGroupProperties()
	if err := models.ValidateAttributes(properties, g.Attributes); err != nil {
		return Group{}, fmt.Errorf("%w: %v", ErrInvalidGroup, err)
	}
	if err := m.checkUniqueAttributes(ctx, m.table(roleGroup), properties, g.Attributes); err != nil {
		return Group{}, err
	}

	g.ID = newSpecID()
	g.CreatedAt = NowRFC3339()
	g.LastUpdated = g.CreatedAt
	g.Deleted = false

	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		code, err := nextCode(ctx, tx, m.table(roleCodeSequence), "group", "G")
		if err != nil {
			return err
		}
		g.Code = code
		row, err := encode(m.object(roleGroup), g)
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Table(m.table(roleGroup)).Create(row).Error; err != nil {
			return classifyDuplicateID(err)
		}
		return nil
	})
	if err != nil {
		return Group{}, fmt.Errorf("CreateGroup: %w", err)
	}
	return g, nil
}

func (m *userManager) GetGroup(ctx context.Context, id string) (Group, error) {
	var out Group
	q := m.store.Query(ctx, roleGroup).Where("id = ?", id).Where("deleted = ?", false)
	if err := m.store.Take(q, roleGroup, &out, ErrGroupNotFound); err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			return Group{}, ErrGroupNotFound
		}
		return Group{}, fmt.Errorf("GetGroup: %w", err)
	}
	return out, nil
}

func (m *userManager) EditGroup(ctx context.Context, id string, e GroupEdit) (Group, error) {
	current, err := m.GetGroup(ctx, id)
	if err != nil {
		return Group{}, err
	}
	now := NowRFC3339()
	err = m.store.Query(ctx, roleGroup).Where("id = ?", id).
		Updates(map[string]any{
			"name":                     e.Name,
			"node_type":                e.NodeType,
			"anchor_level_override_id": e.AnchorLevelOverrideID,
			"last_updated":             now,
		}).Error
	if err != nil {
		return Group{}, fmt.Errorf("EditGroup: %w", err)
	}
	current.Name = e.Name
	current.NodeType = e.NodeType
	current.AnchorLevelOverrideID = e.AnchorLevelOverrideID
	current.LastUpdated = now
	return current, nil
}

func (m *userManager) MoveGroup(ctx context.Context, id, newParentID string) (Group, error) {
	if newParentID == id {
		return Group{}, ErrParentIsSelf
	}
	current, err := m.GetGroup(ctx, id)
	if err != nil {
		return Group{}, err
	}
	if current.ParentID == "" {
		return Group{}, ErrRootCannotMove
	}
	if newParentID != "" {
		if _, err := m.GetGroup(ctx, newParentID); err != nil {
			if errors.Is(err, ErrGroupNotFound) {
				return Group{}, ErrParentNotFound
			}
			return Group{}, err
		}
	}

	all, err := m.ListGroups(ctx, "")
	if err != nil {
		return Group{}, fmt.Errorf("MoveGroup: %w", err)
	}
	byID := make(map[string]Group, len(all))
	for _, g := range all {
		byID[g.ID] = g
	}
	seen := map[string]bool{}
	for cur, ok := byID[newParentID]; ok && !seen[cur.ID]; cur, ok = byID[cur.ParentID] {
		if cur.ID == id {
			return Group{}, ErrParentInSubtree
		}
		seen[cur.ID] = true
	}

	now := NowRFC3339()
	err = m.store.Query(ctx, roleGroup).Where("id = ?", id).
		Updates(map[string]any{"parent_id": newParentID, "last_updated": now}).Error
	if err != nil {
		return Group{}, fmt.Errorf("MoveGroup: %w", err)
	}
	current.ParentID = newParentID
	current.LastUpdated = now
	return current, nil
}

func (m *userManager) ListGroupChildren(ctx context.Context, parentID string) ([]Group, error) {
	all, err := m.ListGroups(ctx, "")
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

func (m *userManager) ListGroups(ctx context.Context, hierarchyID string) ([]Group, error) {
	q := m.store.Query(ctx, roleGroup).Where("deleted = ?", false)
	if hierarchyID != "" {
		q = q.Where("hierarchy_id = ?", hierarchyID)
	}
	out, err := specstore.List[Group](m.store, q, roleGroup)
	if err != nil {
		return nil, fmt.Errorf("ListGroups: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (m *userManager) ListDiscoverableGroups(ctx context.Context, query string) ([]Group, error) {
	all, err := m.ListGroups(ctx, "")
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
