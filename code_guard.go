package mwanachamaactor

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

const codeCollisionSkips = 200

func (m *userManager) nextCode(ctx context.Context, tx *gorm.DB, entityType, prefix string) (string, error) {
	role := roleOfEntityType(entityType)
	for skipped := 0; skipped < codeCollisionSkips; skipped++ {
		code, err := nextCode(ctx, tx, m.table(roleCodeSequence), m.store.RunID(), entityType, prefix)
		if err != nil {
			return "", err
		}
		taken, err := m.codeTaken(ctx, tx, role, code)
		if err != nil {
			return "", err
		}
		if !taken {
			return code, nil
		}
	}
	return "", fmt.Errorf("nextCode: %s: %d codes in a row were already in use: %w", entityType, codeCollisionSkips, ErrConflict)
}

func roleOfEntityType(entityType string) string {
	for _, c := range codedRoles {
		if c.entityType == entityType {
			return c.role
		}
	}
	return ""
}

func (m *userManager) codeTaken(ctx context.Context, tx *gorm.DB, role, code string) (bool, error) {
	if role == "" {
		return false, nil
	}
	var rows []map[string]any
	err := inRun(tx.WithContext(ctx).Table(m.table(role)), m.store.RunID()).
		Where("code = ?", code).Limit(1).Find(&rows).Error
	if err != nil {
		return false, fmt.Errorf("nextCode: look for %s: %w", code, err)
	}
	return len(rows) > 0, nil
}
