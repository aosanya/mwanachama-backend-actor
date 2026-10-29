package mwanachamaactor

import "gorm.io/gorm"

const outsiderLookupChunk = 500

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

func (m *userManager) outsidersAmong(tx *gorm.DB, actorIDs []string) (map[string]bool, error) {
	out := map[string]bool{}
	for start := 0; start < len(actorIDs); start += outsiderLookupChunk {
		chunk := actorIDs[start:min(start+outsiderLookupChunk, len(actorIDs))]
		ordinarySeats := m.activeSeats(tx).
			Where("actor_id IN ?", chunk).
			Where("kind_id NOT IN (?)", m.outsiderKinds(tx))
		var found []string
		err := m.activeSeats(tx).
			Where("actor_id IN ?", chunk).
			Where("kind_id IN (?)", m.outsiderKinds(tx)).
			Where("actor_id NOT IN (?)", ordinarySeats).
			Distinct().
			Pluck("actor_id", &found).Error
		if err != nil {
			return nil, err
		}
		for _, id := range found {
			out[id] = true
		}
	}
	return out, nil
}

func (m *userManager) isOutsider(tx *gorm.DB, actorID string) (bool, error) {
	outsiders, err := m.outsidersAmong(tx, []string{actorID})
	if err != nil {
		return false, err
	}
	return outsiders[actorID], nil
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
