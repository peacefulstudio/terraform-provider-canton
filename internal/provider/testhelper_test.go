// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// schemaToTftype converts a resource schema to a tftypes.Object for constructing
// test Plan/State values.
func schemaToTftype(s rschema.Schema) tftypes.Object {
	attrTypes := map[string]tftypes.Type{}
	for name, attr := range s.Attributes {
		attrTypes[name] = attrFrameworkToTf(attr)
	}
	return tftypes.Object{AttributeTypes: attrTypes}
}

func attrFrameworkToTf(attr rschema.Attribute) tftypes.Type {
	switch a := attr.(type) {
	case rschema.StringAttribute:
		return tftypes.String
	case rschema.BoolAttribute:
		return tftypes.Bool
	case rschema.SetAttribute:
		return tftypes.Set{ElementType: elemType(a.ElementType)}
	default:
		panic(fmt.Sprintf("attrFrameworkToTf: unsupported attribute type %T — update test helper", attr))
	}
}

// elemType maps framework element types to tftypes. All sets in this provider
// use StringType; panics if a different framework element type is encountered,
// so test failures are explicit rather than silent type mismatches.
func elemType(t interface{}) tftypes.Type {
	switch t.(type) {
	case basetypes.StringTypable:
		return tftypes.String
	default:
		panic(fmt.Sprintf("elemType: unsupported element type %T — update test helper", t))
	}
}

// newTestPlan creates a tfsdk.Plan from a resource schema and a tftypes.Value.
func newTestPlan(s rschema.Schema, val tftypes.Value) tfsdk.Plan {
	return tfsdk.Plan{
		Schema: s,
		Raw:    val,
	}
}

// newTestState creates a tfsdk.State from a resource schema and a tftypes.Value.
func newTestState(s rschema.Schema, val tftypes.Value) tfsdk.State {
	return tfsdk.State{
		Schema: s,
		Raw:    val,
	}
}

// emptyState creates an empty tfsdk.State (no resource) from a resource schema.
func emptyState(s rschema.Schema) tfsdk.State {
	return tfsdk.State{
		Schema: s,
		Raw:    tftypes.NewValue(schemaToTftype(s), nil),
	}
}

// getResourceSchema calls Schema on the resource and returns the schema.
func getResourceSchema(r resource.Resource) rschema.Schema {
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	return schemaResp.Schema
}

// getState reads the resource model from state, failing the test if diagnostics contain errors.
func getState[T any](t *testing.T, state tfsdk.State) T {
	t.Helper()
	var m T
	diags := state.Get(context.Background(), &m)
	if diags.HasError() {
		t.Fatalf("failed to read state: %s", diags.Errors())
	}
	return m
}
