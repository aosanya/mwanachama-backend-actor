package mwanachamaactor

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func Provision(db *gorm.DB, s *spec.Spec) error {
	ready, err := upToDate(db, s)
	if err != nil {
		return err
	}
	if ready {
		return nil
	}
	if err := renameLegacy(db, s); err != nil {
		return err
	}
	if err := addColumnsDeclaredSince(db, s); err != nil {
		return err
	}
	if err := spec.Migrate(db, s); err != nil {
		return err
	}
	if err := syncDefaultAnchorIndex(db, s); err != nil {
		return err
	}
	return seed(db, s)
}

func upToDate(db *gorm.DB, s *spec.Spec) (bool, error) {
	declared := make([]string, 0, len(s.Objects))
	legacy := make([]string, 0, len(s.Objects))
	for _, o := range s.Objects {
		declared = append(declared, s.TableFor(o))
		for _, name := range legacyTables(s, o) {
			if name != s.TableFor(o) {
				legacy = append(legacy, name)
			}
		}
	}

	held, err := tablesNamed(db, append(append([]string{}, declared...), legacy...))
	if err != nil {
		return false, err
	}
	for _, name := range declared {
		if !held[name] {
			return false, nil
		}
	}
	for _, name := range legacy {
		if held[name] {
			return false, nil
		}
	}
	missing, err := columnsDeclaredSinceMissing(db, s)
	if err != nil {
		return false, err
	}
	if missing {
		return false, nil
	}

	for _, c := range codedRoles {
		var uncoded int64
		if err := db.Table(tableFor(s, c.role)).
			Where("code = '' OR code IS NULL").Count(&uncoded).Error; err != nil {
			return false, fmt.Errorf("actor: read the codes in %s: %w", tableFor(s, c.role), err)
		}
		if uncoded > 0 {
			return false, nil
		}
	}
	return true, nil
}

func tablesNamed(db *gorm.DB, names []string) (map[string]bool, error) {
	query := `select table_name from information_schema.tables
		where table_schema = current_schema() and table_name in ?`
	if db.Dialector.Name() != "postgres" {
		query = `select name from sqlite_master where type = 'table' and name in ?`
	}
	var found []string
	if err := db.Raw(query, names).Scan(&found).Error; err != nil {
		return nil, fmt.Errorf("actor: read the table set: %w", err)
	}
	held := make(map[string]bool, len(found))
	for _, name := range found {
		held[name] = true
	}
	return held, nil
}

func tableFor(s *spec.Spec, role string) string {
	o, ok := s.ByRole(role)
	if !ok {
		return ""
	}
	return s.TableFor(o)
}

func mustObject(s *spec.Spec, role string) spec.Object {
	o, ok := s.ByRole(role)
	if !ok {
		panic("actor: the spec fills no object for the role " + role)
	}
	return o
}

func objectTable(o spec.Object) string {
	if o.Table != "" {
		return o.Table
	}
	return o.Name
}

func legacyTables(s *spec.Spec, o spec.Object) []string {
	if s.MountName() != spec.DefaultMount {
		return nil
	}
	return []string{s.Instance + "_" + objectTable(o)}
}

var legacyColumns = map[string]string{
	"updated_at":            "last_updated",
	"anchor_level_override": "anchor_level_override_id",
}

func renameLegacy(db *gorm.DB, s *spec.Spec) error {
	m := db.Migrator()
	for _, o := range s.Objects {
		declared := s.TableFor(o)

		for _, legacy := range legacyTables(s, o) {
			if legacy == declared || !m.HasTable(legacy) {
				continue
			}
			if m.HasTable(declared) {
				stranded, err := rowCount(db, legacy)
				if err != nil {
					return err
				}
				if stranded > 0 {
					return fmt.Errorf("actor: %s and %s both exist and %s still holds %d row(s), which no read would ever reach again",
						legacy, declared, legacy, stranded)
				}
				continue
			}
			if err := db.Exec(fmt.Sprintf("alter table %s rename to %s", legacy, declared)).Error; err != nil {
				return fmt.Errorf("actor: rename %s to %s: %w", legacy, declared, err)
			}
			if err := dropLegacyIndexes(db, declared); err != nil {
				return err
			}
		}

		if !m.HasTable(declared) {
			continue
		}
		if err := renameColumns(db, declared, o); err != nil {
			return err
		}
		if err := adoptNullParents(db, declared, o); err != nil {
			return err
		}
	}
	return nil
}

