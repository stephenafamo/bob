// Package drivers talks to various database backends and retrieves table,
// column, type, and foreign key information
package drivers

import (
	"fmt"
	"os"
	"sort"
	"sync"
)

// Interface abstracts either a side-effect imported driver or a binary
// that is called in order to produce the data required for generation.
type Interface[T any] interface {
	// Assemble the database information into a nice struct
	Assemble() (*DBInfo[T], error)
}

// DBInfo is the database's table data and dialect.
type DBInfo[T any] struct {
	Schema    string  `json:"schema"`
	Tables    []Table `json:"tables"`
	ExtraInfo T       `json:"extra_info"`
}

// Constructor breaks down the functionality required to implement a driver
// such that the drivers.Tables method can be used to reduce duplication in driver
// implementations.
type Constructor interface {
	// Load all constraints in the database, keyed by table
	Constraints(ColumnFilter) (DBConstraints, error)

	// For tables
	TableNames(Filter) ([]string, error)
	TableColumns(tableName string, filter ColumnFilter) ([]Column, error)

	// For views
	ViewNames(Filter) ([]string, error)
	ViewColumns(tableName string, filter ColumnFilter) ([]Column, error)
}

// TablesConcurrently is a concurrent version of Tables. It returns the
// metadata for all tables, minus the tables specified in the excludes.
func Tables(c Constructor, concurrency int, includes, excludes []string) ([]Table, error) {
	var err error
	var ret []Table

	if concurrency < 1 {
		concurrency = 1
	}

	tableFilter := Filter{
		Include: TablesFromList(includes),
		Exclude: TablesFromList(excludes),
	}

	var tableNames, viewNames []string
	if tableNames, err = c.TableNames(tableFilter); err != nil {
		return nil, fmt.Errorf("unable to get table names: %w", err)
	}

	if viewNames, err = c.ViewNames(tableFilter); err != nil {
		return nil, fmt.Errorf("unable to get view names: %w", err)
	}

	colFilter := ParseColumnFilter(append(tableNames, viewNames...), includes, excludes)

	ret, err = tables(c, concurrency, tableNames, colFilter)
	if err != nil {
		return nil, fmt.Errorf("unable to load tables: %w", err)
	}

	v, err := views(c, concurrency, viewNames, colFilter)
	if err != nil {
		return nil, fmt.Errorf("unable to load views: %w", err)
	}

	ret = append(ret, v...)

	constraints, err := c.Constraints(colFilter)
	if err != nil {
		return nil, fmt.Errorf("unable to load constraints: %w", err)
	}
	for i, t := range ret {
		ret[i].PKey = constraints.PKs[t.Name]
		ret[i].FKeys = constraints.FKs[t.Name]
		ret[i].Uniques = constraints.Uniques[t.Name]
		ret[i].IsJoinTable = IsJoinTable(ret[i])
	}

	relationships := BuildRelationships(ret)
	for i, t := range ret {
		ret[i].Relationships = relationships[t.Name]
	}

	return ret, nil
}

func tables(c Constructor, concurrency int, names []string, filter ColumnFilter) ([]Table, error) {
	sort.Strings(names)
	ret := make([]Table, len(names))

	limiter := newConcurrencyLimiter(concurrency)
	wg := sync.WaitGroup{}
	errs := make(chan error, len(names))
	for i, name := range names {
		wg.Add(1)
		limiter.get()
		go func(i int, name string) {
			defer wg.Done()
			defer limiter.put()
			t, err := table(c, name, filter)
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
func table(c Constructor, name string, filter ColumnFilter) (Table, error) {
	var err error
	t := &Table{
		Name: name,
	}

	if t.Columns, err = c.TableColumns(name, filter); err != nil {
		return Table{}, fmt.Errorf("unable to fetch table column info (%s): %w", name, err)
	}

	return *t, nil
}

// views returns the metadata for all views, minus the views
// specified in the excludes.
func views(c Constructor, concurrency int, names []string, filter ColumnFilter) ([]Table, error) {
	sort.Strings(names)

	ret := make([]Table, len(names))

	limiter := newConcurrencyLimiter(concurrency)
	wg := sync.WaitGroup{}
	errs := make(chan error, len(names))
	for i, name := range names {
		wg.Add(1)
		limiter.get()
		go func(i int, name string) {
			defer wg.Done()
			defer limiter.put()
			t, err := view(c, name, filter)
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

// view returns columns info for a given view
func view(c Constructor, name string, filter ColumnFilter) (Table, error) {
	var err error
	t := Table{
		Name: name,
	}

	if t.Columns, err = c.ViewColumns(name, filter); err != nil {
		return Table{}, fmt.Errorf("unable to fetch view column info (%s): %w", name, err)
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

// DefaultEnv grabs a value from the environment or a default.
// This is shared by drivers to get config for testing.
func DefaultEnv(key, def string) string {
	val := os.Getenv(key)
	if len(val) == 0 {
		val = def
	}
	return val
}
