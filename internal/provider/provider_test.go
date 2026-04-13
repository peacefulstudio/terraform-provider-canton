// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNew(t *testing.T) {
	factory := New("1.0.0")
	p := factory()
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestProviderMetadata(t *testing.T) {
	p := &cantonProvider{version: "1.2.3"}
	resp := &provider.MetadataResponse{}
	p.Metadata(context.Background(), provider.MetadataRequest{}, resp)

	if resp.TypeName != "canton" {
		t.Errorf("expected type name 'canton', got %q", resp.TypeName)
	}
	if resp.Version != "1.2.3" {
		t.Errorf("expected version '1.2.3', got %q", resp.Version)
	}
}

func TestProviderSchema(t *testing.T) {
	p := &cantonProvider{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)

	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	attrs := resp.Schema.Attributes
	if _, ok := attrs["participant_url"]; !ok {
		t.Error("expected participant_url attribute")
	}
	if _, ok := attrs["oauth2"]; !ok {
		t.Error("expected oauth2 attribute")
	}
}

func TestProviderResources(t *testing.T) {
	p := &cantonProvider{}
	resources := p.Resources(context.Background())

	if len(resources) != 3 {
		t.Fatalf("expected 3 resource factories, got %d", len(resources))
	}

	// Verify each factory produces a non-nil resource.
	for i, factory := range resources {
		r := factory()
		if r == nil {
			t.Errorf("resource factory %d returned nil", i)
		}
	}
}

func TestProviderDataSources(t *testing.T) {
	p := &cantonProvider{}
	dataSources := p.DataSources(context.Background())

	if len(dataSources) != 3 {
		t.Fatalf("expected 3 data source factories, got %d", len(dataSources))
	}

	for i, factory := range dataSources {
		ds := factory()
		if ds == nil {
			t.Errorf("data source factory %d returned nil", i)
		}
	}
}

func TestProviderDataSources_TypeNames(t *testing.T) {
	p := &cantonProvider{}
	dataSources := p.DataSources(context.Background())

	expectedTypes := map[string]bool{
		"canton_party":   false,
		"canton_user":    false,
		"canton_parties": false,
	}

	for _, factory := range dataSources {
		ds := factory()
		metaResp := &datasource.MetadataResponse{}
		ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "canton"}, metaResp)
		if _, ok := expectedTypes[metaResp.TypeName]; !ok {
			t.Errorf("unexpected data source type: %s", metaResp.TypeName)
		}
		if expectedTypes[metaResp.TypeName] {
			t.Errorf("duplicate data source type: %s", metaResp.TypeName)
		}
		expectedTypes[metaResp.TypeName] = true
	}

	for name, found := range expectedTypes {
		if !found {
			t.Errorf("expected data source type %s not found", name)
		}
	}
}

func TestProviderResources_TypeNames(t *testing.T) {
	p := &cantonProvider{}
	resources := p.Resources(context.Background())

	expectedTypes := map[string]bool{
		"canton_party":       false,
		"canton_user":        false,
		"canton_user_rights": false,
	}

	for _, factory := range resources {
		r := factory()
		metaResp := &resource.MetadataResponse{}
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "canton"}, metaResp)
		if _, ok := expectedTypes[metaResp.TypeName]; !ok {
			t.Errorf("unexpected resource type: %s", metaResp.TypeName)
		}
		if expectedTypes[metaResp.TypeName] {
			t.Errorf("duplicate resource type: %s", metaResp.TypeName)
		}
		expectedTypes[metaResp.TypeName] = true
	}

	for name, found := range expectedTypes {
		if !found {
			t.Errorf("expected resource type %s not found", name)
		}
	}
}

func TestProviderDataSources_ReturnsNonNilSlice(t *testing.T) {
	p := &cantonProvider{}
	ds := p.DataSources(context.Background())
	if ds == nil {
		t.Error("expected non-nil slice (even if empty)")
	}
}

func TestStringValueOrEnv(t *testing.T) {
	tests := []struct {
		name     string
		value    types.String
		envVar   string
		envValue string
		want     string
	}{
		{
			name:   "uses value when set",
			value:  types.StringValue("from-config"),
			envVar: "TEST_UNUSED_VAR",
			want:   "from-config",
		},
		{
			name:     "falls back to env var",
			value:    types.StringNull(),
			envVar:   "TEST_STRING_VALUE_OR_ENV",
			envValue: "from-env",
			want:     "from-env",
		},
		{
			name:   "returns empty when neither set",
			value:  types.StringNull(),
			envVar: "TEST_NONEXISTENT_VAR_12345",
			want:   "",
		},
		{
			name:   "unknown value falls back to env",
			value:  types.StringUnknown(),
			envVar: "TEST_STRING_VALUE_UNKNOWN",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.envVar, tt.envValue)
			}
			got := stringValueOrEnv(tt.value, tt.envVar)
			if got != tt.want {
				t.Errorf("stringValueOrEnv() = %q, want %q", got, tt.want)
			}
		})
	}
}
