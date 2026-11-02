package mwanachamaactor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

var ErrSchemaName = errors.New("actor: not a schema name")

var schemaName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type SeatDuplicate struct {
	ActorID         string
	GroupID         string
	GroupName       string
	KindID          string
	KindName        string
	Seats           int
	SeatIDs         []string
	OldestGrantedAt string
	NewestGrantedAt string
}

type seatDuplicateRow struct {
	ID        string
	ActorID   string
	GroupID   string
	GroupName string
	KindID    string
	KindName  string
	Seats     int
	Oldest    string
	Newest    string
}

func FindSeatDuplicates(ctx context.Context, db *gorm.DB, s *spec.Spec, schema string) ([]SeatDuplicate, error) {
	if schema != "" && !schemaName.MatchString(schema) {
		return nil, fmt.Errorf("%w: %q", ErrSchemaName, schema)
	}
	seats, groups, kinds := tableFor(s, roleActorRoleAssignment), tableFor(s, roleGroup), tableFor(s, roleRoleKind)
	if seats == "" || groups == "" || kinds == "" {
		return nil, errors.New("actor: the spec fills no object for the seat, group or role kind role")
	}
	query := seatDuplicatesSQL(qualified(schema, seats), qualified(schema, groups), qualified(schema, kinds))

	var rows []seatDuplicateRow
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Raw(query).Scan(&rows).Error
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("actor: read the live seats in %s: %w", seats, err)
	}
	return groupSeatDuplicates(rows), nil
}

func qualified(schema, table string) string {
	if schema == "" {
		return `"` + table + `"`
	}
	return `"` + schema + `"."` + table + `"`
}

func seatDuplicatesSQL(seats, groups, kinds string) string {
	return `select s.id, s.actor_id, s.group_id, s.kind_id,
	d.seats, d.oldest, d.newest,
	coalesce(g.name, '') as group_name, coalesce(k.name, '') as kind_name
from ` + seats + ` s
join (
	select actor_id, group_id, kind_id,
		count(*) as seats,
		coalesce(min(granted_at), '') as oldest,
		coalesce(max(granted_at), '') as newest
	from ` + seats + `
	where active = true
	group by actor_id, group_id, kind_id
	having count(*) > 1
) d on d.actor_id = s.actor_id and d.group_id = s.group_id and d.kind_id = s.kind_id
left join ` + groups + ` g on g.id = s.group_id
left join ` + kinds + ` k on k.id = s.kind_id
where s.active = true
order by s.actor_id, s.group_id, s.kind_id, s.granted_at, s.id`
}

func groupSeatDuplicates(rows []seatDuplicateRow) []SeatDuplicate {
	var out []SeatDuplicate
	for _, r := range rows {
		last := len(out) - 1
		if last >= 0 && out[last].ActorID == r.ActorID && out[last].GroupID == r.GroupID && out[last].KindID == r.KindID {
			out[last].SeatIDs = append(out[last].SeatIDs, r.ID)
			continue
		}
		out = append(out, SeatDuplicate{
			ActorID:         r.ActorID,
			GroupID:         r.GroupID,
			GroupName:       r.GroupName,
			KindID:          r.KindID,
			KindName:        r.KindName,
			Seats:           r.Seats,
			SeatIDs:         []string{r.ID},
			OldestGrantedAt: r.Oldest,
			NewestGrantedAt: r.Newest,
		})
	}
	return out
}
