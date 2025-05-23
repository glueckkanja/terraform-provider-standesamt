// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package tools

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"strings"
)

func GetBaseString(s types.String) string {
	if s.IsUnknown() {
		return ""
	}
	if s.IsNull() {
		return ""
	}
	return strings.Trim(s.ValueString(), "\"")
}
