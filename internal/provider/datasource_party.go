// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/noders-team/go-daml/pkg/service/admin"
)

var (
	_ datasource.DataSource              = &partyDataSource{}
	_ datasource.DataSourceWithConfigure = &partyDataSource{}
)

type partyDataSource struct {
	partyMng admin.PartyManagement
}

type partyDataSourceModel struct {
	PartyID types.String `tfsdk:"party_id"`
	IsLocal types.Bool   `tfsdk:"is_local"`
}

func NewPartyDataSource() datasource.DataSource {
	return &partyDataSource{}
}

func (d *partyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_party"
}

func (d *partyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing party on a Canton participant node.",
		Attributes: map[string]schema.Attribute{
			"party_id": schema.StringAttribute{
				Description: "The party ID to look up.",
				Required:    true,
			},
			"is_local": schema.BoolAttribute{
				Description: "Whether the party is local to this participant.",
				Computed:    true,
			},
		},
	}
}

func (d *partyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*cantonClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
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

	d.partyMng = clients.binding.PartyMng
}

func (d *partyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config partyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	partyID := config.PartyID.ValueString()
	tflog.Debug(ctx, "Reading party data source", map[string]interface{}{"party_id": partyID})

	parties, err := d.partyMng.GetParties(ctx, []string{partyID}, "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading party",
			"Could not read party "+partyID+": "+err.Error(),
		)
		return
	}

	if len(parties) == 0 {
		resp.Diagnostics.AddError(
			"Party not found",
			"Party "+partyID+" was not found on the participant.",
		)
		return
	}

	party := parties[0]
	if party == nil {
		resp.Diagnostics.AddError(
			"Error reading party",
			"The Canton participant returned a nil party entry for "+partyID+".",
		)
		return
	}

	config.PartyID = types.StringValue(party.Party)
	config.IsLocal = types.BoolValue(party.IsLocal)

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
