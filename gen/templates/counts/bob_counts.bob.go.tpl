{{$.Importer.Import "context"}}
{{$.Importer.Import "io"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
{{$.Importer.Import "github.com/stephenafamo/scan"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

{{block "helpers/count_variables" . -}}
var (
	PreloadCount = getPreloadCount()
	ThenLoadCount = getThenLoadCount[*dialect.SelectQuery]()
	InsertThenLoadCount = getThenLoadCount[*dialect.InsertQuery]()
)
{{- end}}

type preloadCounts struct {
	{{range $table := .Tables -}}
	{{- $rels := $.Relationships.Get $table.Key -}}
	{{- $hasToMany := false -}}
	{{- range $rel := $rels -}}
		{{- if $rel.IsToMany -}}{{- $hasToMany = true -}}{{- end -}}
	{{- end -}}
	{{- if $hasToMany -}}
	{{$tAlias := $.Aliases.Table $table.Key -}}
	{{$tAlias.UpSingular}} {{$tAlias.DownSingular}}CountPreloader
	{{end}}{{end}}
}

func getPreloadCount() preloadCounts {
	return preloadCounts{
		{{range $table := .Tables -}}
		{{- $rels := $.Relationships.Get $table.Key -}}
		{{- $hasToMany := false -}}
		{{- range $rel := $rels -}}
			{{- if $rel.IsToMany -}}{{- $hasToMany = true -}}{{- end -}}
		{{- end -}}
		{{- if $hasToMany -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpSingular}}: build{{$tAlias.UpSingular}}CountPreloader(),
		{{end}}{{end}}
	}
}

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

// countPreloadable is an interface for models that can have counts preloaded
type countPreloadable interface {
	PreloadCount(name string, count int64) error
}

// countPreloadMod is used to add a count subquery to the SELECT
type countPreloadMod[T countPreloadable] struct {
	name      string
	countExpr func(from string) bob.Expression
}

// Apply implements bob.Mod
func (c countPreloadMod[T]) Apply(q *dialect.SelectQuery) {
	c.applyCount(q, "")
}

// applyCount adds the count subquery to the query
func (c countPreloadMod[T]) applyCount(q *dialect.SelectQuery, parent string) {
	countCol := c.countExpr(parent)
	q.AppendPreloadSelect(aliasedExpr{expr: countCol, alias: "__count_" + c.name})
}

// aliasedExpr wraps an expression with an alias
type aliasedExpr struct {
	expr  bob.Expression
	alias string
}

func (a aliasedExpr) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	args, err := a.expr.WriteSQL(ctx, w, d, start)
	if err != nil {
		return nil, err
	}
	w.WriteString(" AS ")
	d.WriteQuoted(w, a.alias)
	return args, nil
}

// countPreloader returns a Preloader that adds a count subquery
func countPreloader[T countPreloadable](name string, countExpr func(from string) bob.Expression) {{$.Dialect}}.Preloader {
	return func(parent string) (bob.Mod[*dialect.SelectQuery], scan.MapperMod, []bob.Loader) {
		m := countPreloadMod[T]{
			name:      name,
			countExpr: countExpr,
		}

		queryMod := bob.ModFunc[*dialect.SelectQuery](func(q *dialect.SelectQuery) {
			m.applyCount(q, parent)
		})

		mapperMod := func(ctx context.Context, cols []string) (scan.BeforeFunc, scan.AfterMod) {
			// Find the count column
			countColName := "__count_" + name
			colIndex := -1
			for i, col := range cols {
				if col == countColName {
					colIndex = i
					break
				}
			}

			return func(r *scan.Row) (any, error) {
					if colIndex >= 0 {
						var count *int64
						r.ScheduleScanByIndex(colIndex, &count)
						return &count, nil
					}
					return nil, nil
				}, func(link, retrieved any) error {
					if link == nil {
						return nil
					}
					countPtr, ok := link.(**int64)
					if !ok || countPtr == nil || *countPtr == nil {
						return nil
					}

					loader, isLoader := retrieved.(countPreloadable)
					if !isLoader {
						return nil
					}

					return loader.PreloadCount(name, **countPtr)
				}
		}

		return queryMod, mapperMod, nil
	}
}
