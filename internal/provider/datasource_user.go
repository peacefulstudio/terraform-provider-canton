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
	_ datasource.DataSource              = &userDataSource{}
	_ datasource.DataSourceWithConfigure = &userDataSource{}
)

type userDataSource struct {
	userMng admin.UserManagement
}

type userDataSourceModel struct {
	UserID       types.String `tfsdk:"user_id"`
	PrimaryParty types.String `tfsdk:"primary_party"`
}

func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing user on a Canton participant node.",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.StringAttribute{
				Description: "The user ID to look up.",
				Required:    true,
			},
			"primary_party": schema.StringAttribute{
				Description: "The primary party for the user.",
				Computed:    true,
			},
		},
	}
}

func (d *userDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	if clients.binding.UserMng == nil {
		resp.Diagnostics.AddError(
			"User management service unavailable",
			"The Canton client was created but the user management service is nil. This may indicate a client initialization problem.",
		)
		return
	}

	d.userMng = clients.binding.UserMng
}

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config userDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := config.UserID.ValueString()
	tflog.Debug(ctx, "Reading user data source", map[string]interface{}{"user_id": userID})

	user, err := d.userMng.GetUser(ctx, userID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading user",
			"Could not read user "+userID+": "+err.Error(),
		)
		return
	}

	if user == nil {
		resp.Diagnostics.AddError(
			"Unexpected nil user response",
			"The Canton participant returned a nil user entry for "+userID+". This may indicate a server-side bug.",
		)
		return
	}

	config.UserID = types.StringValue(user.ID)
	config.PrimaryParty = types.StringValue(user.PrimaryParty)

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
