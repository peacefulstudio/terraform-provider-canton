// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	damlclient "github.com/noders-team/go-daml/pkg/client"
	"github.com/noders-team/go-daml/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUserRightsResource_Metadata(t *testing.T) {
	r := NewUserRightsResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "canton"}, resp)

	if resp.TypeName != "canton_user_rights" {
		t.Errorf("expected type name 'canton_user_rights', got %q", resp.TypeName)
	}
}

func TestUserRightsResource_Schema(t *testing.T) {
	r := NewUserRightsResource()
	s := getResourceSchema(r)

	if _, ok := s.Attributes["user_id"]; !ok {
		t.Error("missing user_id attribute")
	}
	if _, ok := s.Attributes["act_as"]; !ok {
		t.Error("missing act_as attribute")
	}
	if _, ok := s.Attributes["read_as"]; !ok {
		t.Error("missing read_as attribute")
	}
	if _, ok := s.Attributes["participant_admin"]; !ok {
		t.Error("missing participant_admin attribute")
	}
	if _, ok := s.Attributes["identity_provider_admin"]; !ok {
		t.Error("missing identity_provider_admin attribute")
	}
}

func TestUserRightsResource_Configure_NilProviderData(t *testing.T) {
	r := &userRightsResource{}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error, got: %s", resp.Diagnostics.Errors())
	}
}

func TestUserRightsResource_Configure_WrongType(t *testing.T) {
	r := &userRightsResource{}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "bad"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for wrong provider data type")
	}
}

func TestUserRightsResource_Configure_NilUserMng(t *testing.T) {
	r := &userRightsResource{}
	resp := &resource.ConfigureResponse{}
	binding := &damlclient.DamlBindingClient{}
	clients := &cantonClients{binding: binding}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: clients}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for nil user management service")
	}
}

func TestUserRightsResource_Configure_Success(t *testing.T) {
	mock := &mockUserManagement{}
	r := &userRightsResource{}
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

// userRightsValue builds a tftypes.Value for the user_rights schema.
func userRightsValue(userID string, actAs, readAs []string, participantAdmin, idpAdmin bool) tftypes.Value {
	r := NewUserRightsResource()
	s := getResourceSchema(r)
	objType := schemaToTftype(s)

	actAsVals := make([]tftypes.Value, len(actAs))
	for i, p := range actAs {
		actAsVals[i] = tftypes.NewValue(tftypes.String, p)
	}
	readAsVals := make([]tftypes.Value, len(readAs))
	for i, p := range readAs {
		readAsVals[i] = tftypes.NewValue(tftypes.String, p)
	}

	return tftypes.NewValue(objType, map[string]tftypes.Value{
		"user_id":                 tftypes.NewValue(tftypes.String, userID),
		"act_as":                  tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, actAsVals),
		"read_as":                 tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, readAsVals),
		"participant_admin":       tftypes.NewValue(tftypes.Bool, participantAdmin),
		"identity_provider_admin": tftypes.NewValue(tftypes.Bool, idpAdmin),
	})
}

func TestUserRightsResource_Create_WithRights(t *testing.T) {
	var grantedRights []*model.Right
	mock := &mockUserManagement{
		GrantUserRightsFn: func(_ context.Context, _ string, _ string, rights []*model.Right) ([]*model.Right, error) {
			grantedRights = rights
			return rights, nil
		},
		ListUserRightsFn: func(_ context.Context, _ string) ([]*model.Right, error) {
			return []*model.Right{
				{Type: model.CanActAs{Party: "party::a"}},
				{Type: model.CanReadAs{Party: "party::b"}},
				{Type: model.ParticipantAdmin{}},
			}, nil
		},
	}

	r := &userRightsResource{userMng: mock}
	s := getResourceSchema(r)

	val := userRightsValue("alice", []string{"party::a"}, []string{"party::b"}, true, false)
	plan := newTestPlan(s, val)

	resp := &resource.CreateResponse{State: emptyState(s)}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}

	if len(grantedRights) != 3 {
		t.Errorf("expected 3 rights to be granted, got %d", len(grantedRights))
	}

	state := getState[userRightsResourceModel](t, resp.State)
	if state.UserID.ValueString() != "alice" {
		t.Errorf("expected user_id 'alice', got %q", state.UserID.ValueString())
	}
	if !state.ParticipantAdmin.ValueBool() {
		t.Error("expected participant_admin to be true in state")
	}

	var actAs []string
	if d := state.ActAs.ElementsAs(context.Background(), &actAs, false); d.HasError() {
		t.Fatalf("unexpected act_as decode error: %s", d.Errors())
	}
	if len(actAs) != 1 || actAs[0] != "party::a" {
		t.Errorf("expected act_as [party::a], got %v", actAs)
	}
}

