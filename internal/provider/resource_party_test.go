// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	damlclient "github.com/noders-team/go-daml/pkg/client"
	"github.com/noders-team/go-daml/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPartyResource_Metadata(t *testing.T) {
	r := NewPartyResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "canton"}, resp)

	if resp.TypeName != "canton_party" {
		t.Errorf("expected type name 'canton_party', got %q", resp.TypeName)
	}
}

func TestPartyResource_Schema(t *testing.T) {
	r := NewPartyResource()
	s := getResourceSchema(r)

	requiredAttrs := []string{"party_id_hint"}
	computedAttrs := []string{"party_id", "is_local"}

	for _, name := range requiredAttrs {
		attr, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing attribute: %s", name)
			continue
		}
		if !attr.IsRequired() {
			t.Errorf("attribute %s should be required", name)
		}
	}
	for _, name := range computedAttrs {
		attr, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing attribute: %s", name)
			continue
		}
		if !attr.IsComputed() {
			t.Errorf("attribute %s should be computed", name)
		}
	}
}

func TestPartyResource_Configure_NilProviderData(t *testing.T) {
	r := &partyResource{}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for nil provider data, got: %s", resp.Diagnostics.Errors())
	}
}

func TestPartyResource_Configure_WrongType(t *testing.T) {
	r := &partyResource{}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "wrong"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for wrong provider data type")
	}
}

func TestPartyResource_Configure_NilPartyMng(t *testing.T) {
	r := &partyResource{}
	resp := &resource.ConfigureResponse{}
	binding := &damlclient.DamlBindingClient{} // PartyMng is nil
	clients := &cantonClients{binding: binding}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: clients}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for nil party management service")
	}
}

func TestPartyResource_Configure_Success(t *testing.T) {
	mock := &mockPartyManagement{}
	r := &partyResource{}
	resp := &resource.ConfigureResponse{}
	binding := &damlclient.DamlBindingClient{PartyMng: mock}
	clients := &cantonClients{binding: binding}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: clients}, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %s", resp.Diagnostics.Errors())
	}
	if r.partyMng != mock {
		t.Error("expected partyMng to be set")
	}
}

func TestPartyResource_Create_Success(t *testing.T) {
	mock := &mockPartyManagement{
		AllocatePartyFn: func(_ context.Context, hint string, _ map[string]string, _ string) (*model.PartyDetails, error) {
			return &model.PartyDetails{
				Party:   "party::" + hint + "::1234",
				IsLocal: true,
			}, nil
		},
	}
	r := &partyResource{partyMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	plan := newTestPlan(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"party_id_hint": tftypes.NewValue(tftypes.String, "validator"),
		"party_id":      tftypes.NewValue(tftypes.String, nil),
		"is_local":      tftypes.NewValue(tftypes.Bool, nil),
	}))

	resp := &resource.CreateResponse{State: emptyState(s)}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}

	state := getState[partyResourceModel](t, resp.State)

	if state.PartyID.ValueString() != "party::validator::1234" {
		t.Errorf("expected party_id 'party::validator::1234', got %q", state.PartyID.ValueString())
	}
	if !state.IsLocal.ValueBool() {
		t.Error("expected is_local to be true")
	}
}

func TestPartyResource_Create_NilParty(t *testing.T) {
	mock := &mockPartyManagement{
		AllocatePartyFn: func(_ context.Context, _ string, _ map[string]string, _ string) (*model.PartyDetails, error) {
			return nil, nil
		},
	}
	r := &partyResource{partyMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	plan := newTestPlan(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"party_id_hint": tftypes.NewValue(tftypes.String, "validator"),
		"party_id":      tftypes.NewValue(tftypes.String, nil),
		"is_local":      tftypes.NewValue(tftypes.Bool, nil),
	}))

	resp := &resource.CreateResponse{State: emptyState(s)}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error when AllocateParty returns nil party")
	}
}

func TestPartyResource_Create_Error(t *testing.T) {
	mock := &mockPartyManagement{
		AllocatePartyFn: func(_ context.Context, _ string, _ map[string]string, _ string) (*model.PartyDetails, error) {
			return nil, fmt.Errorf("allocation failed")
		},
	}
	r := &partyResource{partyMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	plan := newTestPlan(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"party_id_hint": tftypes.NewValue(tftypes.String, "validator"),
		"party_id":      tftypes.NewValue(tftypes.String, nil),
		"is_local":      tftypes.NewValue(tftypes.Bool, nil),
	}))

	resp := &resource.CreateResponse{State: emptyState(s)}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error from failed allocation")
	}
}

