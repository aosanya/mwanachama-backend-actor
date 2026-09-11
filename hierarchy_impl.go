// hierarchy_impl.go — Hierarchy/Level lifecycle for [userManager]. Ported
// from mwanachama-backend-api-gateway's internal/domain/chapter.Repository's
// Hierarchy/Level methods and internal/store/{memory,postgres}/hierarchy_*.go
// — DSN-1699 gap 1, resolved: these two types move here rather than staying
// behind in the gateway's own tables, because every row that references a
// Level (Group.LevelID, Group.AnchorLevelOverrideID, RoleKind.LevelID) is
// already stored in this package. That is also why DeleteLevel's two
// refusals — worn by a Group, scoped to by a RoleKind — can both be checked
// natively here in one transaction, restoring the RoleKind half of that
// check, which was silently dropped from the gateway's split-store version
// after role moved to actor and nobody re-wired it.
package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-actor/gormstore"
)

// CreateHierarchy stores a hierarchy, minting an id when empty.
func (m *userManager) CreateHierarchy(ctx context.Context, h Hierarchy) (Hierarchy, error) {
	var row gormstore.HierarchyRow
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		code, err := gormstore.NextCode(ctx, tx, m.tables.CodeSequences, "hierarchy", "H")
		if err != nil {
			return err
		}
		h.Code = code
		row = gormstore.HierarchyToRow(h)
		return tx.WithContext(ctx).Table(m.tables.Hierarchies).Create(&row).Error
	})
	if err != nil {
		return Hierarchy{}, fmt.Errorf("CreateHierarchy: %w", err)
	}
	return gormstore.HierarchyFromRow(row), nil
}

// ListHierarchies returns every hierarchy, id order.
func (m *userManager) ListHierarchies(ctx context.Context) ([]Hierarchy, error) {
	var rows []gormstore.HierarchyRow
	if err := m.db.WithContext(ctx).Table(m.tables.Hierarchies).Order("id").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListHierarchies: %w", err)
	}
	out := make([]Hierarchy, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.HierarchyFromRow(r))
	}
	return out, nil
}

// GetHierarchy returns a hierarchy by id. Returns [ErrHierarchyNotFound] if
// no matching hierarchy exists.
func (m *userManager) GetHierarchy(ctx context.Context, id string) (Hierarchy, error) {
	var row gormstore.HierarchyRow
	err := m.db.WithContext(ctx).Table(m.tables.Hierarchies).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Hierarchy{}, ErrHierarchyNotFound
	}
	if err != nil {
		return Hierarchy{}, fmt.Errorf("GetHierarchy: %w", err)
	}
	return gormstore.HierarchyFromRow(row), nil
}

// RenameHierarchy writes a hierarchy's name and nothing else.
func (m *userManager) RenameHierarchy(ctx context.Context, id, name string) (Hierarchy, error) {
	current, err := m.GetHierarchy(ctx, id)
	if err != nil {
		return Hierarchy{}, err
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Hierarchies).Where("id = ?", id).
		Update("name", name).Error; err != nil {
		return Hierarchy{}, fmt.Errorf("RenameHierarchy: %w", err)
	}
	current.Name = name
	return current, nil
}

// CreateLevel stores a level, minting an id when empty. Returns
// [ErrHierarchyNotFound] when l.HierarchyID names no hierarchy, and
// [ErrDuplicateDefaultAnchor] when l.IsDefaultAnchor is set and the
// hierarchy already has one — the Go-level half of that guarantee;
// gormstore.Migrate's partial unique index is the database-level half that
// stops a concurrent write from racing past this pre-check.
func (m *userManager) CreateLevel(ctx context.Context, l Level) (Level, error) {
	if _, err := m.GetHierarchy(ctx, l.HierarchyID); err != nil {
		return Level{}, err
	}
	if l.IsDefaultAnchor {
		var count int64
		if err := m.db.WithContext(ctx).Table(m.tables.Levels).
			Where("hierarchy_id = ? AND is_default_anchor", l.HierarchyID).
			Count(&count).Error; err != nil {
			return Level{}, fmt.Errorf("CreateLevel: %w", err)
		}
		if count > 0 {
			return Level{}, ErrDuplicateDefaultAnchor
		}
	}
	var row gormstore.LevelRow
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		code, err := gormstore.NextCode(ctx, tx, m.tables.CodeSequences, "level", "L")
		if err != nil {
			return err
		}
		l.Code = code
		row = gormstore.LevelToRow(l)
		return tx.WithContext(ctx).Table(m.tables.Levels).Create(&row).Error
	})
	if err != nil {
		return Level{}, fmt.Errorf("CreateLevel: %w", err)
	}
	return gormstore.LevelFromRow(row), nil
}

