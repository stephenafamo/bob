{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

{{block "helpers/then_load_count_variables" . -}}
var (
	ThenLoadCount = getThenLoadCount[*dialect.SelectQuery]()
	InsertThenLoadCount = getThenLoadCount[*dialect.InsertQuery]()
	UpdateThenLoadCount = getThenLoadCount[*dialect.UpdateQuery]()
)
{{- end}}

type thenLoadCounts[Q orm.Loadable] struct {
	{{range $table := .Tables -}}
	{{- $rels := $.Relationships.Get $table.Key -}}
	{{- $hasToMany := false -}}
	{{- range $rel := $rels -}}
		{{- if $rel.IsToMany -}}{{- $hasToMany = true -}}{{- end -}}
	{{- end -}}
	{{- if $hasToMany -}}
	{{$tAlias := $.Aliases.Table $table.Key -}}
	{{$tAlias.UpSingular}} {{$tAlias.DownSingular}}CountThenLoader[Q]
	{{end}}{{end}}
}

func getThenLoadCount[Q orm.Loadable]() thenLoadCounts[Q] {
	return thenLoadCounts[Q]{
		{{range $table := .Tables -}}
		{{- $rels := $.Relationships.Get $table.Key -}}
		{{- $hasToMany := false -}}
		{{- range $rel := $rels -}}
			{{- if $rel.IsToMany -}}{{- $hasToMany = true -}}{{- end -}}
		{{- end -}}
		{{- if $hasToMany -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpSingular}}: build{{$tAlias.UpSingular}}CountThenLoader[Q](),
		{{end}}{{end}}
	}
}

func countThenLoadBuilder[Q orm.Loadable, T any](name string, f func(context.Context, bob.Executor, T, ...bob.Mod[*dialect.SelectQuery]) error) func(...bob.Mod[*dialect.SelectQuery]) orm.Loader[Q] {
	return func(queryMods ...bob.Mod[*dialect.SelectQuery]) orm.Loader[Q] {
		return func(ctx context.Context, exec bob.Executor, retrieved any) error {
			loader, isLoader := retrieved.(T)
			if !isLoader {
				return nil // silently skip if not the right type
			}

			return f(ctx, exec, loader, queryMods...)
		}
	}
}
