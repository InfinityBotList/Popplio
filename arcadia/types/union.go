// Package types holds the wire types of the staff panel API (ported from
// Arcadia, a Rust service).
//
// The wire format is frozen: a live SvelteKit staff panel and several other
// Infinity List services already speak it. Everything in this package exists to
// reproduce serde's DEFAULT enum representation, which Go's encoding/json has no
// equivalent of. Read arcadia/CONFORMANCE.md before changing anything here.
package types

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Externally tagged enums (serde's default) encode as follows:
//
//	struct variant  Hello { login_token, version }  -> {"Hello":{"login_token":..,"version":..}}
//	unit variant    ListPositions                   -> "ListPositions"
//	newtype variant Bot(PartialBot)                 -> {"Bot":{..}}
//
// decodeUnion splits such a payload into its variant name and its inner value.
// A bare JSON string is a unit variant and yields a nil payload.
func decodeUnion(data []byte) (variant string, payload json.RawMessage, err error) {
	// A unit variant arrives as a bare string.
	var unit string
	if err := json.Unmarshal(data, &unit); err == nil {
		return unit, nil, nil
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return "", nil, fmt.Errorf("expected a string or a single-key object: %w", err)
	}

	if len(obj) != 1 {
		return "", nil, fmt.Errorf("expected exactly one variant key, got %d", len(obj))
	}

	for k, v := range obj {
		return k, v, nil
	}

	return "", nil, errors.New("unreachable")
}

// encodeUnit encodes a unit variant: a bare JSON string.
func encodeUnit(name string) ([]byte, error) {
	return json.Marshal(name)
}

// encodeVariant encodes a struct or newtype variant: a single-key object.
func encodeVariant(name string, inner any) ([]byte, error) {
	payload, err := json.Marshal(inner)
	if err != nil {
		return nil, err
	}

	// Built by hand rather than through a map so the key is emitted verbatim and
	// no HTML escaping is applied to the variant name.
	out := make([]byte, 0, len(name)+len(payload)+6)
	out = append(out, '{')
	key, err := json.Marshal(name)
	if err != nil {
		return nil, err
	}
	out = append(out, key...)
	out = append(out, ':')
	out = append(out, payload...)
	out = append(out, '}')

	return out, nil
}

// Unit is the payload of a variant that carries no fields. Its presence (a
// non-nil pointer) is what selects the variant.
type Unit struct{}

// unitSet is shorthand for setting a unit variant.
func unitSet() *Unit { return &Unit{} }

// unmarshalUnion decodes an externally tagged enum against a table of variant
// constructors. Each constructor assigns its variant on the destination and
// returns the value to decode the payload into, or nil for a unit variant.
func unmarshalUnion(union string, data []byte, variants map[string]func() any) error {
	name, payload, err := decodeUnion(data)
	if err != nil {
		return fmt.Errorf("%s: %w", union, err)
	}

	construct, ok := variants[name]
	if !ok {
		return errUnknownVariant(union, name)
	}

	target := construct()

	if target == nil {
		// Unit variant. serde accepts both "Name" and {"Name":null} and rejects
		// anything else, so we do the same.
		if payload == nil || string(payload) == "null" {
			return nil
		}

		return fmt.Errorf("invalid type: expected unit for variant `%s` of %s", name, union)
	}

	if payload == nil {
		return errVariantNeedsPayload(union, name)
	}

	return json.Unmarshal(payload, target)
}

// errUnknownVariant is the shape of the error serde produces for an unmatched
// variant. The panel never relies on the text, but keeping it uniform makes the
// dispatcher's 500s readable.
func errUnknownVariant(union, variant string) error {
	return fmt.Errorf("unknown variant `%s` for %s", variant, union)
}

// errVariantNeedsPayload is returned when a struct variant arrives as a bare
// string (i.e. was treated as a unit variant by the caller).
func errVariantNeedsPayload(union, variant string) error {
	return fmt.Errorf("variant `%s` of %s requires an object payload", variant, union)
}
