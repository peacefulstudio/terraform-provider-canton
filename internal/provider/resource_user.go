// Copyright (c) 2026 Peaceful Studio OÜ
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/noders-team/go-daml/pkg/model"
	"github.com/noders-team/go-daml/pkg/service/admin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

type userResource struct {
	userMng admin.UserManagement
}

type userResourceModel struct {
	UserID       types.String `tfsdk:"user_id"`
	PrimaryParty types.String `tfsdk:"primary_party"`
}

func NewUserResource() resource.Resource {
	return &userResource{}
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a user on a Canton participant node.",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.StringAttribute{
				Description: "The user identifier.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"primary_party": schema.StringAttribute{
				Description: "The primary party for the user.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*cantonClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *cantonClients, got: %T", req.ProviderData),
		)
		return
	}

	if clients.binding.UserMng == nil {
		resp.Diagnostics.AddError(
			"User management service unavailable",
			"The Canton client was created but the user management service is nil. This may indicate a client initialization problem.",
		)
		return
	}

	r.userMng = clients.binding.UserMng
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := plan.UserID.ValueString()
	primaryParty := plan.PrimaryParty.ValueString()

	tflog.Debug(ctx, "Creating user", map[string]interface{}{
		"user_id":       userID,
		"primary_party": primaryParty,
	})

	user := &model.User{
		ID:           userID,
		PrimaryParty: primaryParty,
	}

	_, err := r.userMng.CreateUser(ctx, user, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating user",
			"Could not create user "+userID+": "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "User created", map[string]interface{}{"user_id": userID})

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := state.UserID.ValueString()

	user, err := r.userMng.GetUser(ctx, userID)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			tflog.Warn(ctx, "User not found on participant, removing from state", map[string]interface{}{
				"user_id": userID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading user",
			"Could not read user "+userID+": "+err.Error(),
		)
		return
	}

	if user == nil {
		tflog.Warn(ctx, "User not found on participant, removing from state", map[string]interface{}{
			"user_id": userID,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	state.UserID = types.StringValue(user.ID)
	state.PrimaryParty = types.StringValue(user.PrimaryParty)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update should never be called because all attributes use RequiresReplace, but return an error defensively.
func (r *userResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "User resources cannot be updated in-place.")
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := state.UserID.ValueString()

	tflog.Debug(ctx, "Deleting user", map[string]interface{}{"user_id": userID})

	err := r.userMng.DeleteUser(ctx, userID)
	if err != nil {
		// If the user was already deleted out-of-band, treat as successful.
		if status.Code(err) == codes.NotFound {
			tflog.Warn(ctx, "User already deleted, treating as successful", map[string]interface{}{
				"user_id": userID,
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting user",
			"Could not delete user "+userID+": "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "User deleted", map[string]interface{}{"user_id": userID})
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("user_id"), req, resp)
}
