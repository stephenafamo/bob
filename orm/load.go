package orm

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/aarondl/opt"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/internal/mappings"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/scan"
)

// Loadable is a constraint for types that can be loaded
type Loadable interface {
	AppendLoader(f ...bob.Loader)
	AppendMapperMod(f scan.MapperMod)
}

// Loader builds a query mod that makes an extra query after the object is retrieved
// it can be used to prevent N+1 queries by loading relationships in batches
type Loader[Q Loadable] func(ctx context.Context, exec bob.Executor, retrieved any) error

// Load is called after the original object is retrieved
func (l Loader[Q]) Load(ctx context.Context, exec bob.Executor, retrieved any) error {
	return l(ctx, exec, retrieved)
}

// Apply satisfies the bob.Mod[Q] interface
func (l Loader[Q]) Apply(q Q) {
	q.AppendLoader(l)
}

// modifyPreloader makes a Loader also work as a mod for a [Preloader]
func (l Loader[Q]) ModifyPreloadSettings(s *PreloadSettings[Q]) {
	s.ExtraLoader.AppendLoader(l)
}

func NewPreloadSettings[T any, Ts ~[]T, Q Loadable](cols []string) PreloadSettings[Q] {
	return PreloadSettings[Q]{
		Columns:     cols,
		ExtraLoader: NewAfterPreloader[T, Ts](),
	}
}

type preloadfilter = func(from, to string) []bob.Expression

type PreloadSettings[Q Loadable] struct {
	Columns     []string
	SubLoaders  []Preloader[Q]
	ExtraLoader *AfterPreloader
	Mods        [][]preloadfilter
	Alias       string
}

type PreloadOption[Q Loadable] interface {
	ModifyPreloadSettings(*PreloadSettings[Q])
}

type PreloadAs[Q Loadable] string

func (o PreloadAs[Q]) ModifyPreloadSettings(el *PreloadSettings[Q]) {
	el.Alias = string(o)
}

type PreloadOnly[Q Loadable] []string

func (o PreloadOnly[Q]) ModifyPreloadSettings(el *PreloadSettings[Q]) {
	if len(o) > 0 {
		el.Columns = internal.Only(el.Columns, o...)
	}
}

type PreloadExcept[Q Loadable] []string

func (e PreloadExcept[Q]) ModifyPreloadSettings(el *PreloadSettings[Q]) {
	if len(e) > 0 {
		el.Columns = internal.Except(el.Columns, e...)
	}
}

type PreloadWhere[Q Loadable] []preloadfilter

func (filters PreloadWhere[Q]) ModifyPreloadSettings(el *PreloadSettings[Q]) {
	diff := len(filters) - len(el.Mods)
	if diff > 0 {
		extra := make([][]preloadfilter, diff)
		el.Mods = append(el.Mods, extra...)
	}

	for i, filter := range filters {
		el.Mods[i] = append(el.Mods[i], filter)
	}
}

// Preloader builds a query mod that modifies the original query to retrieve related fields
// while it can be used as a queryMod, it does not have any direct effect.
// if using manually, the ApplyPreload method should be called
// with the query's context AFTER other mods have been applied
type Preloader[Q Loadable] func(parent string) (bob.Mod[Q], scan.MapperMod, []bob.Loader)

// Apply satisfies bob.Mod[*dialect.SelectQuery].
// 1. It modifies the query to join the preloading table and the extra columns to retrieve
// 2. It modifies the mapper to scan the new columns.
// 3. It calls the original object's Preload method with the loaded object
func (l Preloader[Q]) Apply(q Q) {
	mod, mapperMod, afterLoaders := l("")

	mod.Apply(q)                    // add preload columns
	q.AppendMapperMod(mapperMod)    // add mapper
	q.AppendLoader(afterLoaders...) // add the loader
}

// modifyPreloader makes a Loader also work as a mod for a [Preloader]
func (l Preloader[Q]) ModifyPreloadSettings(s *PreloadSettings[Q]) {
	s.SubLoaders = append(s.SubLoaders, l)
}

