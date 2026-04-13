// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/noders-team/go-daml/pkg/model"
	"github.com/noders-team/go-daml/pkg/service/admin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	_ resource.Resource                = &userRightsResource{}
	_ resource.ResourceWithConfigure   = &userRightsResource{}
	_ resource.ResourceWithImportState = &userRightsResource{}
)

type userRightsResource struct {
	userMng admin.UserManagement
}

type userRightsResourceModel struct {
	UserID                types.String `tfsdk:"user_id"`
	ActAs                 types.Set    `tfsdk:"act_as"`
	ReadAs                types.Set    `tfsdk:"read_as"`
	ParticipantAdmin      types.Bool   `tfsdk:"participant_admin"`
	IdentityProviderAdmin types.Bool   `tfsdk:"identity_provider_admin"`
}

var emptyStringSet = func() types.Set {
	s, diags := types.SetValueFrom(context.Background(), types.StringType, []string{})
	if diags.HasError() {
		panic("failed to create empty string set for schema defaults: " + diags.Errors()[0].Detail())
	}
	return s
}()

func NewUserRightsResource() resource.Resource {
	return &userRightsResource{}
}

func (r *userRightsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_rights"
}

func (r *userRightsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages rights granted to a user on a Canton participant node.",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.StringAttribute{
				Description: "The target user ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"act_as": schema.SetAttribute{
				Description: "Set of party IDs the user can act as.",
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(emptyStringSet),
				ElementType: types.StringType,
			},
			"read_as": schema.SetAttribute{
				Description: "Set of party IDs the user can read as.",
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(emptyStringSet),
				ElementType: types.StringType,
			},
			"participant_admin": schema.BoolAttribute{
				Description: "Whether the user has participant admin rights.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"identity_provider_admin": schema.BoolAttribute{
				Description: "Whether the user has identity provider admin rights.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *userRightsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *userRightsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userRightsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := plan.UserID.ValueString()
	rights := modelToRights(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(rights) > 0 {
		tflog.Debug(ctx, "Granting user rights", map[string]interface{}{
			"user_id":     userID,
			"right_count": len(rights),
		})

		_, err := r.userMng.GrantUserRights(ctx, userID, "", rights)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error granting user rights",
				"Could not grant rights to user "+userID+": "+err.Error(),
			)
			return
		}
	}

	tflog.Info(ctx, "User rights granted", map[string]interface{}{"user_id": userID})

	// Refresh from server to populate computed attributes with known values.
	r.writeRefreshedState(ctx, userID, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *userRightsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userRightsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := state.UserID.ValueString()

	rights, err := r.userMng.ListUserRights(ctx, userID)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			tflog.Warn(ctx, "User not found on participant, removing user_rights from state", map[string]interface{}{
				"user_id": userID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading user rights",
			"Could not list rights for user "+userID+": "+err.Error(),
		)
		return
	}

	rightsToModel(ctx, rights, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *userRightsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state userRightsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := plan.UserID.ValueString()

	desiredRights := modelToRights(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	currentRights := modelToRights(ctx, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	toGrant := diffRights(desiredRights, currentRights)
	toRevoke := diffRights(currentRights, desiredRights)

	// Grant first (additive), then revoke (subtractive). If one step fails,
	// the user ends up with a superset of rights rather than a deficit.

	if len(toGrant) > 0 {
		tflog.Debug(ctx, "Granting user rights", map[string]interface{}{
			"user_id":     userID,
			"right_count": len(toGrant),
		})

		_, err := r.userMng.GrantUserRights(ctx, userID, "", toGrant)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error granting user rights",
				"Could not grant rights to user "+userID+": "+err.Error(),
			)
			r.refreshUpdateState(ctx, userID, resp)
			return
		}
	}

	if len(toRevoke) > 0 {
		tflog.Debug(ctx, "Revoking user rights", map[string]interface{}{
			"user_id":     userID,
			"right_count": len(toRevoke),
		})

		_, err := r.userMng.RevokeUserRights(ctx, userID, toRevoke)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error revoking user rights",
				"Could not revoke rights from user "+userID+": "+err.Error(),
			)
			r.refreshUpdateState(ctx, userID, resp)
			return
		}
	}

	tflog.Info(ctx, "User rights updated", map[string]interface{}{"user_id": userID})

	// Refresh from server to ensure state reflects actual rights, not potentially
	// unknown plan values.
	var result userRightsResourceModel
	result.UserID = types.StringValue(userID)
	r.writeRefreshedState(ctx, userID, &result, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

// writeRefreshedState reads current rights from the server and populates the model,
// ensuring all computed attributes have known values.
func (r *userRightsResource) writeRefreshedState(ctx context.Context, userID string, m *userRightsResourceModel, diags *diag.Diagnostics) {
	rights, err := r.userMng.ListUserRights(ctx, userID)
	if err != nil {
		diags.AddError(
			"Could not refresh rights from server",
			"Rights were modified but could not be read back for user "+userID+": "+err.Error(),
		)
		return
	}

	m.UserID = types.StringValue(userID)
	rightsToModel(ctx, rights, m, diags)
}

// refreshUpdateState reads the current rights from the server and writes them to state,
// so that Terraform's state reflects reality after a partial mutation failure.
func (r *userRightsResource) refreshUpdateState(ctx context.Context, userID string, resp *resource.UpdateResponse) {
	rights, err := r.userMng.ListUserRights(ctx, userID)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Could not refresh state after partial failure",
			"Failed to read current rights for user "+userID+": "+err.Error()+
				". State may not reflect the actual rights on the participant.",
		)
		return
	}

	var refreshed userRightsResourceModel
	refreshed.UserID = types.StringValue(userID)
	rightsToModel(ctx, rights, &refreshed, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *userRightsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userRightsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := state.UserID.ValueString()
	rights := modelToRights(ctx, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(rights) > 0 {
		tflog.Debug(ctx, "Revoking all managed user rights", map[string]interface{}{
			"user_id":     userID,
			"right_count": len(rights),
		})

		_, err := r.userMng.RevokeUserRights(ctx, userID, rights)
		if err != nil {
			// If the user was deleted out-of-band, treat as successful — nothing to revoke.
			if status.Code(err) == codes.NotFound {
				tflog.Warn(ctx, "User not found during rights delete, treating as already clean", map[string]interface{}{
					"user_id": userID,
				})
				return
			}
			resp.Diagnostics.AddError(
				"Error revoking user rights",
				"Could not revoke rights from user "+userID+": "+err.Error(),
			)
			return
		}
	}

	tflog.Info(ctx, "User rights revoked", map[string]interface{}{"user_id": userID})
}

func (r *userRightsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("user_id"), req, resp)
}

