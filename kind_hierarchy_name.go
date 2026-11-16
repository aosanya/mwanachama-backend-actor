package mwanachamaactor

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/plaintext"
)

type namedRow struct {
	ID   string
	Name string
}

func (m *userManager) nameHeldBy(tx *gorm.DB, role, name, exceptID string, liveOnly bool) (bool, error) {
	q := m.from(tx, role).Select("id, name")
	if liveOnly {
		q = q.Where("retired_at = ?", "")
	}
	var rows []namedRow
	if err := q.Find(&rows).Error; err != nil {
		return false, err
	}
	key := plaintext.FoldDrawnName(name)
	for _, r := range rows {
		if r.ID != exceptID && plaintext.FoldDrawnName(r.Name) == key {
			return true, nil
		}
	}
	return false, nil
}

func (m *userManager) checkRoleKindNameFree(tx *gorm.DB, name, exceptID string) error {
	held, err := m.nameHeldBy(tx, roleRoleKind, name, exceptID, true)
	if err != nil {
		return fmt.Errorf("checkRoleKindNameFree: %w", err)
	}
	if held {
		return fmt.Errorf("%w: %q", ErrDuplicateRoleKindName, name)
	}
	return nil
}

func (m *userManager) checkHierarchyNameFree(tx *gorm.DB, name, exceptID string) error {
	held, err := m.nameHeldBy(tx, roleHierarchy, name, exceptID, false)
	if err != nil {
		return fmt.Errorf("checkHierarchyNameFree: %w", err)
	}
	if held {
		return fmt.Errorf("%w: %q", ErrDuplicateHierarchyName, name)
	}
	return nil
}
