package mwanachamaactor

import (
	"errors"
	"time"

	"github.com/aosanya/mwanachama-backend-shared/versioncheck"
)

var ErrStaleVersion = errors.New("mwanachamaactor: the record changed since you read it")

type staleVersion struct {
	cause *versioncheck.StaleError
}

func (e staleVersion) Error() string { return e.cause.Error() }

func (e staleVersion) Unwrap() error { return e.cause }

func (e staleVersion) Is(target error) bool { return target == ErrStaleVersion }

func checkGroupVersion(current Group, e GroupEdit) error {
	if e.Version == "" {
		return nil
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, current.LastUpdated)
	if err != nil {
		return nil
	}
	err = versioncheck.Stale(e.Version, updatedAt, []versioncheck.Field{
		{Name: "name", Sent: e.Name, Stored: current.Name},
		{Name: "node_type", Sent: e.NodeType, Stored: current.NodeType},
		{Name: "anchor_level_override", Sent: e.AnchorLevelOverrideID, Stored: current.AnchorLevelOverrideID},
	})
	if stale, ok := versioncheck.IsStale(err); ok {
		return staleVersion{cause: stale}
	}
	return nil
}
