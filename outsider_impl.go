package mwanachamaactor

import "gorm.io/gorm"

func (m *userManager) outsiderSeats(tx *gorm.DB) *gorm.DB {
	outsiderKinds := tx.Table(m.table(roleRoleKind)).Select("id").Where("outsider = ?", true)
	return tx.Table(m.table(roleActorRoleAssignment)).Select("actor_id").
		Where("active").
		Where("kind_id IN (?)", outsiderKinds)
}

func (m *userManager) isOutsider(tx *gorm.DB, actorID string) (bool, error) {
	var n int64
	if err := m.outsiderSeats(tx).Where("actor_id = ?", actorID).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (m *userManager) refuseOutsider(tx *gorm.DB, actorID string) error {
	outsider, err := m.isOutsider(tx, actorID)
	if err != nil {
		return err
	}
	if outsider {
		return ErrOutsiderRole
	}
	return nil
}
