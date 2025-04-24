func zero[T any]() (z T) {
  return z
}

{{if eq $.Dialect "psql"}}
  {{$.Importer.Import "pg_query" "github.com/wasilibs/go-pgquery"}}
  func formatQuery(s string) (string, error) {
    aTree, err := pg_query.Parse(s)
    if err != nil {
      return "", err
    }

    return pg_query.Deparse(aTree)
  }
{{else}}
  {{$.Importer.Import "testutils" "github.com/stephenafamo/bob/test/utils"}}
  func formatQuery(query string) (string, error) {
    return testutils.Clean(query), nil
  }
{{end}}