func TestUserRightsResource_Create_NoRights(t *testing.T) {
	grantCalled := false
	mock := &mockUserManagement{
		GrantUserRightsFn: func(_ context.Context, _ string, _ string, _ []*model.Right) ([]*model.Right, error) {
			grantCalled = true
			return nil, nil
		},
		ListUserRightsFn: func(_ context.Context, _ string) ([]*model.Right, error) {
			return []*model.Right{}, nil
		},
	}

	r := &userRightsResource{userMng: mock}
	s := getResourceSchema(r)

	val := userRightsValue("alice", []string{}, []string{}, false, false)
	plan := newTestPlan(s, val)

	resp := &resource.CreateResponse{State: emptyState(s)}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}
	if grantCalled {
		t.Error("GrantUserRights should not be called when there are no rights")
	}
}

func TestUserRightsResource_Create_GrantError(t *testing.T) {
	mock := &mockUserManagement{
		GrantUserRightsFn: func(_ context.Context, _ string, _ string, _ []*model.Right) ([]*model.Right, error) {
			return nil, fmt.Errorf("grant failed")
		},
	}

	r := &userRightsResource{userMng: mock}
	s := getResourceSchema(r)

	val := userRightsValue("alice", []string{"party::a"}, nil, false, false)
	plan := newTestPlan(s, val)

	resp := &resource.CreateResponse{State: emptyState(s)}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error from failed grant")
	}
}

func TestUserRightsResource_Read_Success(t *testing.T) {
	mock := &mockUserManagement{
		ListUserRightsFn: func(_ context.Context, _ string) ([]*model.Right, error) {
			return []*model.Right{
				{Type: model.CanActAs{Party: "party::a"}},
				{Type: model.ParticipantAdmin{}},
			}, nil
		},
	}

	r := &userRightsResource{userMng: mock}
	s := getResourceSchema(r)

	val := userRightsValue("alice", []string{"party::a"}, []string{}, true, false)
	state := newTestState(s, val)

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}

	result := getState[userRightsResourceModel](t, resp.State)
	if !result.ParticipantAdmin.ValueBool() {
		t.Error("expected participant_admin to be true")
	}
}

func TestUserRightsResource_Read_NotFound(t *testing.T) {
	mock := &mockUserManagement{
		ListUserRightsFn: func(_ context.Context, _ string) ([]*model.Right, error) {
			return nil, status.Error(codes.NotFound, "user not found")
		},
	}

	r := &userRightsResource{userMng: mock}
	s := getResourceSchema(r)

	val := userRightsValue("alice", []string{}, []string{}, false, false)
	state := newTestState(s, val)

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed on NotFound")
	}
}

