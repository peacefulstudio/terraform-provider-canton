// Copyright (c) 2026 Peaceful Studio OÜ
// SPDX-License-Identifier: Apache-2.0

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

func TestUserResource_Metadata(t *testing.T) {
	r := NewUserResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "canton"}, resp)

	if resp.TypeName != "canton_user" {
		t.Errorf("expected type name 'canton_user', got %q", resp.TypeName)
	}
}

func TestUserResource_Schema(t *testing.T) {
	r := NewUserResource()
	s := getResourceSchema(r)

	for _, name := range []string{"user_id", "primary_party"} {
		attr, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing attribute: %s", name)
			continue
		}
		if !attr.IsRequired() {
			t.Errorf("attribute %s should be required", name)
		}
	}
}

func TestUserResource_Configure_NilProviderData(t *testing.T) {
	r := &userResource{}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for nil provider data, got: %s", resp.Diagnostics.Errors())
	}
}

func TestUserResource_Configure_WrongType(t *testing.T) {
	r := &userResource{}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: 42}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for wrong provider data type")
	}
}

func TestUserResource_Configure_NilUserMng(t *testing.T) {
	r := &userResource{}
	resp := &resource.ConfigureResponse{}
	binding := &damlclient.DamlBindingClient{}
	clients := &cantonClients{binding: binding}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: clients}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for nil user management service")
	}
}

func TestUserResource_Configure_Success(t *testing.T) {
	mock := &mockUserManagement{}
	r := &userResource{}
	resp := &resource.ConfigureResponse{}
	binding := &damlclient.DamlBindingClient{UserMng: mock}
	clients := &cantonClients{binding: binding}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: clients}, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %s", resp.Diagnostics.Errors())
	}
	if r.userMng != mock {
		t.Error("expected userMng to be set")
	}
}

func TestUserResource_Create_Success(t *testing.T) {
	mock := &mockUserManagement{
		CreateUserFn: func(_ context.Context, user *model.User, _ []*model.Right) (*model.User, error) {
			return user, nil
		},
	}
	r := &userResource{userMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	plan := newTestPlan(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"user_id":       tftypes.NewValue(tftypes.String, "alice"),
		"primary_party": tftypes.NewValue(tftypes.String, "party::alice::1"),
	}))

	resp := &resource.CreateResponse{State: emptyState(s)}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}

	state := getState[userResourceModel](t, resp.State)

	if state.UserID.ValueString() != "alice" {
		t.Errorf("expected user_id 'alice', got %q", state.UserID.ValueString())
	}
	if state.PrimaryParty.ValueString() != "party::alice::1" {
		t.Errorf("expected primary_party 'party::alice::1', got %q", state.PrimaryParty.ValueString())
	}
}

func TestUserResource_Create_Error(t *testing.T) {
	mock := &mockUserManagement{
		CreateUserFn: func(_ context.Context, _ *model.User, _ []*model.Right) (*model.User, error) {
			return nil, fmt.Errorf("user already exists")
		},
	}
	r := &userResource{userMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	plan := newTestPlan(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"user_id":       tftypes.NewValue(tftypes.String, "alice"),
		"primary_party": tftypes.NewValue(tftypes.String, "party::alice::1"),
	}))

	resp := &resource.CreateResponse{State: emptyState(s)}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error from failed create")
	}
}

func TestUserResource_Read_Success(t *testing.T) {
	mock := &mockUserManagement{
		GetUserFn: func(_ context.Context, userID string) (*model.User, error) {
			return &model.User{ID: userID, PrimaryParty: "party::alice::1"}, nil
		},
	}
	r := &userResource{userMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	state := newTestState(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"user_id":       tftypes.NewValue(tftypes.String, "alice"),
		"primary_party": tftypes.NewValue(tftypes.String, "party::alice::1"),
	}))

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}

	result := getState[userResourceModel](t, resp.State)
	if result.UserID.ValueString() != "alice" {
		t.Errorf("expected user_id 'alice', got %q", result.UserID.ValueString())
	}
	if result.PrimaryParty.ValueString() != "party::alice::1" {
		t.Errorf("expected primary_party 'party::alice::1', got %q", result.PrimaryParty.ValueString())
	}
}

