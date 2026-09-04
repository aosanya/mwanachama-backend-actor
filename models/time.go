package models

import "time"

// TimeLayout is the fixed-width, nanosecond-precision RFC 3339 layout every
// timestamp field on a model in this package is written and read with —
// including sort keys such as Group.CreatedAt and ActorGroupAssignment.JoinedAt.
//
// Plain time.RFC3339 (second precision) is NOT enough: two registrations
// created microseconds apart in the same second would compare equal on
// JoinedAt and the sort would fall through to the tie-break (actor/group
// id), which for randomly-minted UUIDs does not track real join order — see
// the gateway's TestRegistrationRosterOrderIsNotCallerWritable, the parity
// test that caught exactly this. time.RFC3339Nano is not safe either: Go
// trims trailing fractional zeros, so two timestamps with different digit
// counts stop comparing correctly as plain strings. A fixed nine-digit
// fractional part keeps every stored timestamp both parseable AND
// lexicographically sortable in the same order as chronologically.
const TimeLayout = "2006-01-02T15:04:05.000000000Z07:00"

// NowRFC3339 returns the current UTC time formatted per [TimeLayout].
func NowRFC3339() string {
	return time.Now().UTC().Format(TimeLayout)
}
