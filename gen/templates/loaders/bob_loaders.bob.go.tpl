{{$.Importer.Import "fmt"}}
{{$.Importer.Import "errors"}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "database/sql"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

var Preload = getPreloaders()

type preloaders struct {
		{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpSingular}} {{$tAlias.DownSingular}}Preloader
		{{end}}{{end}}
}

func getPreloaders() preloaders {
	return preloaders{
		{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpSingular}}: build{{$tAlias.UpSingular}}Preloader(),
		{{end}}{{end}}
	}
}

{{block "helpers/then_load_variables" . -}}
var (
	SelectThenLoad = getThenLoaders[*dialect.SelectQuery]()
	InsertThenLoad = getThenLoaders[*dialect.InsertQuery]()
	UpdateThenLoad = getThenLoaders[*dialect.UpdateQuery]()
)
{{- end}}

type thenLoaders[Q orm.Loadable] struct {
		{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpSingular}} {{$tAlias.DownSingular}}ThenLoader[Q]
		{{end}}{{end}}
}

func getThenLoaders[Q orm.Loadable]() thenLoaders[Q] {
	return thenLoaders[Q]{
		{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpSingular}}: build{{$tAlias.UpSingular}}ThenLoader[Q](),
		{{end}}{{end}}
	}
}


func thenLoadBuilder[Q orm.Loadable, T any](name string, f func(context.Context, bob.Executor, T, ...bob.Mod[*dialect.SelectQuery]) error) func(...bob.Mod[*dialect.SelectQuery]) orm.Loader[Q] {
	return func(queryMods ...bob.Mod[*dialect.SelectQuery]) orm.Loader[Q] {
    return func(ctx context.Context, exec bob.Executor, retrieved any) error {
      loader, isLoader := retrieved.(T)
      if !isLoader {
        return fmt.Errorf("object %T cannot load %q", retrieved, name)
      }

      err := f(ctx, exec, loader, queryMods...)

      // Don't cause an issue due to missing relationships
      if errors.Is(err, sql.ErrNoRows) {
        return nil
      }

      return err
    }
  }
}

