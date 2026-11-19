// Copyright glueckkanja AG 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

const duplicateWarningHint = "Remove the duplicate from the schema library so that edits to the earlier entry are not silently ignored. " +
	"If you use the default schema library, this is a known upstream issue and no action is required."

// addDuplicateWarnings turns duplicate-entry findings from the schema loader
// into warning diagnostics. Duplicates never fail a run: the last entry wins,
// exactly as before detection was added.
func addDuplicateWarnings(diags *diag.Diagnostics, summary string, messages []string) {
	for _, m := range messages {
		diags.AddWarning(summary, m+"\n\n"+duplicateWarningHint)
	}
}