func renameColumns(db *gorm.DB, table string, o spec.Object) error {
	for legacy, want := range legacyColumns {
		if !declaresColumn(o, want) {
			continue
		}
		held, err := hasColumn(db, table, legacy)
		if err != nil {
			return err
		}
		declared, err := hasColumn(db, table, want)
		if err != nil {
			return err
		}
		if !held || declared {
			continue
		}
		stmt := fmt.Sprintf("alter table %s rename column %s to %s", table, legacy, want)
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("actor: rename %s.%s: %w", table, legacy, err)
		}
	}
	return nil
}

func adoptNullParents(db *gorm.DB, table string, o spec.Object) error {
	if !declaresColumn(o, "parent_id") {
		return nil
	}
	held, err := hasColumn(db, table, "parent_id")
	if err != nil {
		return err
	}
	if !held {
		return nil
	}
	if err := db.Exec(fmt.Sprintf("update %s set parent_id = '' where parent_id is null", table)).Error; err != nil {
		return fmt.Errorf("actor: adopt the rootless groups in %s: %w", table, err)
	}
	return nil
}

func dropLegacyIndexes(db *gorm.DB, table string) error {
	var names []string
	switch db.Dialector.Name() {
	case "postgres":
		if err := db.Raw(`select indexname from pg_indexes where tablename = ?`, table).
			Scan(&names).Error; err != nil {
			return fmt.Errorf("actor: read indexes of %s: %w", table, err)
		}
	default:
		if err := db.Raw(`select name from sqlite_master where type = 'index' and tbl_name = ?`, table).
			Scan(&names).Error; err != nil {
			return fmt.Errorf("actor: read indexes of %s: %w", table, err)
		}
	}

	for _, name := range names {
		if !strings.HasPrefix(name, "idx_") && !strings.HasSuffix(name, "_attr_phone_uniq") &&
			!strings.HasSuffix(name, "_attr_email_uniq") && !strings.HasSuffix(name, "_one_default_anchor") {
			continue
		}
		if err := db.Exec("drop index if exists " + name).Error; err != nil {
			return fmt.Errorf("actor: drop index %s: %w", name, err)
		}
	}
	return nil
}

func syncDefaultAnchorIndex(db *gorm.DB, s *spec.Spec) error {
	switch db.Dialector.Name() {
	case "postgres", "sqlite":
	default:
		return nil
	}
	table := tableFor(s, roleLevel)
	if table == "" {
		return nil
	}
	name := table + "_one_default_anchor"
	if len(name) > spec.MaxIdentifier {
		return fmt.Errorf("actor: the default-anchor index needs %d bytes as %q, and the database truncates at %d",
			len(name), name, spec.MaxIdentifier)
	}
	stmt := fmt.Sprintf(
		"create unique index if not exists %s on %s (hierarchy_id) where is_default_anchor = true",
		name, table,
	)
	if err := db.Exec(stmt).Error; err != nil {
		return fmt.Errorf("actor: %s: %w", name, err)
	}
	return nil
}

func seed(db *gorm.DB, s *spec.Spec) error {
	if err := backfillCodes(db, s); err != nil {
		return err
	}
	return raiseSequencesPastExistingCodes(db, s)
}

