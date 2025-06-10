var TableNames = struct {
	{{range $table := .Tables -}}
	{{$tAlias := $.Aliases.Table $table.Key -}}
	{{$tAlias.UpPlural}} string
	{{end -}}
}{
	{{range $table := .Tables -}}
	{{$tAlias := $.Aliases.Table $table.Key -}}
	{{$tAlias.UpPlural}}: {{quote $table.Name}},
	{{end -}}
}

var ColumnNames = struct {
	{{range $table := .Tables -}}
	{{$tAlias := $.Aliases.Table $table.Key -}}
	{{$tAlias.UpPlural}} {{$tAlias.DownSingular}}ColumnNames
	{{end -}}
}{
	{{range $table := .Tables -}}
	{{$tAlias := $.Aliases.Table $table.Key -}}
	{{$tAlias.UpPlural}}: {{$tAlias.DownSingular}}ColumnNames{
		{{range $column := $table.Columns -}}
		{{- $colAlias := $tAlias.Column $column.Name -}}
		{{$colAlias}}: {{quote $column.Name}},
		{{end -}}
	},
	{{end -}}
}

{{block "helpers/where_variables" . -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
var (
	SelectWhere = Where[*dialect.SelectQuery]()
	UpdateWhere = Where[*dialect.UpdateQuery]()
	DeleteWhere = Where[*dialect.DeleteQuery]()
	OnConflictWhere = Where[*clause.ConflictClause]() // Used in ON CONFLICT DO UPDATE
)
{{- end}}

{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
func Where[Q {{$.Dialect}}.Filterable]() struct {
	{{range $table := .Tables -}}
	{{$tAlias := $.Aliases.Table $table.Key -}}
	{{$tAlias.UpPlural}} {{$tAlias.DownSingular}}Where[Q]
	{{end -}}
} {
	return struct {
		{{range $table := .Tables -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpPlural}} {{$tAlias.DownSingular}}Where[Q]
		{{end -}}
	}{
		{{range $table := .Tables -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpPlural}}: build{{$tAlias.UpSingular}}Where[Q]({{$tAlias.UpSingular}}Columns),
		{{end -}}
	}
}

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

{{$.Importer.Import "fmt"}}
{{$.Importer.Import "errors"}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "database/sql"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}

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

{{block "helpers/join_variables" . -}}
var (
	SelectJoins = getJoins[*dialect.SelectQuery]()
	UpdateJoins = getJoins[*dialect.UpdateQuery]()
	DeleteJoins = getJoins[*dialect.DeleteQuery]()
)
{{- end}}

type joinSet[Q interface{ aliasedAs(string) Q }] struct {
    InnerJoin Q
    LeftJoin Q
    RightJoin Q
}

func (j joinSet[Q]) AliasedAs(alias string) joinSet[Q] {
  return joinSet[Q]{
    InnerJoin: j.InnerJoin.aliasedAs(alias),
    LeftJoin: j.LeftJoin.aliasedAs(alias),
    RightJoin: j.RightJoin.aliasedAs(alias),
  }
}

type joins[Q dialect.Joinable] struct {
		{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpPlural}} joinSet[{{$tAlias.DownSingular}}Joins[Q]]
		{{end}}{{end}}
}

{{$.Importer.Import "github.com/stephenafamo/bob/clause"}}
func buildJoinSet[Q interface { aliasedAs(string) Q }, C any, F func(C, string) Q](c C, f F) joinSet[Q] {
	return joinSet[Q] {
	  InnerJoin: f(c, clause.InnerJoin),
	  LeftJoin: f(c, clause.LeftJoin),
	  RightJoin: f(c, clause.RightJoin),
	}
}

func getJoins[Q dialect.Joinable]() joins[Q] {
	return joins[Q]{
		{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpPlural}}: buildJoinSet[{{$tAlias.DownSingular}}Joins[Q]]({{$tAlias.UpSingular}}Columns, build{{$tAlias.UpSingular}}Joins),
		{{end}}{{end}}
	}
}

type modAs[Q any, C interface{ AliasedAs(string) C }] struct {
  c C
  f func(C) bob.Mod[Q]
}

func (m modAs[Q, C]) Apply(q Q) {
  m.f(m.c).Apply(q)
}

func (m modAs[Q, C]) AliasedAs(alias string) bob.Mod[Q] {
  m.c = m.c.AliasedAs(alias)
  return m
}

{{$.Importer.Import "hash/maphash"}}
func randInt() int64 {
	out := int64(new(maphash.Hash).Sum64())

	if out < 0 {
		return -out % 10000
	}

	return out % 10000
}

// ErrUniqueConstraint captures all unique constraint errors by explicitly leaving `s` empty.
var ErrUniqueConstraint = &UniqueConstraintError{s: ""}

type UniqueConstraintError struct {
  // schema is the schema where the unique constraint is defined.
  schema string
  // table is the name of the table where the unique constraint is defined.
  table string
  // columns are the columns constituting the unique constraint.
  columns []string
	// s is a string uniquely identifying the constraint in the raw error message returned from the database.
	s string
}

func (e *UniqueConstraintError) Error() string {
  return e.s
}

{{block "unique_constraint_error_detection_method" . -}}
func (e *UniqueConstraintError) Is(target error) bool {
  return false
}
{{end}}
