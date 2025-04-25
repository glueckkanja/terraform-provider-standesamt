// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package random

import (
	"testing"
)

func TestHash(t *testing.T) {
	length := 10
	seed := int64(42)

	hash1 := Hash(length, seed)
	hash2 := Hash(length, seed)

	if len(hash1) != length {
		t.Errorf("Expected hash length %d, got %d", length, len(hash1))
	}

	if hash1 != hash2 {
		t.Errorf("Expected deterministic hash values, but got %s and %s", hash1, hash2)
	}
}

func TestStringWithCharset(t *testing.T) {
	length := 8
	customCharset := "abc123"

	result := StringWithCharset(length, customCharset)

	if len(result) != length {
		t.Errorf("Expected string length %d, got %d", length, len(result))
	}

	for _, char := range result {
		if !contains(customCharset, char) {
			t.Errorf("Unexpected character %c in result", char)
		}
	}
}

func contains(charset string, char rune) bool {
	for _, c := range charset {
		if c == char {
			return true
		}
	}
	return false
}
