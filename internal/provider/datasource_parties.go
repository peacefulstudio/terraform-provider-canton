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

const maxPaginationPages = 10_000

var (
	_ datasource.DataSource              = &partiesDataSource{}
	_ datasource.DataSourceWithConfigure = &partiesDataSource{}
)

type partiesDataSource struct {
	partyMng admin.PartyManagement
}

type partiesDataSourceModel struct {
	Parties []partyDataSourceModel `tfsdk:"parties"`
}

func NewPartiesDataSource() datasource.DataSource {
	return &partiesDataSource{}
}

func (d *partiesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_parties"
}

func (d *partiesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all known parties on a Canton participant node.",
		Attributes: map[string]schema.Attribute{
			"parties": schema.ListNestedAttribute{
				Description: "List of known parties.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"party_id": schema.StringAttribute{
							Description: "The party ID.",
							Computed:    true,
						},
						"is_local": schema.BoolAttribute{
							Description: "Whether the party is local to this participant.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *partiesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *partiesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading parties data source")

	var allParties []partyDataSourceModel
	pageToken := ""
	seen := make(map[string]bool)
	allPagesRead := false

	for range maxPaginationPages {
		result, err := d.partyMng.ListKnownParties(ctx, pageToken, 0, "")
		if err != nil {
			resp.Diagnostics.AddError(
				"Error listing parties",
				"Could not list known parties: "+err.Error(),
			)
			return
		}

		if result == nil {
			resp.Diagnostics.AddError(
				"Error listing parties",
				"Received an empty response from the Canton participant while listing known parties.",
			)
			return
		}

		for _, p := range result.PartyDetails {
			if p == nil {
				continue
			}
			allParties = append(allParties, partyDataSourceModel{
				PartyID: types.StringValue(p.Party),
				IsLocal: types.BoolValue(p.IsLocal),
			})
		}

		if result.NextPageToken == "" {
			allPagesRead = true
			break
		}
		if seen[result.NextPageToken] {
			resp.Diagnostics.AddError(
				"Error listing parties",
				"Detected a pagination cycle (repeated page token). This indicates a server-side bug.",
			)
			return
		}
		seen[result.NextPageToken] = true
		pageToken = result.NextPageToken
	}

	if !allPagesRead {
		resp.Diagnostics.AddWarning(
			"Partial result — pagination limit reached",
			fmt.Sprintf("Listed parties across %d pages but more pages remain. "+
				"The result may be incomplete.", maxPaginationPages),
		)
	}

	// Use empty slice (not nil) so Terraform sees an empty list, not null.
	if allParties == nil {
		allParties = []partyDataSourceModel{}
	}

	state := partiesDataSourceModel{Parties: allParties}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
