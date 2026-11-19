// Copyright glueckkanja AG 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// findDuplicateResourceTypes reports resourceTypes listed more than once (the last entry wins).
func findDuplicateResourceTypes(schemas []JsonNamingSchema) []string {
	indices := make(map[string][]int)
	for i, s := range schemas {
		indices[s.ResourceType] = append(indices[s.ResourceType], i)
	}

	dupTypes := make([]string, 0)
	for t, idx := range indices {
		if len(idx) > 1 {
			dupTypes = append(dupTypes, t)
		}
	}
	if len(dupTypes) == 0 {
		return nil
	}
	sort.Strings(dupTypes)

	messages := make([]string, 0, len(dupTypes))
	for _, t := range dupTypes {
		idx := indices[t]

		identical := true
		for _, i := range idx[1:] {
			if !reflect.DeepEqual(schemas[idx[0]], schemas[i]) {
				identical = false
				break
			}
		}
		kind := "entries differ"
		if identical {
			kind = "entries are identical"
		}

		refs := make([]string, len(idx))
		for n, i := range idx {
			refs[n] = fmt.Sprintf("#%d", i+1)
		}

		messages = append(messages, fmt.Sprintf(
			"resourceType %q is defined %d times (entries %s); %s; the last entry (%s) is used",
			t, len(idx), strings.Join(refs, ", "), kind, refs[len(refs)-1],
		))
	}
	return messages
}

type keyValue struct {
	key   string
	value json.RawMessage
}

// objectPairs returns the top-level key/value pairs of a JSON object in file order.
func objectPairs(raw []byte) ([]keyValue, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))

	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("objectPairs: %w", err)
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("objectPairs: expected a JSON object")
	}

	var pairs []keyValue
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("objectPairs: %w", err)
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("objectPairs: expected an object key, got %v", keyTok)
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, fmt.Errorf("objectPairs: value of %q: %w", key, err)
		}
		pairs = append(pairs, keyValue{key: key, value: value})
	}
	return pairs, nil
}

// duplicatePairs reports keys that occur more than once, which encoding/json silently drops.
func duplicatePairs(pairs []keyValue, label string) []string {
	values := make(map[string][]string)
	for _, p := range pairs {
		values[p.key] = append(values[p.key], string(p.value))
	}

	keys := make([]string, 0)
	for k, v := range values {
		if len(v) > 1 {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)

	messages := make([]string, 0, len(keys))
	for _, k := range keys {
		v := values[k]
		messages = append(messages, fmt.Sprintf(
			"%s %q is defined %d times (values %s); the last value (%s) is used",
			label, k, len(v), strings.Join(v, ", "), v[len(v)-1],
		))
	}
	return messages
}
