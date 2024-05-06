package testfiles

import "embed"

//go:embed psql
var PostgresSchema embed.FS

//go:embed mysql
var MySQLSchema embed.FS

//go:embed sqlite
var SQLiteSchema embed.FS