func TestPartyResource_Read_Success(t *testing.T) {
	mock := &mockPartyManagement{
		GetPartiesFn: func(_ context.Context, parties []string, _ string) ([]*model.PartyDetails, error) {
			return []*model.PartyDetails{
				{Party: parties[0], IsLocal: true},
			}, nil
		},
	}
	r := &partyResource{partyMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	state := newTestState(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"party_id_hint": tftypes.NewValue(tftypes.String, "validator"),
		"party_id":      tftypes.NewValue(tftypes.String, "party::validator::1234"),
		"is_local":      tftypes.NewValue(tftypes.Bool, true),
	}))

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}

	result := getState[partyResourceModel](t, resp.State)
	if result.PartyID.ValueString() != "party::validator::1234" {
		t.Errorf("expected party_id 'party::validator::1234', got %q", result.PartyID.ValueString())
	}
	if !result.IsLocal.ValueBool() {
		t.Error("expected is_local to be true")
	}
}

func TestPartyResource_Read_NotFound(t *testing.T) {
	mock := &mockPartyManagement{
		GetPartiesFn: func(_ context.Context, _ []string, _ string) ([]*model.PartyDetails, error) {
			return nil, status.Error(codes.NotFound, "party not found")
		},
	}
	r := &partyResource{partyMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	state := newTestState(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"party_id_hint": tftypes.NewValue(tftypes.String, "validator"),
		"party_id":      tftypes.NewValue(tftypes.String, "party::validator::1234"),
		"is_local":      tftypes.NewValue(tftypes.Bool, true),
	}))

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}

	// State should be removed.
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed on NotFound")
	}
}

func TestPartyResource_Read_EmptyResult(t *testing.T) {
	mock := &mockPartyManagement{
		GetPartiesFn: func(_ context.Context, _ []string, _ string) ([]*model.PartyDetails, error) {
			return []*model.PartyDetails{}, nil
		},
	}
	r := &partyResource{partyMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	state := newTestState(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"party_id_hint": tftypes.NewValue(tftypes.String, "validator"),
		"party_id":      tftypes.NewValue(tftypes.String, "party::validator::1234"),
		"is_local":      tftypes.NewValue(tftypes.Bool, true),
	}))

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error when party not returned (immutability check)")
	}
}

func TestPartyResource_Read_Error(t *testing.T) {
	mock := &mockPartyManagement{
		GetPartiesFn: func(_ context.Context, _ []string, _ string) ([]*model.PartyDetails, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	r := &partyResource{partyMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	state := newTestState(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"party_id_hint": tftypes.NewValue(tftypes.String, "validator"),
		"party_id":      tftypes.NewValue(tftypes.String, "party::validator::1234"),
		"is_local":      tftypes.NewValue(tftypes.Bool, true),
	}))

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error from failed read")
	}
}

func TestPartyResource_Update_ReturnsError(t *testing.T) {
	r := &partyResource{}
	resp := &resource.UpdateResponse{}
	r.Update(context.Background(), resource.UpdateRequest{}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error from Update (party resources cannot be updated)")
	}
}

func TestPartyResource_Delete_NoOp(t *testing.T) {
	r := &partyResource{}
	resp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error from Delete (no-op), got: %s", resp.Diagnostics.Errors())
	}
}

func TestPartyResource_ImportState(t *testing.T) {
	r := &partyResource{}
	s := getResourceSchema(r)

	resp := &resource.ImportStateResponse{State: emptyState(s)}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "party::myparty::5678"}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}

	state := getState[partyResourceModel](t, resp.State)

	if state.PartyID.ValueString() != "party::myparty::5678" {
		t.Errorf("expected party_id 'party::myparty::5678', got %q", state.PartyID.ValueString())
	}
	if state.PartyIDHint.ValueString() != "party::myparty::5678" {
		t.Errorf("expected party_id_hint 'party::myparty::5678', got %q", state.PartyIDHint.ValueString())
	}
}
