// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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

// dsSchemaToTftype converts a datasource schema to a tftypes.Object.
func dsSchemaToTftype(s dschema.Schema) tftypes.Object {
	attrTypes := map[string]tftypes.Type{}
	for name, attr := range s.Attributes {
		attrTypes[name] = dsAttrFrameworkToTf(attr)
	}
	for name, block := range s.Blocks {
		attrTypes[name] = dsBlockToTf(block)
	}
	return tftypes.Object{AttributeTypes: attrTypes}
}

func dsAttrFrameworkToTf(attr dschema.Attribute) tftypes.Type {
	switch a := attr.(type) {
	case dschema.StringAttribute:
		return tftypes.String
	case dschema.BoolAttribute:
		return tftypes.Bool
	case dschema.SetAttribute:
		return tftypes.Set{ElementType: elemType(a.ElementType)}
	case dschema.ListNestedAttribute:
		nested := map[string]tftypes.Type{}
		for n, na := range a.NestedObject.Attributes {
			nested[n] = dsAttrFrameworkToTf(na)
		}
		return tftypes.List{ElementType: tftypes.Object{AttributeTypes: nested}}
	default:
		panic(fmt.Sprintf("dsAttrFrameworkToTf: unsupported attribute type %T — update test helper", attr))
	}
}

func dsBlockToTf(block dschema.Block) tftypes.Type {
	switch b := block.(type) {
	case dschema.ListNestedBlock:
		nested := map[string]tftypes.Type{}
		for n, a := range b.NestedObject.Attributes {
			nested[n] = dsAttrFrameworkToTf(a)
		}
		return tftypes.List{ElementType: tftypes.Object{AttributeTypes: nested}}
	default:
		panic(fmt.Sprintf("dsBlockToTf: unsupported block type %T — update test helper", block))
	}
}

// newTestDSConfig creates a tfsdk.Config from a datasource schema and a tftypes.Value.
func newTestDSConfig(s dschema.Schema, val tftypes.Value) tfsdk.Config {
	return tfsdk.Config{
		Schema: s,
		Raw:    val,
	}
}

// getDatasourceSchema calls Schema on the datasource and returns the schema.
func getDatasourceSchema(ds datasource.DataSource) dschema.Schema {
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)
	return schemaResp.Schema
}

// emptyDSState creates an empty tfsdk.State (no resource) from a datasource schema.
func emptyDSState(s dschema.Schema) tfsdk.State {
	return tfsdk.State{
		Schema: s,
		Raw:    tftypes.NewValue(dsSchemaToTftype(s), nil),
	}
}

// emptyDSConfigValue creates a tftypes.Value with all attributes set to nil for a datasource schema.
func emptyDSConfigValue(s dschema.Schema) tftypes.Value {
	objType := dsSchemaToTftype(s)
	vals := map[string]tftypes.Value{}
	for name, attrType := range objType.AttributeTypes {
		vals[name] = tftypes.NewValue(attrType, nil)
	}
	return tftypes.NewValue(objType, vals)
}

// tftypesObjectValue creates a tftypes.Value from a type and a map of attribute values.
// nil values in the map produce tftypes null values.
func tftypesObjectValue(objType tftypes.Object, vals map[string]interface{}) tftypes.Value {
	tfVals := map[string]tftypes.Value{}
	for name, attrType := range objType.AttributeTypes {
		v, ok := vals[name]
		if !ok || v == nil {
			tfVals[name] = tftypes.NewValue(attrType, nil)
		} else {
			tfVals[name] = tftypes.NewValue(attrType, v)
		}
	}
	return tftypes.NewValue(objType, tfVals)
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