// modelToRights converts the Terraform resource model to a slice of go-daml Rights.
// Returns nil if no rights are configured or if a diagnostic error occurs.
func modelToRights(ctx context.Context, m userRightsResourceModel, diags *diag.Diagnostics) []*model.Right {
	var rights []*model.Right

	if !m.ActAs.IsNull() && !m.ActAs.IsUnknown() {
		var parties []string
		diags.Append(m.ActAs.ElementsAs(ctx, &parties, false)...)
		if diags.HasError() {
			return nil
		}
		for _, party := range parties {
			rights = append(rights, &model.Right{Type: model.CanActAs{Party: party}})
		}
	}

	if !m.ReadAs.IsNull() && !m.ReadAs.IsUnknown() {
		var parties []string
		diags.Append(m.ReadAs.ElementsAs(ctx, &parties, false)...)
		if diags.HasError() {
			return nil
		}
		for _, party := range parties {
			rights = append(rights, &model.Right{Type: model.CanReadAs{Party: party}})
		}
	}

	if m.ParticipantAdmin.ValueBool() {
		rights = append(rights, &model.Right{Type: model.ParticipantAdmin{}})
	}

	if m.IdentityProviderAdmin.ValueBool() {
		rights = append(rights, &model.Right{Type: model.IdentityProviderAdmin{}})
	}

	return rights
}

// rightsToModel converts a slice of go-daml Rights to the Terraform resource model.
func rightsToModel(ctx context.Context, rights []*model.Right, m *userRightsResourceModel, diags *diag.Diagnostics) {
	actAs := make([]string, 0)
	readAs := make([]string, 0)
	participantAdmin := false
	idpAdmin := false

	for _, right := range rights {
		switch r := right.Type.(type) {
		case model.CanActAs:
			actAs = append(actAs, r.Party)
		case model.CanReadAs:
			readAs = append(readAs, r.Party)
		case model.ParticipantAdmin:
			participantAdmin = true
		case model.IdentityProviderAdmin:
			idpAdmin = true
		default:
			diags.AddWarning(
				"Unknown right type",
				fmt.Sprintf("User has a right of unrecognized type %T that cannot be managed by this provider version. "+
					"Consider upgrading the provider.", right.Type),
			)
		}
	}

	// Use empty set (not null) to distinguish "no rights" from "not configured",
	// preventing perpetual diffs when the user writes act_as = [].
	actAsSet, d := types.SetValueFrom(ctx, types.StringType, actAs)
	diags.Append(d...)
	m.ActAs = actAsSet

	readAsSet, d := types.SetValueFrom(ctx, types.StringType, readAs)
	diags.Append(d...)
	m.ReadAs = readAsSet

	m.ParticipantAdmin = types.BoolValue(participantAdmin)
	m.IdentityProviderAdmin = types.BoolValue(idpAdmin)
}

// diffRights returns rights in `a` that are not in `b`.
func diffRights(a, b []*model.Right) []*model.Right {
	bSet := make(map[string]struct{}, len(b))
	for _, r := range b {
		bSet[rightKey(r)] = struct{}{}
	}

	var diff []*model.Right
	for _, r := range a {
		if _, found := bSet[rightKey(r)]; !found {
			diff = append(diff, r)
		}
	}
	return diff
}

// rightKey returns a string key for deduplication of rights.
func rightKey(r *model.Right) string {
	switch rt := r.Type.(type) {
	case model.CanActAs:
		return "act_as:" + rt.Party
	case model.CanReadAs:
		return "read_as:" + rt.Party
	case model.ParticipantAdmin:
		return "participant_admin"
	case model.IdentityProviderAdmin:
		return "identity_provider_admin"
	default:
		return fmt.Sprintf("unknown:%T", rt)
	}
}
