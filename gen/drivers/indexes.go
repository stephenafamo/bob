package drivers

// DBIndexes lists all indexes in the database schema keyed by table name
type DBIndexes map[string][]Index

// Index represents an index in a table
type Index NamedColumnList