// NewAfterPreloader returns a new AfterPreloader based on the given types.
// The type parameters are captured in closures so that collecting and
// assembling the loaded objects needs no reflection at load time.
func NewAfterPreloader[T any, Ts ~[]T]() *AfterPreloader {
	var collected Ts
	return &AfterPreloader{
		appendCollected: func(v any) error {
			t, ok := v.(T)
			if !ok {
				return fmt.Errorf("expected to receive %T but got %T", *new(T), v)
			}
			collected = append(collected, t)
			return nil
		},
		numCollected: func() int { return len(collected) },
		toLoad: func() any {
			// a single object is passed as-is (T); many are passed as the
			// slice (Ts), matching the reflection-based implementation.
			if len(collected) == 1 {
				return collected[0]
			}
			return collected
		},
	}
}

// AfterPreloader is embedded in a Preloader to chain loading
// whenever a preloaded object is scanned, it should be collected with the Collect method
// The loading functions should be added with AppendLoader
// later, when this object is called like any other [bob.Loader], it
// calls the appended loaders with the collected objects
type AfterPreloader struct {
	funcs []bob.Loader

	// typed helpers set by [NewAfterPreloader]; they close over a Ts buffer
	appendCollected func(any) error
	numCollected    func() int
	toLoad          func() any
}

func (a *AfterPreloader) AppendLoader(fs ...bob.Loader) {
	a.funcs = append(a.funcs, fs...)
}

func (a *AfterPreloader) Collect(v any) error {
	if len(a.funcs) == 0 {
		return nil
	}

	return a.appendCollected(v)
}

func (a *AfterPreloader) Load(ctx context.Context, exec bob.Executor, _ any) error {
	if len(a.funcs) == 0 || a.numCollected() == 0 {
		return nil
	}

	obj := a.toLoad()

	for _, f := range a.funcs {
		if err := f.Load(ctx, exec, obj); err != nil {
			return err
		}
	}

	return nil
}

type Preloadable interface {
	Preload(name string, rel any) error
}

type PreloadRel[E bob.Expression] struct {
	Name  string
	Sides []PreloadSide[E]
}

type nameable[E bob.Expression] interface {
	NameExpr() E
	Alias() string
}

type PreloadSide[E bob.Expression] struct {
	From        nameable[E]
	To          nameable[E]
	FromColumns []string `yaml:"-"`
	ToColumns   []string `yaml:"-"`

	FromWhere []RelWhere `yaml:"from_where"`
	ToWhere   []RelWhere `yaml:"to_where"`
}

type PreloadableQuery interface {
	Loadable
	AppendJoin(clause.Join)
	AppendPreloadSelect(columns ...any)
}

// PreloadMapper returns a mapper for the preloaded child rows of a query.
// The prefix is the runtime-generated join alias followed by "." — every
// column belonging to the child is prefixed with it in the result set.
//
// The returned mapper must reproduce the LEFT JOIN semantics of the default
// reflection-based mapper: return a zero value (and no error) when every
// prefixed column is NULL (an unmatched join), and tolerate NULL values in
// columns whose struct fields cannot otherwise hold them.
type PreloadMapper[T any] func(prefix string) scan.Mapper[T]

