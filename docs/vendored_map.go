// Package orderedmap implements an ordered map, i.e. a map that also keeps track of
// the order in which keys were inserted.
//
// All operations are constant-time.
//
// Vendored fork of Github repo: https://github.com/wk8/go-ordered-map
package docs

import (
	"encoding"
	"encoding/json"
	"fmt"

	list "github.com/bahlo/generic-list-go"
	"github.com/valyala/fastjson"
)

type Pair[K comparable, V any] struct {
	Key   K
	Value V

	element *list.Element[*Pair[K, V]]
}

type OrderedMap[K comparable, V any] struct {
	pairs map[K]*Pair[K, V]
	list  *list.List[*Pair[K, V]]
}

// New creates a new OrderedMap.
func NewMap[K comparable, V any]() *OrderedMap[K, V] {
	return &OrderedMap[K, V]{
		pairs: make(map[K]*Pair[K, V]),
		list:  list.New[*Pair[K, V]](),
	}
}

// Get looks for the given key, and returns the value associated with it,
// or V's nil value if not found. The boolean it returns says whether the key is present in the map.
func (om *OrderedMap[K, V]) Get(key K) (val V, present bool) {
	if pair, present := om.pairs[key]; present {
		return pair.Value, true
	}

	return
}

// Load is an alias for Get, mostly to present an API similar to `sync.Map`'s.
func (om *OrderedMap[K, V]) Load(key K) (V, bool) {
	return om.Get(key)
}

// GetPair looks for the given key, and returns the pair associated with it,
// or nil if not found. The Pair struct can then be used to iterate over the ordered map
// from that point, either forward or backward.
func (om *OrderedMap[K, V]) GetPair(key K) *Pair[K, V] {
	return om.pairs[key]
}

// Set sets the key-value pair, and returns what `Get` would have returned
// on that key prior to the call to `Set`.
func (om *OrderedMap[K, V]) Set(key K, value V) (val V, present bool) {
	if pair, present := om.pairs[key]; present {
		oldValue := pair.Value
		pair.Value = value
		return oldValue, true
	}

	pair := &Pair[K, V]{
		Key:   key,
		Value: value,
	}
	pair.element = om.list.PushBack(pair)
	om.pairs[key] = pair

	return
}

// Store is an alias for Set, mostly to present an API similar to `sync.Map`'s.
func (om *OrderedMap[K, V]) Store(key K, value V) (V, bool) {
	return om.Set(key, value)
}

// Delete removes the key-value pair, and returns what `Get` would have returned
// on that key prior to the call to `Delete`.
func (om *OrderedMap[K, V]) Delete(key K) (val V, present bool) {
	if pair, present := om.pairs[key]; present {
		om.list.Remove(pair.element)
		delete(om.pairs, key)
		return pair.Value, true
	}
	return
}

// Len returns the length of the ordered map.
func (om *OrderedMap[K, V]) Len() int {
	return len(om.pairs)
}

// Oldest returns a pointer to the oldest pair. It's meant to be used to iterate on the ordered map's
// pairs from the oldest to the newest, e.g.:
// for pair := orderedMap.Oldest(); pair != nil; pair = pair.Next() { fmt.Printf("%v => %v\n", pair.Key, pair.Value) }
func (om *OrderedMap[K, V]) Oldest() *Pair[K, V] {
	return listElementToPair(om.list.Front())
}

