// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	damlclient "github.com/noders-team/go-daml/pkg/client"
	"github.com/noders-team/go-daml/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUserDataSource_Metadata(t *testing.T) {
	ds := NewUserDataSource()
	resp := &datasource.MetadataResponse{}
	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "canton"}, resp)

	if resp.TypeName != "canton_user" {
		t.Errorf("expected type name 'canton_user', got %q", resp.TypeName)
	}
}

func TestUserDataSource_Schema(t *testing.T) {
	ds := NewUserDataSource()
	s := getDatasourceSchema(ds)

	if s.Description == "" {
		t.Error("expected non-empty schema description")
	}

	if _, ok := s.Attributes["user_id"]; !ok {
		t.Error("expected user_id attribute")
	}
	if _, ok := s.Attributes["primary_party"]; !ok {
		t.Error("expected primary_party attribute")
	}
}

func TestUserDataSource_Configure_NilProviderData(t *testing.T) {
	ds := &userDataSource{}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Error("expected no errors for nil provider data")
	}
}

func TestUserDataSource_Configure_WrongType(t *testing.T) {
	ds := &userDataSource{}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "wrong"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for wrong provider data type")
	}
}

func TestUserDataSource_Configure_NilService(t *testing.T) {
	ds := &userDataSource{}
	resp := &datasource.ConfigureResponse{}
	clients := &cantonClients{binding: &damlclient.DamlBindingClient{}}
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: clients}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for nil user management service")
	}
}

func TestUserDataSource_Configure_Success(t *testing.T) {
	ds := &userDataSource{}
	resp := &datasource.ConfigureResponse{}
	mock := &mockUserManagement{}
	clients := &cantonClients{binding: &damlclient.DamlBindingClient{UserMng: mock}}
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: clients}, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected errors: %s", resp.Diagnostics.Errors())
	}
	if ds.userMng != mock {
		t.Error("expected userMng to be set")
	}
}

func TestUserDataSource_Read_Success(t *testing.T) {
	mock := &mockUserManagement{
		GetUserFn: func(_ context.Context, userID string) (*model.User, error) {
			if userID != "treasury-service" {
				t.Fatalf("unexpected user ID: %s", userID)
			}
			return &model.User{ID: "treasury-service", PrimaryParty: "party::treasury"}, nil
		},
	}
	ds := &userDataSource{userMng: mock}
	s := getDatasourceSchema(ds)
	objType := dsSchemaToTftype(s)

	config := newTestDSConfig(s, tftypesObjectValue(objType, map[string]interface{}{
		"user_id":       "treasury-service",
		"primary_party": nil,
	}))

	resp := &datasource.ReadResponse{State: emptyDSState(s)}
	ds.Read(context.Background(), datasource.ReadRequest{Config: config}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", resp.Diagnostics.Errors())
	}

	state := getState[userDataSourceModel](t, resp.State)
	if state.UserID.ValueString() != "treasury-service" {
		t.Errorf("expected user_id 'treasury-service', got %q", state.UserID.ValueString())
	}
	if state.PrimaryParty.ValueString() != "party::treasury" {
		t.Errorf("expected primary_party 'party::treasury', got %q", state.PrimaryParty.ValueString())
	}
}

func TestUserDataSource_Read_NotFound_GrpcStatus(t *testing.T) {
	mock := &mockUserManagement{
		GetUserFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, status.Error(codes.NotFound, "user not found")
		},
	}
	ds := &userDataSource{userMng: mock}
	s := getDatasourceSchema(ds)
	objType := dsSchemaToTftype(s)

	config := newTestDSConfig(s, tftypesObjectValue(objType, map[string]interface{}{
		"user_id":       "nonexistent",
		"primary_party": nil,
	}))

	resp := &datasource.ReadResponse{State: emptyDSState(s)}
	ds.Read(context.Background(), datasource.ReadRequest{Config: config}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for not-found user")
	}
}

func TestUserDataSource_Read_NilUser(t *testing.T) {
	mock := &mockUserManagement{
		GetUserFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, nil
		},
	}
	ds := &userDataSource{userMng: mock}
	s := getDatasourceSchema(ds)
	objType := dsSchemaToTftype(s)

	config := newTestDSConfig(s, tftypesObjectValue(objType, map[string]interface{}{
		"user_id":       "ghost",
		"primary_party": nil,
	}))

	resp := &datasource.ReadResponse{State: emptyDSState(s)}
	ds.Read(context.Background(), datasource.ReadRequest{Config: config}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for nil user response")
	}
}

func TestUserDataSource_Read_Error(t *testing.T) {
	mock := &mockUserManagement{
		GetUserFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	ds := &userDataSource{userMng: mock}
	s := getDatasourceSchema(ds)
	objType := dsSchemaToTftype(s)

	config := newTestDSConfig(s, tftypesObjectValue(objType, map[string]interface{}{
		"user_id":       "treasury-service",
		"primary_party": nil,
	}))

	resp := &datasource.ReadResponse{State: emptyDSState(s)}
	ds.Read(context.Background(), datasource.ReadRequest{Config: config}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error on API failure")
	}
}