// Preload builds a query mod to preload a relationship in the same query.
// If mapper is nil, it falls back to a reflection-based [scan.StructMapper]
// for the child columns.
//
// Related rows with the same join key are shared as a single instance across
// parent rows (the same semantics as ThenLoad), so preloaded structs should be
// treated as read-only.
func Preload[T Preloadable, Ts ~[]T, E bob.Expression, Q PreloadableQuery](rel PreloadRel[E], cols []string, mapper PreloadMapper[T], opts ...PreloadOption[Q]) Preloader[Q] {
	settings := NewPreloadSettings[T, Ts, Q](cols)
	for _, o := range opts {
		if o == nil {
			continue
		}
		o.ModifyPreloadSettings(&settings)
	}

	// the join columns on the preloaded table, identifying the related row for dedup
	var keyColumns []string
	if len(rel.Sides) > 0 {
		keyColumns = rel.Sides[len(rel.Sides)-1].ToColumns
	}

	return buildPreloader[T](func(parent string) (string, mods.QueryMods[Q]) {
		if parent == "" {
			parent = rel.Sides[0].From.Alias()
		}

		var alias string
		queryMods := make(mods.QueryMods[Q], 0, len(rel.Sides)+1)

		for i, side := range rel.Sides {
			alias = settings.Alias
			if settings.Alias == "" {
				alias = fmt.Sprintf("%s_%d", side.To.Alias(), bob.NextUniqueInt())
			}
			on := make([]bob.Expression, 0, len(side.FromColumns)+len(side.FromWhere)+len(side.ToWhere))
			for i, fromCol := range side.FromColumns {
				toCol := side.ToColumns[i]
				on = append(on, expr.OP(
					"=",
					expr.Quote(parent, fromCol),
					expr.Quote(alias, toCol),
				))
			}
			for _, from := range side.FromWhere {
				on = append(on, expr.OP(
					"=",
					expr.Quote(parent, from.Column),
					expr.Raw(from.SQLValue),
				))
			}
			for _, to := range side.ToWhere {
				on = append(on, expr.OP(
					"=",
					expr.Quote(parent, to.Column),
					expr.Raw(to.SQLValue),
				))
			}

			if len(settings.Mods) > i {
				for _, additional := range settings.Mods[i] {
					on = append(on, additional(parent, alias)...)
				}
			}

			queryMods = append(queryMods, mods.Join[Q](clause.Join{
				Type: clause.LeftJoin,
				To: clause.TableRef{
					Expression: side.To.NameExpr(),
					Alias:      alias,
				},
				On: on,
			}))

			// so the condition on the next "side" will be on the right table
			parent = alias
		}

		queryMods = append(queryMods, mods.Preload[Q]{
			expr.NewColumnsExpr(settings.Columns...).WithParent(alias).WithPrefix(alias + "."),
		})
		return alias, queryMods
	}, rel.Name, keyColumns, mapper, settings)
}

func buildPreloader[T any, Q Loadable](f func(string) (string, mods.QueryMods[Q]), name string, keyColumns []string, mapper PreloadMapper[T], opt PreloadSettings[Q]) Preloader[Q] {
	return func(parent string) (bob.Mod[Q], scan.MapperMod, []bob.Loader) {
		alias, queryMods := f(parent)
		prefix := alias + "."

		var mapperMods []scan.MapperMod
		extraLoaders := []bob.Loader{opt.ExtraLoader}

		for _, l := range opt.SubLoaders {
			queryMod, mapperMod, extraLoader := l(alias)
			if queryMod != nil {
				queryMods = append(queryMods, queryMod)
			}

			if mapperMod != nil {
				mapperMods = append(mapperMods, mapperMod)
			}

			if extraLoader != nil {
				extraLoaders = append(extraLoaders, extraLoader...)
			}
		}

		return queryMods, func(ctx context.Context, cols []string) (scan.BeforeFunc, scan.AfterMod) {
			usingStructMapper := mapper == nil

			var childMapper scan.Mapper[T]
			if mapper != nil {
				childMapper = mapper(prefix)
				if len(mapperMods) > 0 {
					childMapper = scan.Mod(childMapper, mapperMods...)
				}
			} else {
				childMapper = scan.StructMapper[T](
					scan.WithStructTagPrefix(prefix),
					scan.WithTypeConverter(NullTypeConverter{}),
					scan.WithRowValidator(rowValidator),
					scan.WithMapperMods(mapperMods...),
				)
			}
			before, after := childMapper(ctx, cols)

			legacy := func(link, retrieved any) error {
				loader, isLoader := retrieved.(Preloadable)
				if !isLoader {
					return fmt.Errorf("object %T cannot pre load", retrieved)
				}

				t, err := after(link)
				if err != nil {
					return err
				}

				if err = opt.ExtraLoader.Collect(t); err != nil {
					return err
				}

				return loader.Preload(name, t)
			}

			// per-execution state: scan runs this generator once per execution; no locking needed
			if usingStructMapper {
				if keyFromLink := preloadKeyFromLink[T](prefix, keyColumns, cols); keyFromLink != nil {
					return before, preloadDedupByLink(name, keyFromLink, after, legacy, opt.ExtraLoader)
				}
			} else if keyFromValue := preloadKeyFromValue[T](prefix, keyColumns, cols); keyFromValue != nil {
				return before, preloadDedupByValue(name, keyFromValue, after, legacy, opt.ExtraLoader)
			}

			// key not resolvable (e.g. dropped by PreloadOnly/PreloadExcept): keep the safe per-row path
			return before, legacy
		}, extraLoaders
	}
}