func TestUserRightsResource_Read_Error(t *testing.T) {
	mock := &mockUserManagement{
		ListUserRightsFn: func(_ context.Context, _ string) ([]*model.Right, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	r := &userRightsResource{userMng: mock}
	s := getResourceSchema(r)

	val := userRightsValue("alice", []string{}, []string{}, false, false)
	state := newTestState(s, val)

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error from failed read")
	}
}

func TestUserRightsResource_Update_GrantAndRevoke(t *testing.T) {
	var granted, revoked []*model.Right
	mock := &mockUserManagement{
		GrantUserRightsFn: func(_ context.Context, _ string, _ string, rights []*model.Right) ([]*model.Right, error) {
			granted = rights
			return rights, nil
		},
		RevokeUserRightsFn: func(_ context.Context, _ string, rights []*model.Right) ([]*model.Right, error) {
			revoked = rights
			return rights, nil
		},
		ListUserRightsFn: func(_ context.Context, _ string) ([]*model.Right, error) {
			return []*model.Right{
				{Type: model.CanActAs{Party: "party::b"}},
			}, nil
		},
	}

	r := &userRightsResource{userMng: mock}
	s := getResourceSchema(r)

	// Current state: act_as = [party::a]
	stateVal := userRightsValue("alice", []string{"party::a"}, []string{}, false, false)
	// Desired plan: act_as = [party::b]
	planVal := userRightsValue("alice", []string{"party::b"}, []string{}, false, false)

	resp := &resource.UpdateResponse{State: emptyState(s)}
	r.Update(context.Background(), resource.UpdateRequest{
		Plan:  newTestPlan(s, planVal),
		State: newTestState(s, stateVal),
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}

	if len(granted) != 1 {
		t.Errorf("expected 1 right to grant, got %d", len(granted))
	}
	if len(revoked) != 1 {
		t.Errorf("expected 1 right to revoke, got %d", len(revoked))
	}

	state := getState[userRightsResourceModel](t, resp.State)
	if state.UserID.ValueString() != "alice" {
		t.Errorf("expected user_id 'alice', got %q", state.UserID.ValueString())
	}
	var actAs []string
	if d := state.ActAs.ElementsAs(context.Background(), &actAs, false); d.HasError() {
		t.Fatalf("unexpected act_as decode error: %s", d.Errors())
	}
	if len(actAs) != 1 || actAs[0] != "party::b" {
		t.Errorf("expected act_as [party::b], got %v", actAs)
	}
}

func TestUserRightsResource_Update_GrantError_RefreshesState(t *testing.T) {
	listCalled := false
	mock := &mockUserManagement{
		GrantUserRightsFn: func(_ context.Context, _ string, _ string, _ []*model.Right) ([]*model.Right, error) {
			return nil, fmt.Errorf("grant failed")
		},
		ListUserRightsFn: func(_ context.Context, _ string) ([]*model.Right, error) {
			listCalled = true
			return []*model.Right{}, nil
		},
	}

	r := &userRightsResource{userMng: mock}
	s := getResourceSchema(r)

	stateVal := userRightsValue("alice", []string{}, []string{}, false, false)
	planVal := userRightsValue("alice", []string{"party::new"}, []string{}, false, false)

	resp := &resource.UpdateResponse{State: emptyState(s)}
	r.Update(context.Background(), resource.UpdateRequest{
		Plan:  newTestPlan(s, planVal),
		State: newTestState(s, stateVal),
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error from failed grant")
	}
	if !listCalled {
		t.Error("expected ListUserRights to be called to refresh state after failure")
	}
}

func TestUserRightsResource_Update_RevokeError_RefreshesState(t *testing.T) {
	listCallCount := 0
	mock := &mockUserManagement{
		GrantUserRightsFn: func(_ context.Context, _ string, _ string, rights []*model.Right) ([]*model.Right, error) {
			return rights, nil
		},
		RevokeUserRightsFn: func(_ context.Context, _ string, _ []*model.Right) ([]*model.Right, error) {
			return nil, fmt.Errorf("revoke failed")
		},
		ListUserRightsFn: func(_ context.Context, _ string) ([]*model.Right, error) {
			listCallCount++
			return []*model.Right{
				{Type: model.CanActAs{Party: "party::a"}},
				{Type: model.CanActAs{Party: "party::b"}},
			}, nil
		},
	}

	r := &userRightsResource{userMng: mock}
	s := getResourceSchema(r)

	stateVal := userRightsValue("alice", []string{"party::a"}, []string{}, false, false)
	planVal := userRightsValue("alice", []string{"party::b"}, []string{}, false, false)

	resp := &resource.UpdateResponse{State: emptyState(s)}
	r.Update(context.Background(), resource.UpdateRequest{
		Plan:  newTestPlan(s, planVal),
		State: newTestState(s, stateVal),
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error from failed revoke")
	}
	if listCallCount == 0 {
		t.Error("expected state refresh after revoke failure")
	}
}

func TestUserRightsResource_Update_NoChanges(t *testing.T) {
	grantCalled := false
	revokeCalled := false
	mock := &mockUserManagement{
		GrantUserRightsFn: func(_ context.Context, _ string, _ string, _ []*model.Right) ([]*model.Right, error) {
			grantCalled = true
			return nil, nil
		},
		RevokeUserRightsFn: func(_ context.Context, _ string, _ []*model.Right) ([]*model.Right, error) {
			revokeCalled = true
			return nil, nil
		},
		ListUserRightsFn: func(_ context.Context, _ string) ([]*model.Right, error) {
			return []*model.Right{
				{Type: model.CanActAs{Party: "party::a"}},
			}, nil
		},
	}

	r := &userRightsResource{userMng: mock}
	s := getResourceSchema(r)

	val := userRightsValue("alice", []string{"party::a"}, []string{}, false, false)

	resp := &resource.UpdateResponse{State: emptyState(s)}
	r.Update(context.Background(), resource.UpdateRequest{
		Plan:  newTestPlan(s, val),
		State: newTestState(s, val),
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}
	if grantCalled {
		t.Error("GrantUserRights should not be called when no changes")
	}
	if revokeCalled {
		t.Error("RevokeUserRights should not be called when no changes")
	}
}

func TestUserRightsResource_Delete_WithRights(t *testing.T) {
	var revokedRights []*model.Right
	mock := &mockUserManagement{
		RevokeUserRightsFn: func(_ context.Context, _ string, rights []*model.Right) ([]*model.Right, error) {
			revokedRights = rights
			return rights, nil
		},
	}

	r := &userRightsResource{userMng: mock}
	s := getResourceSchema(r)

	val := userRightsValue("alice", []string{"party::a"}, []string{"party::b"}, true, true)
	state := newTestState(s, val)

	resp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}
	if len(revokedRights) != 4 {
		t.Errorf("expected 4 rights to revoke, got %d", len(revokedRights))
	}
}

func TestUserRightsResource_Delete_NoRights(t *testing.T) {
	revokeCalled := false
	mock := &mockUserManagement{
		RevokeUserRightsFn: func(_ context.Context, _ string, _ []*model.Right) ([]*model.Right, error) {
			revokeCalled = true
			return nil, nil
		},
	}

	r := &userRightsResource{userMng: mock}
	s := getResourceSchema(r)

	val := userRightsValue("alice", []string{}, []string{}, false, false)
	state := newTestState(s, val)

	resp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}
	if revokeCalled {
		t.Error("RevokeUserRights should not be called when there are no rights")
	}
}

func TestUserRightsResource_Delete_NotFound(t *testing.T) {
	mock := &mockUserManagement{
		RevokeUserRightsFn: func(_ context.Context, _ string, _ []*model.Right) ([]*model.Right, error) {
			return nil, status.Error(codes.NotFound, "user not found")
		},
	}

	r := &userRightsResource{userMng: mock}
	s := getResourceSchema(r)

	val := userRightsValue("alice", []string{"party::a"}, []string{}, false, false)
	state := newTestState(s, val)

	resp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Error("expected no error when user already deleted (NotFound is graceful)")
	}
}

func TestUserRightsResource_Delete_Error(t *testing.T) {
	mock := &mockUserManagement{
		RevokeUserRightsFn: func(_ context.Context, _ string, _ []*model.Right) ([]*model.Right, error) {
			return nil, fmt.Errorf("permission denied")
		},
	}

	r := &userRightsResource{userMng: mock}
	s := getResourceSchema(r)

	val := userRightsValue("alice", []string{"party::a"}, []string{}, false, false)
	state := newTestState(s, val)

	resp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error from failed revoke")
	}
}

func TestUserRightsResource_ImportState(t *testing.T) {
	r := &userRightsResource{}
	s := getResourceSchema(r)

	resp := &resource.ImportStateResponse{State: emptyState(s)}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "alice"}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}

	state := getState[userRightsResourceModel](t, resp.State)
	if state.UserID.ValueString() != "alice" {
		t.Errorf("expected user_id 'alice', got %q", state.UserID.ValueString())
	}
}

