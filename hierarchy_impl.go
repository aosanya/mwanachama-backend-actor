package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

func (m *userManager) CreateHierarchy(ctx context.Context, h Hierarchy) (Hierarchy, error) {
	h.ID = newSpecID()
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		code, err := m.nextCode(ctx, tx, "hierarchy", "H")
		if err != nil {
			return err
		}
		h.Code = code
		row, err := m.store.Row(roleHierarchy, h)
		if err != nil {
			return err
		}
		if err := m.from(tx.WithContext(ctx), roleHierarchy).Create(row).Error; err != nil {
			return classifyDuplicateID(err)
		}
		return nil
	})
	if err != nil {
		return Hierarchy{}, fmt.Errorf("CreateHierarchy: %w", classifyConflict(err))
	}
	return h, nil
}

func (m *userManager) ListHierarchies(ctx context.Context) ([]Hierarchy, error) {
	q := m.store.Query(ctx, roleHierarchy).Order("id")
	out, err := specstore.List[Hierarchy](m.store, q, roleHierarchy)
	if err != nil {
		return nil, fmt.Errorf("ListHierarchies: %w", err)
	}
	return out, nil
}

func (m *userManager) GetHierarchy(ctx context.Context, id string) (Hierarchy, error) {
	var out Hierarchy
	q := m.store.Query(ctx, roleHierarchy).Where("id = ?", id)
	if err := m.store.Take(q, roleHierarchy, &out, ErrHierarchyNotFound); err != nil {
		if errors.Is(err, ErrHierarchyNotFound) {
			return Hierarchy{}, ErrHierarchyNotFound
		}
		return Hierarchy{}, fmt.Errorf("GetHierarchy: %w", err)
	}
	return out, nil
}

func (m *userManager) RenameHierarchy(ctx context.Context, id, name string) (Hierarchy, error) {
	current, err := m.GetHierarchy(ctx, id)
	if err != nil {
		return Hierarchy{}, err
	}
	if err := m.store.Query(ctx, roleHierarchy).Where("id = ?", id).
		Update("name", name).Error; err != nil {
		return Hierarchy{}, fmt.Errorf("RenameHierarchy: %w", err)
	}
	current.Name = name
	return current, nil
}

func (m *userManager) CreateLevel(ctx context.Context, l Level) (Level, error) {
	if _, err := m.GetHierarchy(ctx, l.HierarchyID); err != nil {
		return Level{}, err
	}
	if l.IsDefaultAnchor {
		var count int64
		if err := m.store.Query(ctx, roleLevel).
			Where("hierarchy_id = ? AND is_default_anchor", l.HierarchyID).
			Count(&count).Error; err != nil {
			return Level{}, fmt.Errorf("CreateLevel: %w", err)
		}
		if count > 0 {
			return Level{}, ErrDuplicateDefaultAnchor
		}
	}
	l.ID = newSpecID()
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		code, err := m.nextCode(ctx, tx, "level", "L")
		if err != nil {
			return err
		}
		l.Code = code
		row, err := m.store.Row(roleLevel, l)
		if err != nil {
			return err
		}
		if err := m.from(tx.WithContext(ctx), roleLevel).Create(row).Error; err != nil {
			return classifyDuplicateID(err)
		}
		return nil
	})
	if err != nil {
		return Level{}, fmt.Errorf("CreateLevel: %w", classifyConflict(err))
	}
	return l, nil
}

func (m *userManager) GetLevel(ctx context.Context, id string) (Level, error) {
	return m.levelIn(m.db.WithContext(ctx), id)
}

func (m *userManager) levelIn(tx *gorm.DB, id string) (Level, error) {
	var out Level
	q := m.from(tx, roleLevel).Where("id = ?", id)
	if err := m.store.Take(q, roleLevel, &out, ErrLevelNotFound); err != nil {
		if errors.Is(err, ErrLevelNotFound) {
			return Level{}, ErrLevelNotFound
		}
		return Level{}, fmt.Errorf("findLevel: %w", err)
	}
	return out, nil
}

func (m *userManager) RenameLevel(ctx context.Context, id, name string) (Level, error) {
	current, err := m.GetLevel(ctx, id)
	if err != nil {
		return Level{}, err
	}
	if err := m.store.Query(ctx, roleLevel).Where("id = ?", id).
		Update("name", name).Error; err != nil {
		return Level{}, fmt.Errorf("RenameLevel: %w", err)
	}
	current.Name = name
	return current, nil
}

func (m *userManager) DeleteLevel(ctx context.Context, id string) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := m.levelIn(tx, id); err != nil {
			return err
		}
		var wornCount int64
		if err := m.from(tx, roleGroup).
			Where("(level_id = ? OR anchor_level_override_id = ?) AND deleted = ?", id, id, false).
			Count(&wornCount).Error; err != nil {
			return fmt.Errorf("DeleteLevel: %w", err)
		}
		if wornCount > 0 {
			return ErrLevelWornByGroups
		}
		var kindCount int64
		if err := m.from(tx, roleRoleKind).
			Where("level_id = ?", id).Count(&kindCount).Error; err != nil {
			return fmt.Errorf("DeleteLevel: %w", err)
		}
		if kindCount > 0 {
			return ErrLevelHasRoleKinds
		}
		if err := m.from(tx, roleLevel).Where("id = ?", id).
			Delete(map[string]any{}).Error; err != nil {
			return fmt.Errorf("DeleteLevel: %w", err)
		}
		return nil
	})
}

func (m *userManager) SetDefaultAnchor(ctx context.Context, hierarchyID, levelID string) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		target, err := m.levelIn(tx, levelID)
		if err != nil {
			return err
		}
		if target.HierarchyID != hierarchyID {
			return ErrLevelNotFound
		}
		if target.IsDefaultAnchor {
			return nil
		}
		if err := m.from(tx, roleLevel).
			Where("hierarchy_id = ? AND is_default_anchor", hierarchyID).
			Update("is_default_anchor", false).Error; err != nil {
			return fmt.Errorf("SetDefaultAnchor: %w", err)
		}
		if err := m.from(tx, roleLevel).Where("id = ?", levelID).
			Update("is_default_anchor", true).Error; err != nil {
			return fmt.Errorf("SetDefaultAnchor: %w", err)
		}
		return nil
	})
}

func (m *userManager) ListLevels(ctx context.Context, hierarchyID string) ([]Level, error) {
	q := m.store.Query(ctx, roleLevel)
	if hierarchyID != "" {
		q = q.Where("hierarchy_id = ?", hierarchyID)
	}
	out, err := specstore.List[Level](m.store, q, roleLevel)
	if err != nil {
		return nil, fmt.Errorf("ListLevels: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Depth != out[j].Depth {
			return out[i].Depth < out[j].Depth
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (m *userManager) HierarchyExists(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, nil
	}
	_, err := m.GetHierarchy(ctx, id)
	if errors.Is(err, ErrHierarchyNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m *userManager) LevelInHierarchy(ctx context.Context, levelID, hierarchyID string) (bool, error) {
	if levelID == "" {
		return false, nil
	}
	lvl, err := m.GetLevel(ctx, levelID)
	if errors.Is(err, ErrLevelNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return lvl.HierarchyID == hierarchyID, nil
}
