// Package mysqlseed contains the current fresh-database baseline. It is never
// used as a production personnel migration.
package mysqlseed

import _ "embed"

//go:embed bootstrap.sql
var SQL string
