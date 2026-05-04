// Copyright (c) 2026 Peaceful Studio OÜ
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	damlclient "github.com/noders-team/go-daml/pkg/client"
	"github.com/noders-team/go-daml/pkg/model"
)

func TestPartyDataSource_Metadata(t *testing.T) {
	ds := NewPartyDataSource()
	resp := &datasource.MetadataResponse{}
	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "canton"}, resp)

	if resp.TypeName != "canton_party" {
		t.Errorf("expected type name 'canton_party', got %q", resp.TypeName)
	}
}

func TestPartyDataSource_Schema(t *testing.T) {
	ds := NewPartyDataSource()
	s := getDatasourceSchema(ds)

	if s.Description == "" {
		t.Error("expected non-empty schema description")
	}

	if _, ok := s.Attributes["party_id"]; !ok {
		t.Error("expected party_id attribute")
	}
	if _, ok := s.Attributes["is_local"]; !ok {
		t.Error("expected is_local attribute")
	}
}

func TestPartyDataSource_Configure_NilProviderData(t *testing.T) {
	ds := &partyDataSource{}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Error("expected no errors for nil provider data")
	}
}

func TestPartyDataSource_Configure_WrongType(t *testing.T) {
	ds := &partyDataSource{}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "wrong"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for wrong provider data type")
	}
}

func TestPartyDataSource_Configure_NilService(t *testing.T) {
	ds := &partyDataSource{}
	resp := &datasource.ConfigureResponse{}
	clients := &cantonClients{binding: &damlclient.DamlBindingClient{}}
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: clients}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for nil party management service")
	}
}

func TestPartyDataSource_Configure_Success(t *testing.T) {
	ds := &partyDataSource{}
	resp := &datasource.ConfigureResponse{}
	mock := &mockPartyManagement{}
	clients := &cantonClients{binding: &damlclient.DamlBindingClient{PartyMng: mock}}
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: clients}, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected errors: %s", resp.Diagnostics.Errors())
	}
	if ds.partyMng != mock {
		t.Error("expected partyMng to be set")
	}
}

func TestPartyDataSource_Read_Success(t *testing.T) {
	mock := &mockPartyManagement{
		GetPartiesFn: func(_ context.Context, parties []string, _ string) ([]*model.PartyDetails, error) {
			if len(parties) != 1 || parties[0] != "party::12345" {
				t.Fatalf("unexpected party ID: %v", parties)
			}
			return []*model.PartyDetails{{Party: "party::12345", IsLocal: true}}, nil
		},
	}
	ds := &partyDataSource{partyMng: mock}
	s := getDatasourceSchema(ds)
	objType := dsSchemaToTftype(s)

	config := newTestDSConfig(s, tftypesObjectValue(objType, map[string]interface{}{
		"party_id": "party::12345",
		"is_local": nil,
	}))

	resp := &datasource.ReadResponse{State: emptyDSState(s)}
	ds.Read(context.Background(), datasource.ReadRequest{Config: config}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", resp.Diagnostics.Errors())
	}

	state := getState[partyDataSourceModel](t, resp.State)
	if state.PartyID.ValueString() != "party::12345" {
		t.Errorf("expected party_id 'party::12345', got %q", state.PartyID.ValueString())
	}
	if !state.IsLocal.ValueBool() {
		t.Error("expected is_local to be true")
	}
}

func TestPartyDataSource_Read_NotFound(t *testing.T) {
	mock := &mockPartyManagement{
		GetPartiesFn: func(_ context.Context, _ []string, _ string) ([]*model.PartyDetails, error) {
			return []*model.PartyDetails{}, nil
		},
	}
	ds := &partyDataSource{partyMng: mock}
	s := getDatasourceSchema(ds)
	objType := dsSchemaToTftype(s)

	config := newTestDSConfig(s, tftypesObjectValue(objType, map[string]interface{}{
		"party_id": "nonexistent::party",
		"is_local": nil,
	}))

	resp := &datasource.ReadResponse{State: emptyDSState(s)}
	ds.Read(context.Background(), datasource.ReadRequest{Config: config}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for not-found party")
	}
}

func TestPartyDataSource_Read_NilPartyEntry(t *testing.T) {
	mock := &mockPartyManagement{
		GetPartiesFn: func(_ context.Context, _ []string, _ string) ([]*model.PartyDetails, error) {
			return []*model.PartyDetails{nil}, nil
		},
	}
	ds := &partyDataSource{partyMng: mock}
	s := getDatasourceSchema(ds)
	objType := dsSchemaToTftype(s)

	config := newTestDSConfig(s, tftypesObjectValue(objType, map[string]interface{}{
		"party_id": "party::12345",
		"is_local": nil,
	}))

	resp := &datasource.ReadResponse{State: emptyDSState(s)}
	ds.Read(context.Background(), datasource.ReadRequest{Config: config}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for nil party entry")
	}
}

func TestPartyDataSource_Read_Error(t *testing.T) {
	mock := &mockPartyManagement{
		GetPartiesFn: func(_ context.Context, _ []string, _ string) ([]*model.PartyDetails, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	ds := &partyDataSource{partyMng: mock}
	s := getDatasourceSchema(ds)
	objType := dsSchemaToTftype(s)

	config := newTestDSConfig(s, tftypesObjectValue(objType, map[string]interface{}{
		"party_id": "party::12345",
		"is_local": nil,
	}))

	resp := &datasource.ReadResponse{State: emptyDSState(s)}
	ds.Read(context.Background(), datasource.ReadRequest{Config: config}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error on API failure")
	}
}