// dedup turns itself off after this many distinct keys with zero hits
const preloadDedupColdLimit = 1024

// preloadDedupCache is per-execution: a last-key fast path over a key->instance map
type preloadDedupCache[T any] struct {
	cache    map[string]T
	scratch  []byte
	lastKey  []byte
	lastVal  T
	haveLast bool
	everHit  bool
	off      bool
}

func newPreloadDedupCache[T any]() *preloadDedupCache[T] {
	return &preloadDedupCache[T]{cache: make(map[string]T)}
}

// fast path first: consecutive rows usually share the same key
func (c *preloadDedupCache[T]) lookup(key []byte) (T, bool) {
	if c.haveLast && bytes.Equal(key, c.lastKey) {
		c.everHit = true
		return c.lastVal, true
	}
	// string(key) in a map index does not allocate
	if t, hit := c.cache[string(key)]; hit {
		c.everHit = true
		c.lastKey = append(c.lastKey[:0], key...)
		c.lastVal = t
		c.haveLast = true
		return t, true
	}
	var zero T
	return zero, false
}

func (c *preloadDedupCache[T]) store(key []byte, t T) {
	c.cache[string(key)] = t

	// safety valve: no hit within the first preloadDedupColdLimit distinct keys
	// means the workload has no duplication — stop caching for this execution
	if !c.everHit && len(c.cache) >= preloadDedupColdLimit {
		*c = preloadDedupCache[T]{off: true}
		return
	}

	c.lastKey = append(c.lastKey[:0], key...)
	c.lastVal = t
	c.haveLast = true
}

// preloadDedupByLink keys before construction (reflection mapper): a duplicate
// row skips after(link) and with it any nested AfterMods/Collects — the cached
// instance already carries that data.
func preloadDedupByLink[T any](name string, keyFromLink func([]byte, any) ([]byte, bool), after func(any) (T, error), legacy scan.AfterMod, collector *AfterPreloader) scan.AfterMod {
	c := newPreloadDedupCache[T]()

	return func(link, retrieved any) error {
		if c.off {
			return legacy(link, retrieved)
		}

		loader, isLoader := retrieved.(Preloadable)
		if !isLoader {
			return fmt.Errorf("object %T cannot pre load", retrieved)
		}

		var ok bool
		c.scratch, ok = keyFromLink(c.scratch[:0], link)
		if !ok {
			// NULL join key = unmatched LEFT JOIN row: keep the legacy path, never cache
			return legacy(link, retrieved)
		}

		if t, hit := c.lookup(c.scratch); hit {
			return loader.Preload(name, t)
		}

		t, err := after(link)
		if err != nil {
			return err
		}

		c.store(c.scratch, t)

		if err = collector.Collect(t); err != nil {
			return err
		}

		return loader.Preload(name, t)
	}
}

// preloadDedupByValue keys after construction: a typed mapper's link is opaque,
// so the key is read from the built value; construction still runs per row.
func preloadDedupByValue[T any](name string, keyFromValue func([]byte, T) ([]byte, bool), after func(any) (T, error), legacy scan.AfterMod, collector *AfterPreloader) scan.AfterMod {
	c := newPreloadDedupCache[T]()

	return func(link, retrieved any) error {
		if c.off {
			return legacy(link, retrieved)
		}

		loader, isLoader := retrieved.(Preloadable)
		if !isLoader {
			return fmt.Errorf("object %T cannot pre load", retrieved)
		}

		t, err := after(link)
		if err != nil {
			return err
		}

		var ok bool
		c.scratch, ok = keyFromValue(c.scratch[:0], t)
		if !ok {
			// unmatched LEFT JOIN row: keep the legacy behavior, never cache
			if err = collector.Collect(t); err != nil {
				return err
			}
			return loader.Preload(name, t)
		}

		if cached, hit := c.lookup(c.scratch); hit {
			return loader.Preload(name, cached)
		}

		c.store(c.scratch, t)

		if err = collector.Collect(t); err != nil {
			return err
		}

		return loader.Preload(name, t)
	}
}