func TestUserRightsResource_Create_RefreshError(t *testing.T) {
	mock := &mockUserManagement{
		GrantUserRightsFn: func(_ context.Context, _ string, _ string, rights []*model.Right) ([]*model.Right, error) {
			return rights, nil
		},
		ListUserRightsFn: func(_ context.Context, _ string) ([]*model.Right, error) {
			return nil, fmt.Errorf("refresh failed")
		},
	}

	r := &userRightsResource{userMng: mock}
	s := getResourceSchema(r)

	val := userRightsValue("alice", []string{"party::a"}, []string{}, false, false)
	plan := newTestPlan(s, val)

	resp := &resource.CreateResponse{State: emptyState(s)}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error when writeRefreshedState fails after successful grant")
	}
}

// --- Helper function tests ---

func TestDiffRights(t *testing.T) {
	tests := []struct {
		name    string
		a       []*model.Right
		b       []*model.Right
		wantLen int
	}{
		{
			name:    "empty both",
			a:       nil,
			b:       nil,
			wantLen: 0,
		},
		{
			name: "a has rights, b empty",
			a: []*model.Right{
				{Type: model.CanActAs{Party: "p1"}},
				{Type: model.CanReadAs{Party: "p2"}},
			},
			b:       nil,
			wantLen: 2,
		},
		{
			name: "identical sets",
			a: []*model.Right{
				{Type: model.CanActAs{Party: "p1"}},
			},
			b: []*model.Right{
				{Type: model.CanActAs{Party: "p1"}},
			},
			wantLen: 0,
		},
		{
			name: "partial overlap",
			a: []*model.Right{
				{Type: model.CanActAs{Party: "p1"}},
				{Type: model.CanActAs{Party: "p2"}},
				{Type: model.ParticipantAdmin{}},
			},
			b: []*model.Right{
				{Type: model.CanActAs{Party: "p1"}},
			},
			wantLen: 2,
		},
		{
			name: "different right types same party",
			a: []*model.Right{
				{Type: model.CanActAs{Party: "p1"}},
			},
			b: []*model.Right{
				{Type: model.CanReadAs{Party: "p1"}},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diffRights(tt.a, tt.b)
			if len(got) != tt.wantLen {
				t.Errorf("diffRights() returned %d rights, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestRightKey(t *testing.T) {
	tests := []struct {
		right *model.Right
		want  string
	}{
		{&model.Right{Type: model.CanActAs{Party: "p1"}}, "act_as:p1"},
		{&model.Right{Type: model.CanReadAs{Party: "p2"}}, "read_as:p2"},
		{&model.Right{Type: model.ParticipantAdmin{}}, "participant_admin"},
		{&model.Right{Type: model.IdentityProviderAdmin{}}, "identity_provider_admin"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := rightKey(tt.right)
			if got != tt.want {
				t.Errorf("rightKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestModelToRights(t *testing.T) {
	ctx := context.Background()

	actAsSet, _ := types.SetValueFrom(ctx, types.StringType, []string{"party::a", "party::b"})
	readAsSet, _ := types.SetValueFrom(ctx, types.StringType, []string{"party::c"})

	m := userRightsResourceModel{
		UserID:                types.StringValue("alice"),
		ActAs:                 actAsSet,
		ReadAs:                readAsSet,
		ParticipantAdmin:      types.BoolValue(true),
		IdentityProviderAdmin: types.BoolValue(false),
	}

	var diags diag.Diagnostics
	rights := modelToRights(ctx, m, &diags)

	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags.Errors())
	}

	// 2 act_as + 1 read_as + 1 participant_admin = 4
	if len(rights) != 4 {
		t.Errorf("expected 4 rights, got %d", len(rights))
	}

	keys := map[string]bool{}
	for _, r := range rights {
		keys[rightKey(r)] = true
	}

	expected := []string{"act_as:party::a", "act_as:party::b", "read_as:party::c", "participant_admin"}
	for _, k := range expected {
		if !keys[k] {
			t.Errorf("expected right %q not found", k)
		}
	}
}

func TestModelToRights_NullSets(t *testing.T) {
	m := userRightsResourceModel{
		UserID:                types.StringValue("alice"),
		ActAs:                 types.SetNull(types.StringType),
		ReadAs:                types.SetNull(types.StringType),
		ParticipantAdmin:      types.BoolValue(false),
		IdentityProviderAdmin: types.BoolValue(false),
	}

	var diags diag.Diagnostics
	rights := modelToRights(context.Background(), m, &diags)

	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags.Errors())
	}
	if len(rights) != 0 {
		t.Errorf("expected 0 rights for null sets, got %d", len(rights))
	}
}

func TestModelToRights_AllAdminTypes(t *testing.T) {
	emptySet, _ := types.SetValueFrom(context.Background(), types.StringType, []string{})
	m := userRightsResourceModel{
		UserID:                types.StringValue("alice"),
		ActAs:                 emptySet,
		ReadAs:                emptySet,
		ParticipantAdmin:      types.BoolValue(true),
		IdentityProviderAdmin: types.BoolValue(true),
	}

	var diags diag.Diagnostics
	rights := modelToRights(context.Background(), m, &diags)

	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags.Errors())
	}
	if len(rights) != 2 {
		t.Errorf("expected 2 admin rights, got %d", len(rights))
	}
}

func TestRightsToModel(t *testing.T) {
	ctx := context.Background()
	rights := []*model.Right{
		{Type: model.CanActAs{Party: "party::a"}},
		{Type: model.CanActAs{Party: "party::b"}},
		{Type: model.CanReadAs{Party: "party::c"}},
		{Type: model.ParticipantAdmin{}},
		{Type: model.IdentityProviderAdmin{}},
	}

	var m userRightsResourceModel
	var diags diag.Diagnostics
	rightsToModel(ctx, rights, &m, &diags)

	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags.Errors())
	}

	var actAs []string
	diags.Append(m.ActAs.ElementsAs(ctx, &actAs, false)...)
	if len(actAs) != 2 {
		t.Errorf("expected 2 act_as parties, got %d", len(actAs))
	}

	var readAs []string
	diags.Append(m.ReadAs.ElementsAs(ctx, &readAs, false)...)
	if len(readAs) != 1 {
		t.Errorf("expected 1 read_as party, got %d", len(readAs))
	}

	if !m.ParticipantAdmin.ValueBool() {
		t.Error("expected participant_admin to be true")
	}
	if !m.IdentityProviderAdmin.ValueBool() {
		t.Error("expected identity_provider_admin to be true")
	}
}

func TestRightsToModel_Empty(t *testing.T) {
	var m userRightsResourceModel
	var diags diag.Diagnostics
	rightsToModel(context.Background(), []*model.Right{}, &m, &diags)

	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags.Errors())
	}

	if m.ParticipantAdmin.ValueBool() {
		t.Error("expected participant_admin to be false")
	}
	if m.IdentityProviderAdmin.ValueBool() {
		t.Error("expected identity_provider_admin to be false")
	}
	// Sets should be empty, not null.
	if m.ActAs.IsNull() {
		t.Error("expected act_as to be empty set, not null")
	}
	if m.ReadAs.IsNull() {
		t.Error("expected read_as to be empty set, not null")
	}
}