// GetLevel returns a level by id. Returns [ErrLevelNotFound] if no matching
// level exists.
func (m *userManager) GetLevel(ctx context.Context, id string) (Level, error) {
	row, err := m.findLevelRow(ctx, m.db, id)
	if err != nil {
		return Level{}, err
	}
	return gormstore.LevelFromRow(row), nil
}

func (m *userManager) findLevelRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.LevelRow, error) {
	var row gormstore.LevelRow
	err := tx.WithContext(ctx).Table(m.tables.Levels).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.LevelRow{}, ErrLevelNotFound
	}
	if err != nil {
		return gormstore.LevelRow{}, fmt.Errorf("findLevelRow: %w", err)
	}
	return row, nil
}

// RenameLevel writes a level's name and nothing else. Depth and HierarchyID
// are not reachable from here — see chapter.Repository.RenameLevel's doc,
// which this mirrors: a rung that can be re-numbered by a rename is not a
// rung.
func (m *userManager) RenameLevel(ctx context.Context, id, name string) (Level, error) {
	current, err := m.GetLevel(ctx, id)
	if err != nil {
		return Level{}, err
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Levels).Where("id = ?", id).
		Update("name", name).Error; err != nil {
		return Level{}, fmt.Errorf("RenameLevel: %w", err)
	}
	current.Name = name
	return current, nil
}

// DeleteLevel removes a rung, refusing the two states in which it is still
// load-bearing: [ErrLevelWornByGroups] when a Group's LevelID or
// AnchorLevelOverrideID names it, [ErrLevelHasRoleKinds] when a RoleKind's
// LevelID names it. Both checks and the delete run in one transaction.
func (m *userManager) DeleteLevel(ctx context.Context, id string) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := m.findLevelRow(ctx, tx, id); err != nil {
			return err
		}
		var wornCount int64
		if err := tx.WithContext(ctx).Table(m.tables.Groups).
			Where("(level_id = ? OR anchor_level_override = ?) AND deleted = ?", id, id, false).
			Count(&wornCount).Error; err != nil {
			return fmt.Errorf("DeleteLevel: %w", err)
		}
		if wornCount > 0 {
			return ErrLevelWornByGroups
		}
		var kindCount int64
		if err := tx.WithContext(ctx).Table(m.tables.RoleKinds).
			Where("level_id = ?", id).Count(&kindCount).Error; err != nil {
			return fmt.Errorf("DeleteLevel: %w", err)
		}
		if kindCount > 0 {
			return ErrLevelHasRoleKinds
		}
		if err := tx.WithContext(ctx).Table(m.tables.Levels).Where("id = ?", id).
			Delete(&gormstore.LevelRow{}).Error; err != nil {
			return fmt.Errorf("DeleteLevel: %w", err)
		}
		return nil
	})
}

// SetDefaultAnchor moves a hierarchy's anchor rung, clearing the old flag
// and setting the new one in one transaction — see
// chapter.Repository.SetDefaultAnchor's doc for why the atomicity is the
// point. Setting the rung that already carries the flag is a no-op.
func (m *userManager) SetDefaultAnchor(ctx context.Context, hierarchyID, levelID string) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		target, err := m.findLevelRow(ctx, tx, levelID)
		if err != nil {
			return err
		}
		if target.HierarchyID != hierarchyID {
			return ErrLevelNotFound
		}
		if target.IsDefaultAnchor {
			return nil
		}
		if err := tx.WithContext(ctx).Table(m.tables.Levels).
			Where("hierarchy_id = ? AND is_default_anchor", hierarchyID).
			Update("is_default_anchor", false).Error; err != nil {
			return fmt.Errorf("SetDefaultAnchor: %w", err)
		}
		if err := tx.WithContext(ctx).Table(m.tables.Levels).Where("id = ?", levelID).
			Update("is_default_anchor", true).Error; err != nil {
			return fmt.Errorf("SetDefaultAnchor: %w", err)
		}
		return nil
	})
}

// ListLevels returns the levels of a hierarchy, or — when hierarchyID is
// empty — every level across every hierarchy, depth-then-id order.
func (m *userManager) ListLevels(ctx context.Context, hierarchyID string) ([]Level, error) {
	q := m.db.WithContext(ctx).Table(m.tables.Levels)
	if hierarchyID != "" {
		q = q.Where("hierarchy_id = ?", hierarchyID)
	}
	var rows []gormstore.LevelRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListLevels: %w", err)
	}
	out := make([]Level, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.LevelFromRow(r))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Depth != out[j].Depth {
			return out[i].Depth < out[j].Depth
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// HierarchyExists reports whether id names a hierarchy — satisfies
// mwanachama-backend-actor/routes.HierarchyChecker directly.
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

// LevelInHierarchy reports whether levelID names a level belonging to
// hierarchyID — satisfies routes.HierarchyChecker directly.
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
