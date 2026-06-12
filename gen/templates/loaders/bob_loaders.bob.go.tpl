{{$.Importer.Import "fmt"}}
{{$.Importer.Import "errors"}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "database/sql"}}
{{$.Importer.Import "sort"}}
{{$.Importer.Import "strings"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
{{if $.IsModelSplitFacade}}{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}{{end}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

var Preload = getPreloaders()

type preloaders struct {
		{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{if $.IsModelSplitFacade -}}
		{{$tAlias.UpSingular}} expand{{$tAlias.UpSingular}}Preloader
		{{else -}}
		{{$tAlias.UpSingular}} {{$.PreloaderType $table.Key}}
		{{end -}}
		{{end}}{{end}}
}

func getPreloaders() preloaders {
	return preloaders{
		{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{if $.IsModelSplitFacade -}}
		{{$tAlias.UpSingular}}: expand{{$tAlias.UpSingular}}Preloader{ {{$.BuildPreloaderFunc $table.Key}}() },
		{{else -}}
		{{$tAlias.UpSingular}}: {{$.BuildPreloaderFunc $table.Key}}(),
		{{end -}}
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
		{{if $.IsModelSplitFacade -}}
		{{$tAlias.UpSingular}} expand{{$tAlias.UpSingular}}ThenLoader[Q]
		{{else -}}
		{{$tAlias.UpSingular}} {{$.ThenLoaderType $table.Key}}[Q]
		{{end -}}
		{{end}}{{end}}
}

func getThenLoaders[Q orm.Loadable]() thenLoaders[Q] {
	return thenLoaders[Q]{
		{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{if $.IsModelSplitFacade -}}
		{{$tAlias.UpSingular}}: expand{{$tAlias.UpSingular}}ThenLoader[Q]{ {{$.BuildThenLoaderFunc $table.Key}}[Q]() },
		{{else -}}
		{{$tAlias.UpSingular}}: {{$.BuildThenLoaderFunc $table.Key}}[Q](),
		{{end -}}
		{{end}}{{end}}
	}
}

{{if $.IsModelSplitFacade -}}
{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
{{$tAlias := $.Aliases.Table $table.Key -}}
type expand{{$tAlias.UpSingular}}Preloader struct {
	{{$.PreloaderType $table.Key}}
}

func (l expand{{$tAlias.UpSingular}}Preloader) ForExpandMap(expands map[string]struct{}, opts ...ExpandLoadOption) ([]bob.Mod[*dialect.SelectQuery], error) {
	paths := make([]string, 0, len(expands))
	for path := range expands {
		paths = append(paths, path)
	}

	return l.ForExpandPaths(paths, opts...)
}

func (l expand{{$tAlias.UpSingular}}Preloader) ForExpandPaths(paths []string, opts ...ExpandLoadOption) ([]bob.Mod[*dialect.SelectQuery], error) {
	options := newExpandLoadOptions(opts...)
	tree, err := buildExpandTree(paths, options.maxDepth)
	if err != nil {
		return nil, err
	}

	preloadOpts, err := l.forExpandTree(tree, 0, options)
	if err != nil {
		return nil, err
	}

	mods := make([]bob.Mod[*dialect.SelectQuery], 0, len(preloadOpts))
	for _, opt := range preloadOpts {
		mod, ok := opt.(bob.Mod[*dialect.SelectQuery])
		if !ok {
			return nil, fmt.Errorf("expand preload option %T is not a select query mod", opt)
		}
		mods = append(mods, mod)
	}

	return mods, nil
}

func (l expand{{$tAlias.UpSingular}}Preloader) forExpandTree(tree expandTree, depth int, opts expandLoadOptions) ([]{{$.Dialect}}.PreloadOption, error) {
	if opts.maxDepth >= 0 && depth > opts.maxDepth {
		return nil, fmt.Errorf("expand path %q exceeds max depth %d", tree.path, opts.maxDepth)
	}

	mods := make([]{{$.Dialect}}.PreloadOption, 0, len(tree.children))
	for _, segment := range tree.sortedSegments() {
		child := *tree.children[segment]
		if child.computedTerminal(opts) {
			continue
		}

		switch segment {
		{{range $rel := $.Relationships.Get $table.Key -}}
		{{- if $rel.IsToMany -}}{{continue}}{{- end -}}
		{{- $relAlias := $tAlias.Relationship $rel.Name -}}
		{{- $fAlias := $.Aliases.Table $rel.Foreign -}}
		case {{snakecase $relAlias | quote}}:
			var childOpts []{{$.Dialect}}.PreloadOption
			{{if $.HasExpandPreloader $rel.Foreign -}}
			var err error
			childOpts, err = Preload.{{$fAlias.UpSingular}}.forExpandTree(child, depth+1, opts)
			if err != nil {
				return nil, err
			}
			{{else -}}
			if len(child.children) > 0 {
				return nil, fmt.Errorf("expand path %q cannot be nested because {{$fAlias.UpSingular}} has no generated preload relationships", child.path)
			}
			{{end -}}
			mods = append(mods, l.{{$relAlias}}(append(childOpts, {{$.Dialect}}.PreloadAs({{snakecase $relAlias | quote}}))...))
		{{end -}}
		default:
			return nil, fmt.Errorf("expand segment %q does not match a relationship on {{$tAlias.UpSingular}}", segment)
		}
	}

	return mods, nil
}

type expand{{$tAlias.UpSingular}}ThenLoader[Q orm.Loadable] struct {
	{{$.ThenLoaderType $table.Key}}[Q]
}

func (l expand{{$tAlias.UpSingular}}ThenLoader[Q]) ForExpandMap(expands map[string]struct{}, opts ...ExpandLoadOption) ([]bob.Mod[Q], error) {
	paths := make([]string, 0, len(expands))
	for path := range expands {
		paths = append(paths, path)
	}

	return l.ForExpandPaths(paths, opts...)
}

func (l expand{{$tAlias.UpSingular}}ThenLoader[Q]) ForExpandPaths(paths []string, opts ...ExpandLoadOption) ([]bob.Mod[Q], error) {
	options := newExpandLoadOptions(opts...)
	tree, err := buildExpandTree(paths, options.maxDepth)
	if err != nil {
		return nil, err
	}

	return l.forExpandTree(tree, 0, options)
}

func (l expand{{$tAlias.UpSingular}}ThenLoader[Q]) forExpandTree(tree expandTree, depth int, opts expandLoadOptions) ([]bob.Mod[Q], error) {
	if opts.maxDepth >= 0 && depth > opts.maxDepth {
		return nil, fmt.Errorf("expand path %q exceeds max depth %d", tree.path, opts.maxDepth)
	}

	mods := make([]bob.Mod[Q], 0, len(tree.children))
	for _, segment := range tree.sortedSegments() {
		child := *tree.children[segment]
		if child.computedTerminal(opts) {
			continue
		}

		switch segment {
		{{range $rel := $.Relationships.Get $table.Key -}}
		{{- $relAlias := $tAlias.Relationship $rel.Name -}}
		{{- $fAlias := $.Aliases.Table $rel.Foreign -}}
		case {{snakecase $relAlias | quote}}:
			{{if $.HasExpandThenLoader $rel.Foreign -}}
			childMods, err := SelectThenLoad.{{$fAlias.UpSingular}}.forExpandTree(child, depth+1, opts)
			if err != nil {
				return nil, err
			}
			mods = append(mods, l.{{$relAlias}}(childMods...))
			{{else -}}
			if len(child.children) > 0 {
				return nil, fmt.Errorf("expand path %q cannot be nested because {{$fAlias.UpSingular}} has no generated expand relationships", child.path)
			}
			mods = append(mods, l.{{$relAlias}}())
			{{end -}}
		{{end -}}
		default:
			return nil, fmt.Errorf("expand segment %q does not match a relationship on {{$tAlias.UpSingular}}", segment)
		}
	}

	return mods, nil
}

{{end}}{{end -}}
{{end -}}

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

type ExpandLoadOption func(*expandLoadOptions)

type expandLoadOptions struct {
	maxDepth         int
	computedTerminal func(path string) bool
}

func defaultExpandLoadOptions() expandLoadOptions {
	return expandLoadOptions{maxDepth: 10}
}

func newExpandLoadOptions(opts ...ExpandLoadOption) expandLoadOptions {
	options := defaultExpandLoadOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	return options
}

func WithMaxExpandDepth(depth int) ExpandLoadOption {
	return func(options *expandLoadOptions) {
		options.maxDepth = depth
	}
}

func WithComputedTerminal(fn func(path string) bool) ExpandLoadOption {
	return func(options *expandLoadOptions) {
		options.computedTerminal = fn
	}
}

type expandTree struct {
	path     string
	children map[string]*expandTree
}

func buildExpandTree(paths []string, maxDepth int) (expandTree, error) {
	root := expandTree{children: map[string]*expandTree{}}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		segments := strings.Split(path, ".")
		if maxDepth >= 0 && len(segments) > maxDepth {
			return expandTree{}, fmt.Errorf("expand path %q exceeds max depth %d", path, maxDepth)
		}

		node := &root
		currentPath := ""
		for _, segment := range segments {
			segment = strings.TrimSpace(segment)
			if segment == "" {
				return expandTree{}, fmt.Errorf("expand path %q contains an empty segment", path)
			}

			if currentPath == "" {
				currentPath = segment
			} else {
				currentPath += "." + segment
			}

			if node.children == nil {
				node.children = map[string]*expandTree{}
			}

			child := node.children[segment]
			if child == nil {
				child = &expandTree{path: currentPath, children: map[string]*expandTree{}}
				node.children[segment] = child
			}

			node = child
		}
	}

	return root, nil
}

func (tree expandTree) sortedSegments() []string {
	segments := make([]string, 0, len(tree.children))
	for segment := range tree.children {
		segments = append(segments, segment)
	}
	sort.Strings(segments)

	return segments
}

func (tree expandTree) computedTerminal(options expandLoadOptions) bool {
	return len(tree.children) == 0 && options.computedTerminal != nil && options.computedTerminal(tree.path)
}
