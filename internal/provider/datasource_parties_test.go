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

func TestPartiesDataSource_Metadata(t *testing.T) {
	ds := NewPartiesDataSource()
	resp := &datasource.MetadataResponse{}
	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "canton"}, resp)

	if resp.TypeName != "canton_parties" {
		t.Errorf("expected type name 'canton_parties', got %q", resp.TypeName)
	}
}

func TestPartiesDataSource_Schema(t *testing.T) {
	ds := NewPartiesDataSource()
	s := getDatasourceSchema(ds)

	if s.Description == "" {
		t.Error("expected non-empty schema description")
	}

	if _, ok := s.Attributes["parties"]; !ok {
		t.Error("expected parties attribute")
	}
}

func TestPartiesDataSource_Configure_NilProviderData(t *testing.T) {
	ds := &partiesDataSource{}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Error("expected no errors for nil provider data")
	}
}

func TestPartiesDataSource_Configure_WrongType(t *testing.T) {
	ds := &partiesDataSource{}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "wrong"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for wrong provider data type")
	}
}

func TestPartiesDataSource_Configure_NilService(t *testing.T) {
	ds := &partiesDataSource{}
	resp := &datasource.ConfigureResponse{}
	clients := &cantonClients{binding: &damlclient.DamlBindingClient{}}
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: clients}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for nil party management service")
	}
}