func backfillCodes(db *gorm.DB, s *spec.Spec) error {
	sequences := tableFor(s, roleCodeSequence)
	for _, c := range codedRoles {
		table := tableFor(s, c.role)
		err := db.Transaction(func(tx *gorm.DB) error {
			var ids []string
			if err := tx.Table(table).Where("code = '' OR code IS NULL").
				Order(backfillOrder(s, c.role)).Pluck("id", &ids).Error; err != nil {
				return err
			}
			for _, id := range ids {
				code, err := nextCode(context.Background(), tx, sequences, c.entityType, c.prefix)
				if err != nil {
					return err
				}
				if err := tx.Table(table).Where("id = ?", id).Update("code", code).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("actor: backfill codes in %s: %w", table, err)
		}
	}
	return nil
}

func backfillOrder(s *spec.Spec, role string) string {
	if declaresColumn(mustObject(s, role), "created_at") {
		return "created_at"
	}
	return "id"
}

func raiseSequencesPastExistingCodes(db *gorm.DB, s *spec.Spec) error {
	sequences := tableFor(s, roleCodeSequence)
	for _, c := range codedRoles {
		var codes []string
		if err := db.Table(tableFor(s, c.role)).Pluck("code", &codes).Error; err != nil {
			return fmt.Errorf("actor: read codes in %s: %w", tableFor(s, c.role), err)
		}
		highest := 0
		for _, code := range codes {
			if n := numberIn(code, c.prefix); n > highest {
				highest = n
			}
		}
		if highest == 0 {
			continue
		}
		if err := raiseSequence(context.Background(), db, sequences, c.entityType, highest+1); err != nil {
			return err
		}
	}
	return nil
}

func numberIn(code, prefix string) int {
	rest, ok := strings.CutPrefix(code, prefix+"-")
	if !ok {
		return 0
	}
	n := 0
	for _, r := range rest {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func raiseSequence(ctx context.Context, tx *gorm.DB, table, entityType string, next int) error {
	var held []int
	if err := tx.WithContext(ctx).Table(table).
		Where("entity_type = ?", entityType).Pluck("next_number", &held).Error; err != nil {
		return fmt.Errorf("raiseSequence: %w", err)
	}
	if len(held) == 0 {
		err := tx.WithContext(ctx).Table(table).
			Create(map[string]any{"entity_type": entityType, "next_number": next}).Error
		if err != nil {
			return fmt.Errorf("raiseSequence: %w", err)
		}
		return nil
	}
	if held[0] >= next {
		return nil
	}
	err := tx.WithContext(ctx).Table(table).
		Where("entity_type = ?", entityType).Update("next_number", next).Error
	if err != nil {
		return fmt.Errorf("raiseSequence: %w", err)
	}
	return nil
}

func rowCount(db *gorm.DB, table string) (int64, error) {
	var n int64
	if err := db.Table(table).Count(&n).Error; err != nil {
		return 0, fmt.Errorf("actor: count %s: %w", table, err)
	}
	return n, nil
}

func hasColumn(db *gorm.DB, table, column string) (bool, error) {
	var found []int
	q := `select 1 from pragma_table_info(?) where name = ?`
	if db.Dialector.Name() == "postgres" {
		q = `select 1 from information_schema.columns where table_name = ? and column_name = ?`
	}
	if err := db.Raw(q, table, column).Scan(&found).Error; err != nil {
		return false, fmt.Errorf("actor: read columns of %s: %w", table, err)
	}
	return len(found) > 0, nil
}

func declaresColumn(o spec.Object, column string) bool {
	for _, f := range o.Fields {
		if f.Name == column {
			return true
		}
	}
	return false
}

var columnsDeclaredSince = map[string][]string{
	roleActor:            {"first_name", "last_name"},
	roleGroup:            {"join_mode", "hidden"},
	roleRoleKind:         {"leads_group"},
	roleGroupJoinRequest: {"asked_at"},
}

func columnsDeclaredSinceMissing(db *gorm.DB, s *spec.Spec) (bool, error) {
	for role, columns := range columnsDeclaredSince {
		table := tableFor(s, role)
		for _, column := range columns {
			held, err := hasColumn(db, table, column)
			if err != nil {
				return false, err
			}
			if !held {
				return true, nil
			}
		}
	}
	return false, nil
}

func addColumnsDeclaredSince(db *gorm.DB, s *spec.Spec) error {
	for role, columns := range columnsDeclaredSince {
		table := tableFor(s, role)
		if table == "" || !db.Migrator().HasTable(table) {
			continue
		}
		o := mustObject(s, role)
		for _, column := range columns {
			held, err := hasColumn(db, table, column)
			if err != nil {
				return err
			}
			if held {
				continue
			}
			definition, err := addedColumnDefinition(o, column)
			if err != nil {
				return err
			}
			stmt := fmt.Sprintf("alter table %s add column %s", table, definition)
			if err := db.Exec(stmt).Error; err != nil {
				return fmt.Errorf("actor: add %s.%s: %w", table, column, err)
			}
		}
	}
	return nil
}

func addedColumnDefinition(o spec.Object, column string) (string, error) {
	for _, f := range o.Fields {
		if f.Name != column {
			continue
		}
		switch f.Type {
		case spec.TypeBool:
			value := f.Default
			if value == "" {
				value = "false"
			}
			return fmt.Sprintf("%s boolean default %s", f.Name, value), nil
		case spec.TypeString, spec.TypeText, spec.TypeEnum, spec.TypeTimestamp:
			return fmt.Sprintf("%s text default '%s'", f.Name, strings.ReplaceAll(f.Default, "'", "''")), nil
		default:
			return "", fmt.Errorf("actor: %s.%s is a %s, which adding a column does not cover", o.Name, f.Name, f.Type)
		}
	}
	return "", fmt.Errorf("actor: %s declares no field %s to add", o.Name, column)
}
