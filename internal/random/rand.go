// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package random

import (
	"math/rand"
)

const charset = "abcdefghijklmnopqrstuvwxyz"

var seededRand *rand.Rand

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func Hash(length int, seed int64) string {
	seededRand = rand.New(rand.NewSource(seed))
	return StringWithCharset(length, charset)
}
