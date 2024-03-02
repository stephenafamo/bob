package sqlite

import (
	"reflect"
	// "context"
	"testing"
	// "github.com/aarondl/opt/omit"
	// "github.com/google/go-cmp/cmp"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	// "github.com/stephenafamo/bob/orm"
	"github.com/stretchr/testify/assert" 

	
	// "github.com/stretchr/testify/assert" // assuming use of testify for assertions
)

func TestNewTable(t *testing.T) {
    tableName := "test_table"
    schema := "test_schema"
    table := NewTable[MyStruct, MyStructSetter](schema, tableName)

    if table.Name() != tableName {
        t.Errorf("Expected table name to be %s, got %s", tableName, table.Name())
    }
    
    if table.Schema() != schema {
        t.Errorf("Expected schema to be %s, got %s", schema, table.Schema())
    }
}


