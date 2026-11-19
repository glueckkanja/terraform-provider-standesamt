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

// findDuplicateResourceTypes reports every resourceType that occurs more than
// once in a single naming schema file. The returned messages are sorted by
// resource type. Detection is informational only: the last entry still wins
// (see NewNamingSchemaMap), which keeps generated names stable.
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

// findDuplicateObjectKeys reports every key that occurs more than once at the
// top level of the JSON object in raw. encoding/json silently keeps the last
// value for a duplicate key; this makes that visible. The returned messages
// are sorted by key.
func findDuplicateObjectKeys(raw []byte, label string) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))

	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("findDuplicateObjectKeys: %w", err)
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("findDuplicateObjectKeys: expected a JSON object")
	}

	values := make(map[string][]string)
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("findDuplicateObjectKeys: %w", err)
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("findDuplicateObjectKeys: expected an object key, got %v", keyTok)
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, fmt.Errorf("findDuplicateObjectKeys: value of %q: %w", key, err)
		}
		values[key] = append(values[key], string(value))
	}

	keys := make([]string, 0)
	for k, v := range values {
		if len(v) > 1 {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil, nil
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
	return messages, nil
}
