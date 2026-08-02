package types

import (
	"encoding/json"
	"strings"
	"time"
)

// Timestamp reproduces chrono's serde representation of DateTime<Utc>, which is
// RFC3339 with "Z" for UTC and SecondsFormat::AutoSi fractional seconds: the
// fraction is omitted entirely when zero, and otherwise printed with exactly 3,
// 6 or 9 digits.
//
// Go's own time.Time marshaller trims trailing zeroes one digit at a time
// (0.500s becomes ".5"), which chrono never emits. The panel parses both, but
// the wire format is frozen, so we match chrono.
type Timestamp struct {
	time.Time
}

func NewTimestamp(t time.Time) Timestamp {
	return Timestamp{Time: t}
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Format())
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return err
	}

	t.Time = parsed
	return nil
}

// Format renders the timestamp the way chrono's SecondsFormat::AutoSi does.
func (t Timestamp) Format() string {
	utc := t.Time.UTC()
	nanos := utc.Nanosecond()

	var layout string
	switch {
	case nanos == 0:
		layout = "2006-01-02T15:04:05"
	case nanos%1_000_000 == 0:
		layout = "2006-01-02T15:04:05.000"
	case nanos%1_000 == 0:
		layout = "2006-01-02T15:04:05.000000"
	default:
		layout = "2006-01-02T15:04:05.000000000"
	}

	return utc.Format(layout) + "Z"
}

func (t Timestamp) String() string {
	return t.Format()
}

// OptTimestamp is a nullable Timestamp. Rust's Option<DateTime<Utc>> emits null
// for None, so this must never carry omitempty.
type OptTimestamp = *Timestamp

// TimestampPtr is a convenience for building an OptTimestamp from a nullable
// database column.
func TimestampPtr(t *time.Time) *Timestamp {
	if t == nil {
		return nil
	}

	ts := NewTimestamp(*t)
	return &ts
}

// IsZeroLike reports whether the rendered form has no fractional part. Used only
// by tests.
func (t Timestamp) hasFraction() bool {
	return strings.Contains(t.Format(), ".")
}
