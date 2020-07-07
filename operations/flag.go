package operations

import "strings"

func joinFlagNames(ids ...string) string { return strings.Join(ids, ", ") }
