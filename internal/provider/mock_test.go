// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"context"

	"github.com/noders-team/go-daml/pkg/model"
)

// mockPartyManagement implements admin.PartyManagement for testing.
// Only methods used in resource CRUD have configurable function fields;
// the rest return zero values.
type mockPartyManagement struct {
	AllocatePartyFn    func(ctx context.Context, partyIDHint string, localMetadata map[string]string, identityProviderID string) (*model.PartyDetails, error)
	GetPartiesFn       func(ctx context.Context, parties []string, identityProviderID string) ([]*model.PartyDetails, error)
	GetParticipantIDFn func(ctx context.Context) (string, error)
	ListKnownPartiesFn func(ctx context.Context, pageToken string, pageSize int32, identityProviderID string) (*model.ListKnownPartiesResponse, error)
}

func (m *mockPartyManagement) AllocateParty(ctx context.Context, partyIDHint string, localMetadata map[string]string, identityProviderID string) (*model.PartyDetails, error) {
	if m.AllocatePartyFn == nil {
		panic("mockPartyManagement.AllocatePartyFn not set but AllocateParty was called")
	}
	return m.AllocatePartyFn(ctx, partyIDHint, localMetadata, identityProviderID)
}

func (m *mockPartyManagement) GetParties(ctx context.Context, parties []string, identityProviderID string) ([]*model.PartyDetails, error) {
	if m.GetPartiesFn == nil {
		panic("mockPartyManagement.GetPartiesFn not set but GetParties was called")
	}
	return m.GetPartiesFn(ctx, parties, identityProviderID)
}

// GetParticipantID returns a default value when the function field is nil,
// unlike other mock methods which panic. This is intentional: GetParticipantID
// is not called by any resource under test, and the default avoids requiring
// every test to set it.
func (m *mockPartyManagement) GetParticipantID(ctx context.Context) (string, error) {
	if m.GetParticipantIDFn != nil {
		return m.GetParticipantIDFn(ctx)
	}
	return "participant-1", nil
}

func (m *mockPartyManagement) ListKnownParties(ctx context.Context, pageToken string, pageSize int32, identityProviderID string) (*model.ListKnownPartiesResponse, error) {
	if m.ListKnownPartiesFn == nil {
		panic("mockPartyManagement.ListKnownPartiesFn not set but ListKnownParties was called")
	}
	return m.ListKnownPartiesFn(ctx, pageToken, pageSize, identityProviderID)
}

func (m *mockPartyManagement) AllocateExternalParty(_ context.Context, _ string, _ []model.SignedTransaction, _ []model.Signature, _ string) (string, error) {
	return "", nil
}

func (m *mockPartyManagement) UpdatePartyDetails(_ context.Context, party *model.PartyDetails, _ *model.UpdateMask) (*model.PartyDetails, error) {
	return party, nil
}

func (m *mockPartyManagement) UpdatePartyIdentityProviderID(_ context.Context, _, _, _ string) error {
	return nil
}

// mockUserManagement implements admin.UserManagement for testing.
// Only methods used in resource CRUD have configurable function fields;
// the rest return zero values.
type mockUserManagement struct {
	CreateUserFn       func(ctx context.Context, user *model.User, rights []*model.Right) (*model.User, error)
	GetUserFn          func(ctx context.Context, userID string) (*model.User, error)
	DeleteUserFn       func(ctx context.Context, userID string) error
	GrantUserRightsFn  func(ctx context.Context, userID, identityProviderID string, rights []*model.Right) ([]*model.Right, error)
	RevokeUserRightsFn func(ctx context.Context, userID string, rights []*model.Right) ([]*model.Right, error)
	ListUserRightsFn   func(ctx context.Context, userID string) ([]*model.Right, error)
	ListUsersFn        func(ctx context.Context) ([]*model.User, error)
}

func (m *mockUserManagement) CreateUser(ctx context.Context, user *model.User, rights []*model.Right) (*model.User, error) {
	if m.CreateUserFn == nil {
		panic("mockUserManagement.CreateUserFn not set but CreateUser was called")
	}
	return m.CreateUserFn(ctx, user, rights)
}

func (m *mockUserManagement) GetUser(ctx context.Context, userID string) (*model.User, error) {
	if m.GetUserFn == nil {
		panic("mockUserManagement.GetUserFn not set but GetUser was called")
	}
	return m.GetUserFn(ctx, userID)
}

func (m *mockUserManagement) DeleteUser(ctx context.Context, userID string) error {
	if m.DeleteUserFn == nil {
		panic("mockUserManagement.DeleteUserFn not set but DeleteUser was called")
	}
	return m.DeleteUserFn(ctx, userID)
}

func (m *mockUserManagement) GrantUserRights(ctx context.Context, userID, identityProviderID string, rights []*model.Right) ([]*model.Right, error) {
	if m.GrantUserRightsFn == nil {
		panic("mockUserManagement.GrantUserRightsFn not set but GrantUserRights was called")
	}
	return m.GrantUserRightsFn(ctx, userID, identityProviderID, rights)
}

func (m *mockUserManagement) RevokeUserRights(ctx context.Context, userID string, rights []*model.Right) ([]*model.Right, error) {
	if m.RevokeUserRightsFn == nil {
		panic("mockUserManagement.RevokeUserRightsFn not set but RevokeUserRights was called")
	}
	return m.RevokeUserRightsFn(ctx, userID, rights)
}

func (m *mockUserManagement) ListUserRights(ctx context.Context, userID string) ([]*model.Right, error) {
	if m.ListUserRightsFn == nil {
		panic("mockUserManagement.ListUserRightsFn not set but ListUserRights was called")
	}
	return m.ListUserRightsFn(ctx, userID)
}

func (m *mockUserManagement) ListUsers(ctx context.Context) ([]*model.User, error) {
	if m.ListUsersFn == nil {
		panic("mockUserManagement.ListUsersFn not set but ListUsers was called")
	}
	return m.ListUsersFn(ctx)
}
