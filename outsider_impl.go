package mwanachamaactor

import "gorm.io/gorm"

func (m *userManager) outsiderKinds(tx *gorm.DB) *gorm.DB {
	return tx.Table(m.table(roleRoleKind)).Select("id").Where("outsider = ?", true)
}

func (m *userManager) activeSeats(tx *gorm.DB) *gorm.DB {
	return tx.Table(m.table(roleActorRoleAssignment)).Select("actor_id").Where("active")
}

func (m *userManager) outsiderActors(tx *gorm.DB) *gorm.DB {
	ordinarySeats := m.activeSeats(tx).Where("kind_id NOT IN (?)", m.outsiderKinds(tx))
	return m.activeSeats(tx).
		Where("kind_id IN (?)", m.outsiderKinds(tx)).
		Where("actor_id NOT IN (?)", ordinarySeats)
}

func (m *userManager) isOutsider(tx *gorm.DB, actorID string) (bool, error) {
	var n int64
	if err := m.outsiderActors(tx).Where("actor_id = ?", actorID).Count(&n).Error; err != nil {
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
