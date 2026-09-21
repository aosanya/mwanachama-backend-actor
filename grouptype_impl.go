// grouptype_impl.go — GroupType lifecycle for [userManager]. Moved here from
// mwanachama-backend-agency on 2026-09-17 (AGD-017), which had briefly
// carried it after deleting its own Department type. Same reasoning as
// Hierarchy/Level's own move in DSN-1699 gap 1: the rows that reference a
// GroupType (Group.NodeType) live in this package, so the vocabulary for the
// operational structure belongs beside the structure, not in a design
// document that has no dependency on it.
package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-actor/gormstore"
)

// CreateGroupType stores a group type, minting an id when empty and a
// "GT-<n>" Code via NextCode. Returns [ErrHierarchyNotFound] when
// g.HierarchyID names no hierarchy, [ErrLevelNotFound] when a non-empty
// g.LevelID names no level, and [ErrInvalidGroupType] when Name is blank.
//
// Singular and Plural fall back to Name and Name+"s" when left empty, so a
// caller with only one word for the type gets a usable set rather than
// blanks a surface would render as nothing.
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
	var row gormstore.GroupTypeRow
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		code, err := gormstore.NextCode(ctx, tx, m.tables.CodeSequences, "grouptype", "GT")
		if err != nil {
			return err
		}
		g.Code = code
		row = gormstore.GroupTypeToRow(g)
		return tx.WithContext(ctx).Table(m.tables.GroupTypes).Create(&row).Error
	})
	if err != nil {
		return GroupType{}, fmt.Errorf("CreateGroupType: %w", err)
	}
	return gormstore.GroupTypeFromRow(row), nil
}

// GetGroupType returns a group type by id. Returns [ErrGroupTypeNotFound] if
// no matching row exists.
func (m *userManager) GetGroupType(ctx context.Context, id string) (GroupType, error) {
	row, err := m.findGroupTypeRow(ctx, m.db, id)
	if err != nil {
		return GroupType{}, err
	}
	return gormstore.GroupTypeFromRow(row), nil
}

// ListGroupTypes returns every group type on hierarchyID, ordered by Code
// then ID — the same stable, creation-order shape ListLevels returns.
func (m *userManager) ListGroupTypes(ctx context.Context, hierarchyID string) ([]GroupType, error) {
	var rows []gormstore.GroupTypeRow
	q := m.db.WithContext(ctx).Table(m.tables.GroupTypes)
	if hierarchyID != "" {
		q = q.Where("hierarchy_id = ?", hierarchyID)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListGroupTypes: %w", err)
	}
	out := make([]GroupType, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.GroupTypeFromRow(r))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// EditGroupType writes Name/Singular/Plural/LevelID and nothing else — Code,
// ID and HierarchyID are never editable, matching RenameLevel's own
// narrowness — except that a rename also carries every Group of this type in
// its hierarchy (Group.NodeType = the old name) over to the new name in the
// same transaction. NodeType is free text matched by name, so without that a
// renamed type would orphan its Groups and DeleteGroupType's worn-by-Groups
// check, which reads the current name, would no longer see them. Empty Singular/Plural fall back to Name the same way
// CreateGroupType's do, so clearing a label restores the default rather than
// blanking the row.
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
		if err := tx.Table(m.tables.GroupTypes).Where("id = ?", id).Updates(cols).Error; err != nil {
			return err
		}
		if name == current.Name {
			return nil
		}
		return tx.Table(m.tables.Groups).
			Where("hierarchy_id = ? AND node_type = ?", current.HierarchyID, current.Name).
			Update("node_type", name).Error
	})
	if err != nil {
		return GroupType{}, fmt.Errorf("EditGroupType: %w", err)
	}
	current.Name, current.Singular, current.Plural, current.LevelID = name, singular, plural, levelID
	return current, nil
}

// DeleteGroupType removes a group type. Returns [ErrGroupTypeWornByGroups]
// while any Group still names it in NodeType — the same refusal DeleteLevel
// makes for a level still worn by a Group, and the reason NodeType's
// free-text-ness is tolerable: the check is by name, so it holds even for
// rows written by a caller that knows nothing about this type.
func (m *userManager) DeleteGroupType(ctx context.Context, id string) error {
	current, err := m.GetGroupType(ctx, id)
	if err != nil {
		return err
	}
	var worn int64
	if err := m.db.WithContext(ctx).Table(m.tables.Groups).
		Where("node_type = ?", current.Name).Count(&worn).Error; err != nil {
		return fmt.Errorf("DeleteGroupType: %w", err)
	}
	if worn > 0 {
		return ErrGroupTypeWornByGroups
	}
	if err := m.db.WithContext(ctx).Table(m.tables.GroupTypes).Where("id = ?", id).
		Delete(&gormstore.GroupTypeRow{}).Error; err != nil {
		return fmt.Errorf("DeleteGroupType: %w", err)
	}
	return nil
}

func (m *userManager) findGroupTypeRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.GroupTypeRow, error) {
	var row gormstore.GroupTypeRow
	err := tx.WithContext(ctx).Table(m.tables.GroupTypes).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.GroupTypeRow{}, ErrGroupTypeNotFound
	}
	if err != nil {
		return gormstore.GroupTypeRow{}, fmt.Errorf("findGroupTypeRow: %w", err)
	}
	return row, nil
}
