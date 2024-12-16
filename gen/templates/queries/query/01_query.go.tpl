{{if .QueryFile.Queries}}
// {{printf "%#v" .QueryFile}}


{{$.Importer.Import "io"}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}


{{range $query := $.QueryFile.Queries}}

{{$upperName := title $query.Name}}
{{$lowerName := untitle $query.Name}}

const {{$lowerName}}SQL = `{{replace "`" "`+\"`\"+`" $query.SQL}}`

func {{$upperName}}() orm.ModExecQuery[*dialect.SelectQuery]{
	return orm.ModExecQuery[*dialect.SelectQuery]{
		ExecQuery: orm.ExecQuery[orm.ModExpression[*dialect.SelectQuery]]{
			BaseQuery: bob.BaseQuery[orm.ModExpression[*dialect.SelectQuery]]{
				Expression: {{$lowerName}}{},
				Dialect:    dialect.Dialect,
				QueryType:  bob.QueryType{{lower $query.Type.String | title}},
			},
		},
	}
}

type {{$lowerName}} struct {
	id   int
	name string
}

func ({{$lowerName}}) expr(from, to int) bob.Expression {
	return nil
}

func ({{$lowerName}}) Apply(q *dialect.SelectQuery) {
  {{if $query.Mods -}}
    {{join "\n" ($query.Mods $.Importer)}}
  {{- end}}
}

func (q {{$lowerName}}) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return q.expr(0, len({{$lowerName}}SQL)).WriteSQL(ctx, w, d, start)
}

{{end}}


{{end}}
