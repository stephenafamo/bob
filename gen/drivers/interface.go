// Package drivers talks to various database backends and retrieves table,
// column, type, and foreign key information
package drivers

import (
	"fmt"
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
	Tables    []Table `json:"tables"`
	ExtraInfo T       `json:"extra_info"`
}

type TablesInfo []TableInfo

type TableInfo struct {
	Key    string
	Schema string
	Name   string
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
type Constructor interface {
	// Load all constraints in the database, keyed by TableInfo.Key
	Constraints(ColumnFilter) (DBConstraints, error)

	// For tables
	TablesInfo(Filter) (TablesInfo, error)
	TableColumns(info TableInfo, filter ColumnFilter) (schema, name string, _ []Column, _ error)

	// For views
	ViewsInfo(Filter) (TablesInfo, error)
	ViewColumns(info TableInfo, filter ColumnFilter) (schema, name string, _ []Column, _ error)
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

	var tablesInfo, viewsInfo TablesInfo
	if tablesInfo, err = c.TablesInfo(tableFilter); err != nil {
		return nil, fmt.Errorf("unable to get table names: %w", err)
	}

	if viewsInfo, err = c.ViewsInfo(tableFilter); err != nil {
		return nil, fmt.Errorf("unable to get view names: %w", err)
	}

	colFilter := ParseColumnFilter(append(tablesInfo.Keys(), viewsInfo.Keys()...), includes, excludes)

	ret, err = tables(c, concurrency, tablesInfo, colFilter)
	if err != nil {
		return nil, fmt.Errorf("unable to load tables: %w", err)
	}

	v, err := views(c, concurrency, viewsInfo, colFilter)
	if err != nil {
		return nil, fmt.Errorf("unable to load views: %w", err)
	}

	ret = append(ret, v...)

	constraints, err := c.Constraints(colFilter)
	if err != nil {
		return nil, fmt.Errorf("unable to load constraints: %w", err)
	}
	for i, t := range ret {
		ret[i].PKey = constraints.PKs[t.Key]
		ret[i].FKeys = constraints.FKs[t.Key]
		ret[i].Uniques = constraints.Uniques[t.Key]
		ret[i].IsJoinTable = IsJoinTable(ret[i])
	}

	relationships := BuildRelationships(ret)
	for i, t := range ret {
		ret[i].Relationships = relationships[t.Key]
	}

	return ret, nil
}

func tables(c Constructor, concurrency int, infos TablesInfo, filter ColumnFilter) ([]Table, error) {
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Key < infos[j].Key
	})

	ret := make([]Table, len(infos))

	limiter := newConcurrencyLimiter(concurrency)
	wg := sync.WaitGroup{}
	errs := make(chan error, len(infos))
	for i, name := range infos {
		wg.Add(1)
		limiter.get()
		go func(i int, info TableInfo) {
			defer wg.Done()
			defer limiter.put()
			t, err := table(c, info, filter)
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
func table(c Constructor, info TableInfo, filter ColumnFilter) (Table, error) {
	var err error
	t := Table{
		Key: info.Key,
	}

	if t.Schema, t.Name, t.Columns, err = c.TableColumns(info, filter); err != nil {
		return Table{}, fmt.Errorf("unable to fetch table column info (%s): %w", info.Key, err)
	}

	return t, nil
}

// views returns the metadata for all views, minus the views
// specified in the excludes.
func views(c Constructor, concurrency int, infos TablesInfo, filter ColumnFilter) ([]Table, error) {
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Key < infos[j].Key
	})

	ret := make([]Table, len(infos))

	limiter := newConcurrencyLimiter(concurrency)
	wg := sync.WaitGroup{}
	errs := make(chan error, len(infos))
	for i, info := range infos {
		wg.Add(1)
		limiter.get()
		go func(i int, info TableInfo) {
			defer wg.Done()
			defer limiter.put()
			t, err := view(c, info, filter)
			if err != nil {
				errs <- err
				return
			}
			ret[i] = t
		}(i, info)
	}

	wg.Wait()

	// return first error occurred if any
	if len(errs) > 0 {
		return nil, <-errs
	}

	return ret, nil
}

// view returns columns info for a given view
func view(c Constructor, info TableInfo, filter ColumnFilter) (Table, error) {
	var err error
	t := Table{
		Key:    info.Key,
		Schema: info.Schema,
		Name:   info.Name,
	}

	if t.Schema, t.Name, t.Columns, err = c.ViewColumns(info, filter); err != nil {
		return Table{}, fmt.Errorf("unable to fetch view column info (%s): %w", info.Key, err)
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
