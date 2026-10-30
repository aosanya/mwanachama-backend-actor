package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

func (m *userManager) CreateGroupType(ctx context.Context, g GroupType) (GroupType, error) {
	if strings.TrimSpace(g.Name) == "" {
		return GroupType{}, ErrInvalidGroupType
	}
	if _, err := m.GetHierarchy(ctx, g.HierarchyID); err != nil {
		return GroupType{}, err
	}
	if g.LevelID != "" {
		if _, err := m.GetLevel(ctx, g.LevelID); err != nil {
			return GroupType{}, err
		}
	}
	if g.Singular == "" {
		g.Singular = g.Name
	}
	if g.Plural == "" {
		g.Plural = g.Name + "s"
	}
	g.ID = newSpecID()
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		code, err := nextCode(ctx, tx, m.table(roleCodeSequence), "grouptype", "GT")
		if err != nil {
			return err
		}
		g.Code = code
		row, err := encode(m.object(roleGroupType), g)
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Table(m.table(roleGroupType)).Create(row).Error; err != nil {
			return classifyDuplicateID(err)
		}
		return nil
	})
	if err != nil {
		return GroupType{}, fmt.Errorf("CreateGroupType: %w", classifyConflict(err))
	}
	return g, nil
}

func (m *userManager) GetGroupType(ctx context.Context, id string) (GroupType, error) {
	return m.groupTypeIn(m.db.WithContext(ctx), id)
}

func (m *userManager) groupTypeIn(tx *gorm.DB, id string) (GroupType, error) {
	var out GroupType
	q := tx.Table(m.table(roleGroupType)).Where("id = ?", id)
	if err := m.store.Take(q, roleGroupType, &out, ErrGroupTypeNotFound); err != nil {
		if errors.Is(err, ErrGroupTypeNotFound) {
			return GroupType{}, ErrGroupTypeNotFound
		}
		return GroupType{}, fmt.Errorf("findGroupType: %w", err)
	}
	return out, nil
}

func (m *userManager) ListGroupTypes(ctx context.Context, hierarchyID string) ([]GroupType, error) {
	q := m.store.Query(ctx, roleGroupType)
	if hierarchyID != "" {
		q = q.Where("hierarchy_id = ?", hierarchyID)
	}
	out, err := specstore.List[GroupType](m.store, q, roleGroupType)
	if err != nil {
		return nil, fmt.Errorf("ListGroupTypes: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (m *userManager) EditGroupType(ctx context.Context, id, name, singular, plural, levelID string) (GroupType, error) {
	current, err := m.GetGroupType(ctx, id)
	if err != nil {
		return GroupType{}, err
	}
	if strings.TrimSpace(name) == "" {
		return GroupType{}, ErrInvalidGroupType
	}
	if levelID != "" {
		if _, err := m.GetLevel(ctx, levelID); err != nil {
			return GroupType{}, err
		}
	}
	if singular == "" {
		singular = name
	}
	if plural == "" {
		plural = name + "s"
	}
	cols := map[string]any{
		"name": name, "singular": singular, "plural": plural, "level_id": levelID,
	}
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(m.table(roleGroupType)).Where("id = ?", id).Updates(cols).Error; err != nil {
			return err
		}
		if name == current.Name {
			return nil
		}
		return tx.Table(m.table(roleGroup)).
			Where("hierarchy_id = ? AND node_type = ?", current.HierarchyID, current.Name).
			Update("node_type", name).Error
	})
	if err != nil {
		return GroupType{}, fmt.Errorf("EditGroupType: %w", err)
	}
	current.Name, current.Singular, current.Plural, current.LevelID = name, singular, plural, levelID
	return current, nil
}

func (m *userManager) DeleteGroupType(ctx context.Context, id string) error {
	current, err := m.GetGroupType(ctx, id)
	if err != nil {
		return err
	}
	var worn int64
	if err := m.store.Query(ctx, roleGroup).
		Where("node_type = ?", current.Name).Count(&worn).Error; err != nil {
		return fmt.Errorf("DeleteGroupType: %w", err)
	}
	if worn > 0 {
		return ErrGroupTypeWornByGroups
	}
	if err := m.store.Query(ctx, roleGroupType).Where("id = ?", id).
		Delete(map[string]any{}).Error; err != nil {
		return fmt.Errorf("DeleteGroupType: %w", err)
	}
	return nil
}
