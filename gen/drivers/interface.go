// Package drivers talks to various database backends and retrieves table,
// column, type, and foreign key information
package drivers

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/stephenafamo/bob/gen/importers"
)

// Interface abstracts either a side-effect imported driver or a binary
// that is called in order to produce the data required for generation.
type Interface[DBExtra, ConstraintExtra, IndexExtra any] interface {
	// The dialect
	Dialect() string
	// Assemble the database information into a nice struct
	Assemble(ctx context.Context) (*DBInfo[DBExtra, ConstraintExtra, IndexExtra], error)
	// Custom types defined by the driver
	Types() Types
}

type Type struct {
	// If this type is an alias of another type
	// this is useful to have custom randomization for a type e.g. xml
	AliasOf string `yaml:"alias_of"`
	// Imports needed for the type
	Imports importers.List `yaml:"imports"`
	// Any other types that this type depends on
	DependsOn []string `yaml:"depends_on"`
	// To be used in factory.random_type
	// a variable `f` of type `faker.Faker` is available
	RandomExpr string `yaml:"random_expr"`
	// Additional imports for the randomize expression
	RandomExprImports importers.List `yaml:"random_expr_imports"`
	// Set this to true if the randomization should not be tested
	// this is useful for low-cardinality types like bool
	NoRandomizationTest bool `yaml:"no_randomization_test"`
	// Set this to true if the test to see if the type implements
	// the scanner and valuer interfaces should be skipped
	// this is useful for types that are based on a primitive type
	NoScannerValuerTest bool `yaml:"no_scanner_valuer_test"`
	// CompareExpr is used to compare two values of this type
	// if not provided, == is used
	// Used AAA and BBB as placeholders for the two values
	CompareExpr string `yaml:"compare_expr"`
	// Imports needed for the compare expression
	CompareExprImports importers.List `yaml:"compare_expr_imports"`
}

type Types map[string]Type

// DBInfo is the database's table data and dialect.
type DBInfo[DBExtra, ConstraintExtra, IndexExtra any] struct {
	Tables       Tables[ConstraintExtra, IndexExtra] `json:"tables"`
	QueryFolders []QueryFolder                       `json:"query_folders"`
	Enums        []Enum                              `json:"enums"`
	ExtraInfo    DBExtra                             `json:"extra_info"`
	// DriverName is the module name of the underlying `database/sql` driver
	DriverName string `json:"driver_name"`
}

type Enum struct {
	Type   string
	Values []string
}

type TablesInfo []TableInfo

type TableInfo struct {
	Key     string
	Schema  string
	Name    string
	Comment string
}

func (t TablesInfo) Keys() []string {
	keys := make([]string, len(t))
	for i, info := range t {
		keys[i] = info.Key
	}
	return keys
}

// Constructor breaks down the functionality required to implement a driver
// such that the drivers.Tables method can be used to reduce duplication in driver
// implementations.
type Constructor[ConstraintExtra, IndexExtra any] interface {
	// Load basic info about all tables
	TablesInfo(context.Context, Filter) (TablesInfo, error)
	// Load details about a single table
	TableDetails(ctx context.Context, info TableInfo, filter ColumnFilter) (schema, name string, _ []Column, _ error)
	// Load all table comments, keyed by TableInfo.Key
	Comments(ctx context.Context) (map[string]string, error)
	// Load all constraints in the database, keyed by TableInfo.Key
	Constraints(context.Context, ColumnFilter) (DBConstraints[ConstraintExtra], error)
	// Load all indexes in the database, keyed by TableInfo.Key
	Indexes(ctx context.Context) (DBIndexes[IndexExtra], error)
}

// This returns the metadata for all tables,
// minus the tables specified in the excludes.
func BuildDBInfo[DBExtra, ConstraintExtra, IndexExtra any](
	ctx context.Context, c Constructor[ConstraintExtra, IndexExtra],
	concurrency int, only, except map[string][]string,
) ([]Table[ConstraintExtra, IndexExtra], error) {
	var err error
	var ret []Table[ConstraintExtra, IndexExtra]

	if concurrency < 1 {
		concurrency = 1
	}

	tableFilter := ParseTableFilter(only, except)

	var tablesInfo TablesInfo
	if tablesInfo, err = c.TablesInfo(ctx, tableFilter); err != nil {
		return nil, fmt.Errorf("unable to get table names: %w", err)
	}

	colFilter := ParseColumnFilter(tablesInfo.Keys(), only, except)

	ret, err = tables(ctx, c, concurrency, tablesInfo, colFilter)
	if err != nil {
		return nil, fmt.Errorf("unable to load tables: %w", err)
	}

	comments, err := c.Comments(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load comments: %w", err)
	}
	for i, t := range ret {
		ret[i].Comment = comments[t.Key]
	}

	constraints, err := c.Constraints(ctx, colFilter)
	if err != nil {
		return nil, fmt.Errorf("unable to load constraints: %w", err)
	}
	for i, t := range ret {
		ret[i].Constraints.Primary = constraints.PKs[t.Key]
		ret[i].Constraints.Foreign = constraints.FKs[t.Key]
		ret[i].Constraints.Uniques = constraints.Uniques[t.Key]
		ret[i].Constraints.Checks = constraints.Checks[t.Key]
	}

	indexes, err := c.Indexes(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load indexes: %w", err)
	}
	for i, t := range ret {
		ret[i].Indexes = indexes[t.Key]
	}

	return ret, nil
}

func tables[C, I any](ctx context.Context, c Constructor[C, I], concurrency int, infos TablesInfo, filter ColumnFilter) ([]Table[C, I], error) {
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Key < infos[j].Key
	})

	ret := make([]Table[C, I], len(infos))

	limiter := newConcurrencyLimiter(concurrency)
	wg := sync.WaitGroup{}
	errs := make(chan error, len(infos))
	for i, name := range infos {
		wg.Add(1)
		limiter.get()
		go func(i int, info TableInfo) {
			defer wg.Done()
			defer limiter.put()
			t, err := table(ctx, c, info, filter)
			if err != nil {
				errs <- err
				return
			}
			ret[i] = t
		}(i, name)
	}

	wg.Wait()

	// return first error occurred if any
	if len(errs) > 0 {
		return nil, <-errs
	}

	return ret, nil
}

// table returns columns info for a given table
func table[C, I any](ctx context.Context, c Constructor[C, I], info TableInfo, filter ColumnFilter) (Table[C, I], error) {
	var err error
	t := Table[C, I]{
		Key: info.Key,
	}

	if t.Schema, t.Name, t.Columns, err = c.TableDetails(ctx, info, filter); err != nil {
		return Table[C, I]{}, fmt.Errorf("unable to fetch table column info (%s): %w", info.Key, err)
	}

	return t, nil
}

// concurrencyCounter is a helper structure that can limit amount of concurrently processed requests
type concurrencyLimiter chan struct{}

func newConcurrencyLimiter(capacity int) concurrencyLimiter {
	ret := make(concurrencyLimiter, capacity)
	for i := 0; i < capacity; i++ {
		ret <- struct{}{}
	}

	return ret
}

func (c concurrencyLimiter) get() {
	<-c
}

func (c concurrencyLimiter) put() {
	c <- struct{}{}
}

func Skip(name string, include, exclude []string) bool {
	switch {
	case len(include) > 0:
		for _, i := range include {
			if i == name {
				return false
			}
		}
		return true

	case len(exclude) > 0:
		for _, i := range exclude {
			if i == name {
				return true
			}
		}
		return false

	default:
		return false
	}
}