func preloadKeyColumnsSelected(prefix string, keyColumns, cols []string) bool {
	if len(keyColumns) == 0 {
		return false
	}

	for _, key := range keyColumns {
		if !slices.Contains(cols, prefix+key) {
			return false
		}
	}

	return true
}

// preloadKeyFromLink builds the pre-construction join-key extractor for the
// reflection-based StructMapper; returns nil if the key cannot be resolved.
func preloadKeyFromLink[T any](prefix string, keyColumns, cols []string) func([]byte, any) ([]byte, bool) {
	if !preloadKeyColumnsSelected(prefix, keyColumns, cols) {
		return nil
	}

	structCols, err := scan.StructMapperColumns[T]()
	if err != nil {
		return nil
	}

	mappable := make(map[string]struct{}, len(structCols))
	for _, c := range structCols {
		mappable[c] = struct{}{}
	}

	keyIdx := make([]int, len(keyColumns))
	for i := range keyIdx {
		keyIdx[i] = -1
	}

	linkLen := 0
	for _, col := range cols {
		name, hasPrefix := strings.CutPrefix(col, prefix)
		if !hasPrefix {
			continue
		}
		if _, ok := mappable[name]; !ok {
			continue
		}
		for i, key := range keyColumns {
			if name == key {
				keyIdx[i] = linkLen
			}
		}
		linkLen++
	}

	for _, idx := range keyIdx {
		if idx < 0 {
			return nil
		}
	}

	composite := len(keyIdx) > 1

	return func(dst []byte, link any) ([]byte, bool) {
		vals, ok := link.([]reflect.Value)
		if !ok || len(vals) != linkLen {
			return dst, false
		}

		for _, idx := range keyIdx {
			w, ok := vals[idx].Interface().(*wrapper)
			if !ok || w.IsNull {
				return dst, false
			}
			dst = appendPreloadKeyPointer(dst, w.V, composite)
		}

		return dst, true
	}
}

// preloadKeyFromValue builds the post-construction join-key extractor for a
// typed PreloadMapper (T must be *struct); returns nil if the key is unresolvable.
func preloadKeyFromValue[T any](prefix string, keyColumns, cols []string) func([]byte, T) ([]byte, bool) {
	if !preloadKeyColumnsSelected(prefix, keyColumns, cols) {
		return nil
	}

	typ := reflect.TypeFor[T]()
	if typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
		return nil
	}

	fieldNames := mappings.GetMappings(typ).All

	keyIdx := make([]int, 0, len(keyColumns))
	for _, key := range keyColumns {
		idx := -1
		for i, name := range fieldNames {
			if name == key {
				idx = i
				break
			}
		}
		if idx < 0 {
			return nil
		}
		keyIdx = append(keyIdx, idx)
	}

	composite := len(keyIdx) > 1

	return func(dst []byte, t T) ([]byte, bool) {
		rv := reflect.ValueOf(t)
		if !rv.IsValid() || rv.IsNil() {
			return dst, false
		}

		rv = rv.Elem()
		for _, idx := range keyIdx {
			dst = appendPreloadKeyValue(dst, rv.Field(idx), composite)
		}

		return dst, true
	}
}

// appendPreloadKeyValue appends an unambiguous encoding of one key column:
// composite-key parts are length-suffixed so parts cannot bleed into each other.
func appendPreloadKeyValue(dst []byte, v reflect.Value, composite bool) []byte {
	start := len(dst)

	switch v.Kind() {
	case reflect.String:
		dst = append(dst, v.String()...)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dst = strconv.AppendInt(dst, v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		dst = strconv.AppendUint(dst, v.Uint(), 10)
	case reflect.Bool:
		dst = strconv.AppendBool(dst, v.Bool())
	default:
		if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
			dst = append(dst, v.Bytes()...)
			break
		}
		dst = fmt.Appendf(dst, "%v", v.Interface())
	}

	return appendPreloadKeyLen(dst, start, composite)
}