func TestPartiesDataSource_Configure_Success(t *testing.T) {
	ds := &partiesDataSource{}
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

func TestPartiesDataSource_Read_Success(t *testing.T) {
	mock := &mockPartyManagement{
		ListKnownPartiesFn: func(_ context.Context, pageToken string, _ int32, _ string) (*model.ListKnownPartiesResponse, error) {
			if pageToken != "" {
				t.Fatalf("expected empty initial page token, got %q", pageToken)
			}
			return &model.ListKnownPartiesResponse{
				PartyDetails: []*model.PartyDetails{
					{Party: "party::alice", IsLocal: true},
					{Party: "party::bob", IsLocal: false},
				},
			}, nil
		},
	}
	ds := &partiesDataSource{partyMng: mock}
	s := getDatasourceSchema(ds)

	resp := &datasource.ReadResponse{State: emptyDSState(s)}
	ds.Read(context.Background(), datasource.ReadRequest{Config: newTestDSConfig(s, emptyDSConfigValue(s))}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", resp.Diagnostics.Errors())
	}

	state := getState[partiesDataSourceModel](t, resp.State)
	if len(state.Parties) != 2 {
		t.Fatalf("expected 2 parties, got %d", len(state.Parties))
	}
	if state.Parties[0].PartyID.ValueString() != "party::alice" {
		t.Errorf("expected first party 'party::alice', got %q", state.Parties[0].PartyID.ValueString())
	}
	if !state.Parties[0].IsLocal.ValueBool() {
		t.Error("expected first party to be local")
	}
	if state.Parties[1].PartyID.ValueString() != "party::bob" {
		t.Errorf("expected second party 'party::bob', got %q", state.Parties[1].PartyID.ValueString())
	}
	if state.Parties[1].IsLocal.ValueBool() {
		t.Error("expected second party to not be local")
	}
}

func TestPartiesDataSource_Read_Pagination(t *testing.T) {
	callCount := 0
	mock := &mockPartyManagement{
		ListKnownPartiesFn: func(_ context.Context, pageToken string, _ int32, _ string) (*model.ListKnownPartiesResponse, error) {
			callCount++
			switch pageToken {
			case "":
				return &model.ListKnownPartiesResponse{
					PartyDetails:  []*model.PartyDetails{{Party: "party::page1", IsLocal: true}},
					NextPageToken: "token-2",
				}, nil
			case "token-2":
				return &model.ListKnownPartiesResponse{
					PartyDetails:  []*model.PartyDetails{{Party: "party::page2", IsLocal: false}},
					NextPageToken: "",
				}, nil
			default:
				t.Fatalf("unexpected page token: %s", pageToken)
				return nil, nil
			}
		},
	}
	ds := &partiesDataSource{partyMng: mock}
	s := getDatasourceSchema(ds)

	resp := &datasource.ReadResponse{State: emptyDSState(s)}
	ds.Read(context.Background(), datasource.ReadRequest{Config: newTestDSConfig(s, emptyDSConfigValue(s))}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", resp.Diagnostics.Errors())
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls for pagination, got %d", callCount)
	}
	if resp.Diagnostics.WarningsCount() != 0 {
		t.Errorf("expected no warnings for complete pagination, got %d", resp.Diagnostics.WarningsCount())
	}

	state := getState[partiesDataSourceModel](t, resp.State)
	if len(state.Parties) != 2 {
		t.Fatalf("expected 2 parties across pages, got %d", len(state.Parties))
	}
	if state.Parties[0].PartyID.ValueString() != "party::page1" {
		t.Errorf("expected first party 'party::page1', got %q", state.Parties[0].PartyID.ValueString())
	}
	if state.Parties[1].PartyID.ValueString() != "party::page2" {
		t.Errorf("expected second party 'party::page2', got %q", state.Parties[1].PartyID.ValueString())
	}
}

func TestPartiesDataSource_Read_Empty(t *testing.T) {
	mock := &mockPartyManagement{
		ListKnownPartiesFn: func(_ context.Context, _ string, _ int32, _ string) (*model.ListKnownPartiesResponse, error) {
			return &model.ListKnownPartiesResponse{PartyDetails: []*model.PartyDetails{}}, nil
		},
	}
	ds := &partiesDataSource{partyMng: mock}
	s := getDatasourceSchema(ds)

	resp := &datasource.ReadResponse{State: emptyDSState(s)}
	ds.Read(context.Background(), datasource.ReadRequest{Config: newTestDSConfig(s, emptyDSConfigValue(s))}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", resp.Diagnostics.Errors())
	}

	state := getState[partiesDataSourceModel](t, resp.State)
	if len(state.Parties) != 0 {
		t.Errorf("expected 0 parties, got %d", len(state.Parties))
	}
}

func TestPartiesDataSource_Read_Error(t *testing.T) {
	mock := &mockPartyManagement{
		ListKnownPartiesFn: func(_ context.Context, _ string, _ int32, _ string) (*model.ListKnownPartiesResponse, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	ds := &partiesDataSource{partyMng: mock}
	s := getDatasourceSchema(ds)

	resp := &datasource.ReadResponse{State: emptyDSState(s)}
	ds.Read(context.Background(), datasource.ReadRequest{Config: newTestDSConfig(s, emptyDSConfigValue(s))}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error on API failure")
	}
}

func TestPartiesDataSource_Read_NilResult(t *testing.T) {
	mock := &mockPartyManagement{
		ListKnownPartiesFn: func(_ context.Context, _ string, _ int32, _ string) (*model.ListKnownPartiesResponse, error) {
			return nil, nil
		},
	}
	ds := &partiesDataSource{partyMng: mock}
	s := getDatasourceSchema(ds)

	resp := &datasource.ReadResponse{State: emptyDSState(s)}
	ds.Read(context.Background(), datasource.ReadRequest{Config: newTestDSConfig(s, emptyDSConfigValue(s))}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for nil result")
	}
}

func TestPartiesDataSource_Read_PaginationCycle(t *testing.T) {
	callCount := 0
	mock := &mockPartyManagement{
		ListKnownPartiesFn: func(_ context.Context, _ string, _ int32, _ string) (*model.ListKnownPartiesResponse, error) {
			callCount++
			return &model.ListKnownPartiesResponse{
				PartyDetails:  []*model.PartyDetails{{Party: "party::loop", IsLocal: true}},
				NextPageToken: "same-token",
			}, nil
		},
	}
	ds := &partiesDataSource{partyMng: mock}
	s := getDatasourceSchema(ds)

	resp := &datasource.ReadResponse{State: emptyDSState(s)}
	ds.Read(context.Background(), datasource.ReadRequest{Config: newTestDSConfig(s, emptyDSConfigValue(s))}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for pagination cycle")
	}
	if callCount != 2 {
		t.Errorf("expected cycle detected in 2 API calls, got %d", callCount)
	}
}

func TestPartiesDataSource_Read_PaginationError(t *testing.T) {
	mock := &mockPartyManagement{
		ListKnownPartiesFn: func(_ context.Context, pageToken string, _ int32, _ string) (*model.ListKnownPartiesResponse, error) {
			if pageToken == "" {
				return &model.ListKnownPartiesResponse{
					PartyDetails:  []*model.PartyDetails{{Party: "party::page1", IsLocal: true}},
					NextPageToken: "token-2",
				}, nil
			}
			return nil, fmt.Errorf("server error on page 2")
		},
	}
	ds := &partiesDataSource{partyMng: mock}
	s := getDatasourceSchema(ds)

	resp := &datasource.ReadResponse{State: emptyDSState(s)}
	ds.Read(context.Background(), datasource.ReadRequest{Config: newTestDSConfig(s, emptyDSConfigValue(s))}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error on pagination failure")
	}
}
