package impls

import (
	"encoding/hex"

	"github.com/jackc/pgx/v5/pgtype"
)

// UUIDString renders a pgtype.UUID the way Rust's Uuid::hyphenated does. An
// invalid (NULL) uuid renders as the empty string.
func UUIDString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}

	buf := make([]byte, 36)
	hex.Encode(buf[0:8], u.Bytes[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u.Bytes[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u.Bytes[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u.Bytes[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], u.Bytes[10:16])

	return string(buf)
}

// UUIDStrings maps a slice of pgtype.UUIDs to their hyphenated forms.
func UUIDStrings(ids []pgtype.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, UUIDString(id))
	}

	return out
}