// Newest returns a pointer to the newest pair. It's meant to be used to iterate on the ordered map's
// pairs from the newest to the oldest, e.g.:
// for pair := orderedMap.Oldest(); pair != nil; pair = pair.Next() { fmt.Printf("%v => %v\n", pair.Key, pair.Value) }
func (om *OrderedMap[K, V]) Newest() *Pair[K, V] {
	return listElementToPair(om.list.Back())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (om *OrderedMap[K, V]) UnmarshalJSON(b []byte) error {
	// Uses fastjson to unmarshal the JSON into a map, then iterates over the map to insert the pairs
	// into the ordered map.]

	v, err := fastjson.ParseBytes(b)

	if err != nil {
		return err
	}

	if v.Type() != fastjson.TypeObject {
		return fmt.Errorf("expected JSON object, got %v", v.Type())
	}

	omObj := NewMap[K, V]()

	// We cant use visit as K and V are not known at compile time

	v.GetObject().Visit(func(key []byte, value *fastjson.Value) {
		var k K
		var v V

		if err := json.Unmarshal(value.MarshalTo(nil), &v); err != nil {
			return
		}

		keyStr := string(key)

		// Copy the key to
		if err := json.Unmarshal([]byte(fmt.Sprintf("\"%s\"", keyStr)), &k); err != nil {
			return
		}

		omObj.Set(k, v)
	})

	// Copy the new ordered map into the receiver
	*om = *omObj

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (om *OrderedMap[K, V]) MarshalJSON() ([]byte, error) {
	result := "{"

	i := 0
	for pair := om.Oldest(); pair != nil; pair = pair.Next() {
		if i > 0 {
			result += ","
		}

		var marshaledKey string
		switch key := any(pair.Key).(type) {
		case string:
			marshaledKey = key
		case encoding.TextMarshaler:
			marshaledKeyBytes, err := key.MarshalText()
			if err != nil {
				return nil, err
			}
			marshaledKey = string(marshaledKeyBytes)
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			marshaledKey = fmt.Sprintf(`%d`, key)
		default:
			return nil, fmt.Errorf("unsupported key type: %T", key)
		}
		marshaledKey = `"` + marshaledKey + `"`

		value, err := json.Marshal(pair.Value)
		if err != nil {
			return nil, err
		}

		result += fmt.Sprintf("%s:%s", marshaledKey, value)
		i++
	}

	result += "}"

	return []byte(result), nil
}

// Next returns a pointer to the next pair.
func (p *Pair[K, V]) Next() *Pair[K, V] {
	return listElementToPair(p.element.Next())
}

// Prev returns a pointer to the previous pair.
func (p *Pair[K, V]) Prev() *Pair[K, V] {
	return listElementToPair(p.element.Prev())
}

func listElementToPair[K comparable, V any](element *list.Element[*Pair[K, V]]) *Pair[K, V] {
	if element == nil {
		return nil
	}
	return element.Value
}

// KeyNotFoundError may be returned by functions in this package when they're called with keys that are not present
// in the map.
type KeyNotFoundError[K comparable] struct {
	MissingKey K
}

func (e *KeyNotFoundError[K]) Error() string {
	return fmt.Sprintf("missing key: %v", e.MissingKey)
}

// MoveAfter moves the value associated with key to its new position after the one associated with markKey.
// Returns an error iff key or markKey are not present in the map.
func (om *OrderedMap[K, V]) MoveAfter(key, markKey K) error {
	elements, err := om.getElements(key, markKey)
	if err != nil {
		return err
	}
	om.list.MoveAfter(elements[0], elements[1])
	return nil
}

// MoveBefore moves the value associated with key to its new position before the one associated with markKey.
// Returns an error iff key or markKey are not present in the map.
func (om *OrderedMap[K, V]) MoveBefore(key, markKey K) error {
	elements, err := om.getElements(key, markKey)
	if err != nil {
		return err
	}
	om.list.MoveBefore(elements[0], elements[1])
	return nil
}

func (om *OrderedMap[K, V]) getElements(keys ...K) ([]*list.Element[*Pair[K, V]], error) {
	elements := make([]*list.Element[*Pair[K, V]], len(keys))
	for i, k := range keys {
		pair, present := om.pairs[k]
		if !present {
			return nil, &KeyNotFoundError[K]{k}
		}
		elements[i] = pair.element
	}
	return elements, nil
}

// MoveToBack moves the value associated with key to the back of the ordered map.
// Returns an error iff key is not present in the map.
func (om *OrderedMap[K, V]) MoveToBack(key K) error {
	pair, present := om.pairs[key]
	if !present {
		return &KeyNotFoundError[K]{key}
	}
	om.list.MoveToBack(pair.element)
	return nil
}

// MoveToFront moves the value associated with key to the front of the ordered map.
// Returns an error iff key is not present in the map.
func (om *OrderedMap[K, V]) MoveToFront(key K) error {
	pair, present := om.pairs[key]
	if !present {
		return &KeyNotFoundError[K]{key}
	}
	om.list.MoveToFront(pair.element)
	return nil
}
