// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

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
	"github.com/noders-team/go-daml/pkg/service/admin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	_ resource.Resource                = &partyResource{}
	_ resource.ResourceWithConfigure   = &partyResource{}
	_ resource.ResourceWithImportState = &partyResource{}
)

type partyResource struct {
	partyMng admin.PartyManagement
}

type partyResourceModel struct {
	PartyIDHint types.String `tfsdk:"party_id_hint"`
	PartyID     types.String `tfsdk:"party_id"`
	IsLocal     types.Bool   `tfsdk:"is_local"`
}

func NewPartyResource() resource.Resource {
	return &partyResource{}
}

func (r *partyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_party"
}

func (r *partyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Allocates a party on a Canton participant node. Note: Canton parties cannot be deleted from the ledger; terraform destroy only removes the party from Terraform state.",
		Attributes: map[string]schema.Attribute{
			"party_id_hint": schema.StringAttribute{
				Description: "A hint for the party ID. The ledger may modify this.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"party_id": schema.StringAttribute{
				Description: "The allocated party ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_local": schema.BoolAttribute{
				Description: "Whether the party is local to this participant.",
				Computed:    true,
			},
		},
	}
}

func (r *partyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	if clients.binding.PartyMng == nil {
		resp.Diagnostics.AddError(
			"Party management service unavailable",
			"The Canton client was created but the party management service is nil. This may indicate a client initialization problem.",
		)
		return
	}

	r.partyMng = clients.binding.PartyMng
}

func (r *partyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan partyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hint := plan.PartyIDHint.ValueString()
	tflog.Debug(ctx, "Allocating party", map[string]interface{}{"party_id_hint": hint})

	party, err := r.partyMng.AllocateParty(ctx, hint, nil, "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error allocating party",
			"Could not allocate party with hint "+hint+": "+err.Error(),
		)
		return
	}

	if party == nil {
		resp.Diagnostics.AddError(
			"Error allocating party",
			"AllocateParty returned no error but a nil party for hint "+hint+". This is unexpected behavior from the participant.",
		)
		return
	}

	plan.PartyID = types.StringValue(party.Party)
	plan.IsLocal = types.BoolValue(party.IsLocal)

	tflog.Info(ctx, "Party allocated", map[string]interface{}{"party_id": party.Party})

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *partyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state partyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	partyID := state.PartyID.ValueString()

	parties, err := r.partyMng.GetParties(ctx, []string{partyID}, "")
	if err != nil {
		if status.Code(err) == codes.NotFound {
			tflog.Warn(ctx, "Party not found on participant, removing from state", map[string]interface{}{
				"party_id": partyID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading party",
			"Could not read party "+partyID+": "+err.Error(),
		)
		return
	}

	// Canton parties are immutable — an empty result likely indicates a transient or
	// permission issue, not deletion. Keep state intact to avoid accidental re-allocation.
	if len(parties) == 0 {
		resp.Diagnostics.AddError(
			"Party not found",
			"Party "+partyID+" was not returned by the participant. Since Canton parties cannot be deleted, "+
				"this likely indicates a connectivity or permission issue. State is preserved to prevent "+
				"accidental re-allocation. Verify the participant is reachable and the credentials have access.",
		)
		return
	}

	party := parties[0]
	state.PartyID = types.StringValue(party.Party)
	state.IsLocal = types.BoolValue(party.IsLocal)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update should never be called because all attributes use RequiresReplace, but return an error defensively.
func (r *partyResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Party resources cannot be updated in-place.")
}

func (r *partyResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Canton parties cannot be deleted. Remove from state only.
	tflog.Warn(ctx, "Canton parties cannot be deleted from the ledger. Removing from Terraform state only.")
}

func (r *partyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Set both party_id and party_id_hint from the import ID. The hint cannot be
	// derived from the participant after allocation, so we use the full party ID
	// as the hint value to keep state consistent and avoid forced replacement.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("party_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("party_id_hint"), req.ID)...)
}
