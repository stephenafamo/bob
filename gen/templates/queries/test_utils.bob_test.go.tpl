{{$.Importer.Import "github.com/stephenafamo/bob"}}

// Set the testDB to enable tests that use the database
var testDB bob.Transactor

{{if eq $.Dialect "psql"}}
  {{$.Importer.Import "pg_query" "github.com/wasilibs/go-pgquery"}}
  func formatQuery(s string) (string, error) {
    aTree, err := pg_query.Parse(s)
    if err != nil {
      return "", err
    }

    return pg_query.Deparse(aTree)
  }
{{else if eq $.Dialect "mysql"}}
  {{$.Importer.Import "errors"}}
	{{$.Importer.Import "github.com/antlr4-go/antlr/v4"}}
  {{$.Importer.Import "mysqlparser" "github.com/stephenafamo/sqlparser/mysql"}}

  func formatQuery(s string) (string, error) {
    input := antlr.NewInputStream(s)
    lexer := mysqlparser.NewMySqlLexer(input)
    stream := antlr.NewCommonTokenStream(lexer, 0)
    p := mysqlparser.NewMySqlParser(stream)

    el := &errorListener{}
    p.AddErrorListener(el)

    tree := p.Root()
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


{{$.Importer.Import "github.com/jaswdr/faker/v2"}}
var defaultFaker = faker.New()

{{$doneTypes := dict }}
{{range $colTyp := $.QueryFolder.Types -}}
  {{- if hasKey $doneTypes $colTyp}}{{continue}}{{end -}}
  {{- $_ := set $doneTypes $colTyp nil -}}
  {{- $typDef := $.Types.Index $colTyp -}}
  {{range $depTyp := $typDef.DependsOn}}
    {{- $_ := set $doneTypes $depTyp nil -}}
  {{end}}
{{end -}}



{{range $colTyp := keys $doneTypes | sortAlpha -}}
    {{- $typDef := $.Types.Index $colTyp -}}
    {{- $typ := $.Types.Get $.CurrentPackage $.Importer $colTyp -}}
    {{- if not $typDef.RandomExpr -}}{{continue}}{{/*
      Ensures that compilation fails.
      Users of custom types can decide to use a non-random expression
      but this would be a conscious decision.
    */}}{{- end -}}
    {{- $.Importer.ImportList $typDef.RandomExprImports -}}
    func random_{{normalizeType $colTyp}}(f *faker.Faker, limits ...string) {{$typ}} {
      if f == nil {
        f = &defaultFaker
      }

      {{replace "BASETYPE" $typ $typDef.RandomExpr}}
    }
{{end -}}