func TestUserResource_Read_NotFound_GrpcStatus(t *testing.T) {
	mock := &mockUserManagement{
		GetUserFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, status.Error(codes.NotFound, "user not found")
		},
	}
	r := &userResource{userMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	state := newTestState(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"user_id":       tftypes.NewValue(tftypes.String, "alice"),
		"primary_party": tftypes.NewValue(tftypes.String, "party::alice::1"),
	}))

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed on NotFound")
	}
}

func TestUserResource_Read_NilUser(t *testing.T) {
	mock := &mockUserManagement{
		GetUserFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, nil
		},
	}
	r := &userResource{userMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	state := newTestState(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"user_id":       tftypes.NewValue(tftypes.String, "alice"),
		"primary_party": tftypes.NewValue(tftypes.String, "party::alice::1"),
	}))

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed when user is nil")
	}
}

func TestUserResource_Read_Error(t *testing.T) {
	mock := &mockUserManagement{
		GetUserFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	r := &userResource{userMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	state := newTestState(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"user_id":       tftypes.NewValue(tftypes.String, "alice"),
		"primary_party": tftypes.NewValue(tftypes.String, "party::alice::1"),
	}))

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error from failed read")
	}
}

func TestUserResource_Update_ReturnsError(t *testing.T) {
	r := &userResource{}
	resp := &resource.UpdateResponse{}
	r.Update(context.Background(), resource.UpdateRequest{}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error from Update (user resources cannot be updated)")
	}
}

func TestUserResource_Delete_Success(t *testing.T) {
	deleted := false
	mock := &mockUserManagement{
		DeleteUserFn: func(_ context.Context, userID string) error {
			if userID != "alice" {
				t.Errorf("expected delete for 'alice', got %q", userID)
			}
			deleted = true
			return nil
		},
	}
	r := &userResource{userMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	state := newTestState(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"user_id":       tftypes.NewValue(tftypes.String, "alice"),
		"primary_party": tftypes.NewValue(tftypes.String, "party::alice::1"),
	}))

	resp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}
	if !deleted {
		t.Error("expected DeleteUser to be called")
	}
}

func TestUserResource_Delete_NotFound(t *testing.T) {
	mock := &mockUserManagement{
		DeleteUserFn: func(_ context.Context, _ string) error {
			return status.Error(codes.NotFound, "user not found")
		},
	}
	r := &userResource{userMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	state := newTestState(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"user_id":       tftypes.NewValue(tftypes.String, "alice"),
		"primary_party": tftypes.NewValue(tftypes.String, "party::alice::1"),
	}))

	resp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Error("expected no error when user already deleted (NotFound is graceful)")
	}
}

func TestUserResource_Delete_Error(t *testing.T) {
	mock := &mockUserManagement{
		DeleteUserFn: func(_ context.Context, _ string) error {
			return fmt.Errorf("permission denied")
		},
	}
	r := &userResource{userMng: mock}
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	state := newTestState(s, tftypes.NewValue(objType, map[string]tftypes.Value{
		"user_id":       tftypes.NewValue(tftypes.String, "alice"),
		"primary_party": tftypes.NewValue(tftypes.String, "party::alice::1"),
	}))

	resp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error from failed delete")
	}
}

func TestUserResource_ImportState(t *testing.T) {
	r := &userResource{}
	s := getResourceSchema(r)

	resp := &resource.ImportStateResponse{State: emptyState(s)}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "alice"}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}

	state := getState[userResourceModel](t, resp.State)
	if state.UserID.ValueString() != "alice" {
		t.Errorf("expected user_id 'alice', got %q", state.UserID.ValueString())
	}
}
