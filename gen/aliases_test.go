package gen

import (
	"testing"

	"github.com/stephenafamo/bob/gen/drivers"
)

func expectPanic(t *testing.T, name string, fn func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic in %s, but did not panic", name)
		} else {
			t.Logf("[%s] Passed: caught panic: %v", name, r)
		}
	}()
	fn()
}

// This testcase depends on issue #451
func TestAliasClash_SelfReferencingRelationships(t *testing.T) {
	tables := []drivers.Table[any, any]{
		{
			Key: "publicv2.instantiated_ports",
			Columns: []drivers.Column{
				{Name: "node_template_id"},
				{Name: "node_workflow_id"},
				{Name: "node_id"},
				{Name: "port_id"},
			},
			Constraints: drivers.Constraints[any]{
				Primary: &drivers.Constraint[any]{
					Columns: []string{"node_template_id", "node_workflow_id", "node_id", "port_id"},
				},
			},
		},
		{
			Key: "publicv2.instantiated_port_connections",
			Columns: []drivers.Column{
				{Name: "instantiated_port_node_template_id"},
				{Name: "instantiated_port_node_workflow_id"},
				{Name: "instantiated_port_node_id"},
				{Name: "instantiated_port_port_id"},

				{Name: "connected_to_port_node_template_id"},
				{Name: "connected_to_port_node_workflow_id"},
				{Name: "connected_to_port_node_id"},
				{Name: "connected_to_port_port_id"},
			},
			Constraints: drivers.Constraints[any]{
				Primary: &drivers.Constraint[any]{
					Columns: []string{
						"instantiated_port_node_template_id",
						"instantiated_port_node_workflow_id",
						"instantiated_port_node_id",
						"instantiated_port_port_id",
						"connected_to_port_node_template_id",
						"connected_to_port_node_workflow_id",
						"connected_to_port_node_id",
						"connected_to_port_port_id",
					},
				},
				Foreign: []drivers.ForeignKey[any]{
					{
						Constraint: drivers.Constraint[any]{
							Name:    "fk_instantiated_port_connections_instantiated_port",
							Columns: []string{"instantiated_port_node_template_id", "instantiated_port_node_workflow_id", "instantiated_port_node_id", "instantiated_port_port_id"},
						},
						ForeignTable:   "publicv2.instantiated_ports",
						ForeignColumns: []string{"node_template_id", "node_workflow_id", "node_id", "port_id"},
					},
					{
						Constraint: drivers.Constraint[any]{
							Name:    "fk_instantiated_port_connections_connected_to",
							Columns: []string{"connected_to_port_node_template_id", "connected_to_port_node_workflow_id", "connected_to_port_node_id", "connected_to_port_port_id"},
						},
						ForeignTable:   "publicv2.instantiated_ports",
						ForeignColumns: []string{"node_template_id", "node_workflow_id", "node_id", "port_id"},
					},
				},
			},
		},
	}

	// Convert to generic drivers.Tables type
	dtab := drivers.Tables[any, any](tables)

	// Build relationships from foreign keys
	rels := buildRelationships(tables)

	// Expect alias clash during alias init
	expectPanic(t, "SelfReferencingRelationships", func() {
		initAliases(drivers.Aliases{}, dtab, rels)
	})
}
