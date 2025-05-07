{{$.Importer.Import "github.com/stephenafamo/bob"}}

// Set the testDB to enable tests that use the database
var testDB bob.Transactor

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
{{else if eq $.Dialect "sqlite"}}
  {{$.Importer.Import "errors"}}
	{{$.Importer.Import "github.com/antlr4-go/antlr/v4"}}
  {{$.Importer.Import "sqliteparser" "github.com/stephenafamo/sqlparser/sqlite"}}

  func formatQuery(s string) (string, error) {
    input := antlr.NewInputStream(s)
    lexer := sqliteparser.NewSQLiteLexer(input)
    stream := antlr.NewCommonTokenStream(lexer, 0)
    p := sqliteparser.NewSQLiteParser(stream)

    el := &errorListener{}
    p.AddErrorListener(el)

    tree := p.Parse()
    if el.err != "" {
      return "", errors.New(el.err)
    }

    return tree.GetText(), nil
  }

  type errorListener struct {
    *antlr.DefaultErrorListener

    err string
  }

  func (el *errorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol any, line, column int, msg string, e antlr.RecognitionException) {
    el.err = msg
  }
{{else}}
  {{$.Importer.Import "testutils" "github.com/stephenafamo/bob/test/utils"}}
  func formatQuery(query string) (string, error) {
    return testutils.Clean(query), nil
  }
{{end}}