// appendPreloadKeyPointer is appendPreloadKeyValue for a scan destination like wrapper.V
func appendPreloadKeyPointer(dst []byte, p any, composite bool) []byte {
	start := len(dst)

	switch x := p.(type) {
	case *string:
		dst = append(dst, *x...)
	case *[]byte:
		dst = append(dst, *x...)
	case *int:
		dst = strconv.AppendInt(dst, int64(*x), 10)
	case *int32:
		dst = strconv.AppendInt(dst, int64(*x), 10)
	case *int64:
		dst = strconv.AppendInt(dst, *x, 10)
	case *uint32:
		dst = strconv.AppendUint(dst, uint64(*x), 10)
	case *uint64:
		dst = strconv.AppendUint(dst, *x, 10)
	case *bool:
		dst = strconv.AppendBool(dst, *x)
	default:
		rv := reflect.ValueOf(p)
		if rv.Kind() == reflect.Pointer && !rv.IsNil() {
			return appendPreloadKeyValue(dst, rv.Elem(), composite)
		}
		dst = fmt.Appendf(dst, "%v", p)
	}

	return appendPreloadKeyLen(dst, start, composite)
}

func appendPreloadKeyLen(dst []byte, start int, composite bool) []byte {
	if !composite {
		return dst
	}

	partLen := len(dst) - start
	dst = append(dst, 0x00)
	dst = strconv.AppendInt(dst, int64(partLen), 10)
	dst = append(dst, 0x1f)

	return dst
}

// the row is valid if at least one column is not null
func rowValidator(_ []string, vals []reflect.Value) bool {
	for _, v := range vals {
		v, ok := v.Interface().(*wrapper)
		if !ok {
			return false
		}

		if !v.IsNull {
			return true
		}
	}

	return false
}

type wrapper struct {
	IsNull bool
	V      any
}

// Scan implements the sql.Scanner interface. If the wrapped type implements
// sql.Scanner then it will call that.
func (v *wrapper) Scan(value any) error {
	if value == nil {
		v.IsNull = true
		return nil
	}

	if scanner, ok := v.V.(sql.Scanner); ok {
		return scanner.Scan(value)
	}

	return opt.ConvertAssign(v.V, value)
}

// NullTypeConverter is a TypeConverter that skips NULL values during scanning even if the destination type does not support NULLs.
// This is useful when scanning complex queries with optional relationships while still wanting to re-use some generated structs.
//

// Example usage:

// Assuming the following generated type in the package "gen":
//   type Thing struct {
//     ID        string              `db:"id,pk" `
//     Name      string              `db:"name" `
//     Country   sql.Null[string]    `db:"country" `
//   }
//
// And the following custom struct that includes the generated type as a field:
//
// type myRow struct {
//     ... so many other cols
//     OptionalThing gen.Thing `db:"thing"` // will be populated by selecting thing.id, thing.name, thing.country
// }
//
// The "thing" table columns are loaded via a LEFT JOIN, so they could be all NULL: NullTypeConverter will make sure that the scan won't fail, leaving the struct empty (like Preload would do).
//
// bob.All(ctx, db,
//   psql.Select(
//      sm.Columns(
//         ...
//         gen.Things.Columns.WithPrefix("thing.")),
//      sm.From(...),
//		sm.LeftJoin(gen.Things.NameExpr()).As("thing").On(
//			gen.Things.Columns.AliasedAs("thing").ID.EQ(...),
//		),
// 	), scan.StructMapper[myRow](scan.WithTypeConverter(orm.NullTypeConverter{})))

type NullTypeConverter struct{}

// TypeToDestination implements the TypeConverter interface and returns a reflect.Value that wraps the destination type in a wrapper struct able to handle NULL values.
func (NullTypeConverter) TypeToDestination(typ reflect.Type) reflect.Value {
	val := reflect.ValueOf(&wrapper{
		V: reflect.New(typ).Interface(),
	})

	return val
}

// ValueFromDestination implements the TypeConverter interface and extracts the actual value from the wrapper struct.
func (NullTypeConverter) ValueFromDestination(val reflect.Value) reflect.Value {
	return val.Elem().FieldByName("V").Elem().Elem()
}
