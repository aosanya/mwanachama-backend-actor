package mwanachamaactor

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

const groupNameKeyColumn = "name_key"

func groupNameKey(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

func (m *userManager) checkGroupNameFree(ctx context.Context, name, exceptID string) error {
	key := groupNameKey(name)
	q := m.store.Query(ctx, roleGroup).
		Where(groupNameKeyColumn+" = ?", key).
		Where("deleted = ?", false)
	if exceptID != "" {
		q = q.Where("id <> ?", exceptID)
	}
	var holders int64
	if err := q.Count(&holders).Error; err != nil {
		return fmt.Errorf("checkGroupNameFree: %w", err)
	}
	if holders > 0 {
		return fmt.Errorf("%w: %q", ErrDuplicateGroupName, name)
	}
	return nil
}

func isGroupNameIndex(s string) bool {
	return strings.Contains(s, groupNameKeyColumn)
}

func groupNameIndexName(table string) string {
	return table + "_name_key_uniq"
}

func groupNameIndexExists(db *gorm.DB, table string) (bool, error) {
	var found []string
	query := `select indexname from pg_indexes where tablename = ? and indexname = ?`
	if db.Dialector.Name() != "postgres" {
		query = `select name from sqlite_master where type = 'index' and tbl_name = ? and name = ?`
	}
	if err := db.Raw(query, table, groupNameIndexName(table)).Scan(&found).Error; err != nil {
		return false, fmt.Errorf("actor: read the indexes of %s: %w", table, err)
	}
	return len(found) > 0, nil
}

func backfillGroupNameKeys(db *gorm.DB, s *spec.Spec) error {
	table := tableFor(s, roleGroup)
	type named struct {
		ID   string
		Name string
	}
	var rows []named
	if err := db.Table(table).Select("id, name").
		Where(groupNameKeyColumn + " = '' OR " + groupNameKeyColumn + " IS NULL").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("actor: read the group names in %s: %w", table, err)
	}
	for _, r := range rows {
		err := db.Table(table).Where("id = ?", r.ID).Update(groupNameKeyColumn, groupNameKey(r.Name)).Error
		if err != nil {
			return fmt.Errorf("actor: set the name key of group %s: %w", r.ID, err)
		}
	}
	return nil
}

func syncGroupNameIndex(db *gorm.DB, s *spec.Spec) error {
	switch db.Dialector.Name() {
	case "postgres", "sqlite":
	default:
		return nil
	}
	table := tableFor(s, roleGroup)
	if table == "" {
		return nil
	}
	name := groupNameIndexName(table)
	if len(name) > spec.MaxIdentifier {
		return fmt.Errorf("actor: the group-name index needs %d bytes as %q, and the database truncates at %d",
			len(name), name, spec.MaxIdentifier)
	}
	var shared int64
	query := fmt.Sprintf(
		"select count(*) from (select %[2]s from %[1]s where deleted = false and %[2]s <> '' group by %[2]s having count(*) > 1) shared",
		table, groupNameKeyColumn,
	)
	err := db.Raw(query).Scan(&shared).Error
	if err != nil {
		return fmt.Errorf("actor: read the shared group names in %s: %w", table, err)
	}
	if shared > 0 {
		slog.Warn("actor: groups share a name, so the unique group-name index is not created until they are renamed",
			"table", table, "names", shared)
		return nil
	}
	stmt := fmt.Sprintf(
		"create unique index if not exists %s on %s (%s) where deleted = false and %s <> ''",
		name, table, groupNameKeyColumn, groupNameKeyColumn,
	)
	if err := db.Exec(stmt).Error; err != nil {
		return fmt.Errorf("actor: %s: %w", name, err)
	}
	return nil
}

func groupsWithoutNameKey(db *gorm.DB, s *spec.Spec) (bool, error) {
	table := tableFor(s, roleGroup)
	var unkeyed int64
	err := db.Table(table).
		Where(groupNameKeyColumn + " = '' OR " + groupNameKeyColumn + " IS NULL").
		Count(&unkeyed).Error
	if err != nil {
		return false, fmt.Errorf("actor: read the group name keys in %s: %w", table, err)
	}
	return unkeyed > 0, nil
}
