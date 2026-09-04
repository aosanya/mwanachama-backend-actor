package gormstore

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

// TableNames configures which physical tables a UserManager reads and
// writes. Mirrors mwanachama-backend-shared/postgres.TableNames' shape (one
// consumer, one set of tables) without depending on that package's type —
// these three tables are specific to this domain, not a generic entitygraph
// convention.
type TableNames struct {
	Actors                string
	Groups                string
	ActorGroupAssignments string
}

// DefaultTableNames builds the conventional table set for one mounted
// instance of this package, e.g. DefaultTableNames("member") yields
// member_actors, member_groups, member_actor_group_assignments —
// preserving the multi-instance-mount capability the old entitygraph-based
// DefaultUserSchema(instance) used to provide via StorageCollection labels.
// mwanachama-backend-api-gateway mounts exactly one instance ("member")
// today; if a second is ever mounted into the same Postgres schema, note
// that GORM derives index/constraint names from the row structs, not the
// runtime table name — Migrate for two instances sharing one schema may
// collide on constraint names. Not solved here since it isn't exercised
// today.
func DefaultTableNames(instance string) TableNames {
	return TableNames{
		Actors:                instance + "_actors",
		Groups:                instance + "_groups",
		ActorGroupAssignments: instance + "_actor_group_assignments",
	}
}

// Migrate creates or updates the three tables t names, via GORM's
// AutoMigrate scoped to each table name in turn. Callers run this once at
// startup (or in test setup) before constructing a UserManager with the
// same db and t.
func Migrate(db *gorm.DB, t TableNames) error {
	if err := db.Table(t.Actors).AutoMigrate(&ActorRow{}); err != nil {
		return err
	}
	if err := syncUniqueAttributeIndexes(db, t.Actors, models.DefaultActorProperties()); err != nil {
		return err
	}
	if err := db.Table(t.Groups).AutoMigrate(&GroupRow{}); err != nil {
		return err
	}
	if err := syncUniqueAttributeIndexes(db, t.Groups, models.DefaultGroupProperties()); err != nil {
		return err
	}
	if err := db.Table(t.ActorGroupAssignments).AutoMigrate(&ActorGroupAssignmentRow{}); err != nil {
		return err
	}
	if err := syncUniqueAttributeIndexes(db, t.ActorGroupAssignments, models.DefaultActorGroupAssignmentProperties()); err != nil {
		return err
	}
	return nil
}

// syncUniqueAttributeIndexes creates a partial unique index over the
// attributes column's JSON path for every Unique property in properties,
// naming it "<table>_attr_<property>_uniq". This is the database-level half
// of the Unique constraint — the root package's checkUniqueAttributes is a
// pre-check for a friendly [ErrDuplicateAttribute], but only this index
// stops two concurrent writes from both passing that pre-check and landing
// duplicate values.
//
// Partial ("WHERE ... IS NOT NULL") because Required is a separate,
// Go-level-only constraint here: nothing at this layer should reject a row
// that simply never set the property. Skipped on any dialect other than
// postgres/sqlite (the two this repo's tests and deployments actually use)
// rather than guessing at unsupported SQL.
func syncUniqueAttributeIndexes(db *gorm.DB, table string, properties []models.Property) error {
	for _, p := range properties {
		if !p.Unique {
			continue
		}
		var expr string
		switch db.Dialector.Name() {
		case "postgres":
			// Double-wrapped: Postgres requires an expression index's
			// column-list entry to be its own parenthesized expression
			// (a bare "(attributes ->> 'x')" parses as the column list
			// itself, not as one expression inside it) — CREATE INDEX ...
			// ON t ((attributes ->> 'x')). The extra parens are harmless
			// where expr is reused in the WHERE clause below.
			expr = fmt.Sprintf("((attributes ->> '%s'))", p.Name)
		case "sqlite":
			expr = fmt.Sprintf("(json_extract(attributes, '$.%s'))", p.Name)
		default:
			continue
		}
		idx := fmt.Sprintf("%s_attr_%s_uniq", table, p.Name)
		sql := fmt.Sprintf(
			"CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s %s WHERE %s IS NOT NULL",
			idx, table, expr, expr,
		)
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("syncUniqueAttributeIndexes: %s: %w", idx, err)
		}
	}
	return nil
}
