// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	damlclient "github.com/noders-team/go-daml/pkg/client"
	"golang.org/x/oauth2/clientcredentials"
)

var _ provider.Provider = &cantonProvider{}

type cantonProvider struct {
	version string
}

type cantonProviderModel struct {
	ParticipantURL types.String `tfsdk:"participant_url"`
	OAuth2         types.Object `tfsdk:"oauth2"`
}

type oauth2Model struct {
	TokenURL     types.String `tfsdk:"token_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	Audience     types.String `tfsdk:"audience"`
	Scope        types.String `tfsdk:"scope"`
}

// cantonClients holds the initialized go-daml binding client for use by resources.
// The gRPC connection is not explicitly closed because Terraform providers are
// short-lived processes and the Plugin Framework has no shutdown hook.
type cantonClients struct {
	binding *damlclient.DamlBindingClient
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &cantonProvider{
			version: version,
		}
	}
}

func (p *cantonProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "canton"
	resp.Version = p.version
}

func (p *cantonProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for Canton participant node administration (parties, users, rights).",
		Attributes: map[string]schema.Attribute{
			"participant_url": schema.StringAttribute{
				Description: "The gRPC endpoint of the Canton participant node. Can also be set via CANTON_PARTICIPANT_URL.",
				Optional:    true,
			},
			"oauth2": schema.SingleNestedAttribute{
				Description: "OAuth2 client credentials for authenticating with the participant. " +
					"Can also be configured entirely via environment variables (CANTON_OAUTH2_*) without this block.",
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"token_url": schema.StringAttribute{
						Description: "OAuth2 token endpoint URL. Can also be set via CANTON_OAUTH2_TOKEN_URL.",
						Optional:    true,
					},
					"client_id": schema.StringAttribute{
						Description: "OAuth2 client ID. Can also be set via CANTON_OAUTH2_CLIENT_ID.",
						Optional:    true,
					},
					"client_secret": schema.StringAttribute{
						Description: "OAuth2 client secret. Can also be set via CANTON_OAUTH2_CLIENT_SECRET.",
						Optional:    true,
						Sensitive:   true,
					},
					"audience": schema.StringAttribute{
						Description: "OAuth2 audience. Can also be set via CANTON_OAUTH2_AUDIENCE.",
						Optional:    true,
					},
					"scope": schema.StringAttribute{
						Description: "OAuth2 scope. Can also be set via CANTON_OAUTH2_SCOPE.",
						Optional:    true,
					},
				},
			},
		},
	}
}

func (p *cantonProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config cantonProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	participantURL := stringValueOrEnv(config.ParticipantURL, "CANTON_PARTICIPANT_URL")
	if participantURL == "" {
		resp.Diagnostics.AddError(
			"Missing participant URL",
			"participant_url must be set in provider config or CANTON_PARTICIPANT_URL environment variable",
		)
		return
	}

	// Resolve OAuth2 token if configured — either via the oauth2 block or env vars.
	var token string
	oauthBlockPresent := !config.OAuth2.IsNull() && !config.OAuth2.IsUnknown()
	oauthEnvPresent := os.Getenv("CANTON_OAUTH2_TOKEN_URL") != ""

	if oauthBlockPresent || oauthEnvPresent {
		var tokenURL, clientID, clientSecret, audience, scope string

		if oauthBlockPresent {
			var oauthConfig oauth2Model
			resp.Diagnostics.Append(config.OAuth2.As(ctx, &oauthConfig, basetypes.ObjectAsOptions{})...)
			if resp.Diagnostics.HasError() {
				return
			}
			tokenURL = stringValueOrEnv(oauthConfig.TokenURL, "CANTON_OAUTH2_TOKEN_URL")
			clientID = stringValueOrEnv(oauthConfig.ClientID, "CANTON_OAUTH2_CLIENT_ID")
			clientSecret = stringValueOrEnv(oauthConfig.ClientSecret, "CANTON_OAUTH2_CLIENT_SECRET")
			audience = stringValueOrEnv(oauthConfig.Audience, "CANTON_OAUTH2_AUDIENCE")
			scope = stringValueOrEnv(oauthConfig.Scope, "CANTON_OAUTH2_SCOPE")
		} else {
			tokenURL = os.Getenv("CANTON_OAUTH2_TOKEN_URL")
			clientID = os.Getenv("CANTON_OAUTH2_CLIENT_ID")
			clientSecret = os.Getenv("CANTON_OAUTH2_CLIENT_SECRET")
			audience = os.Getenv("CANTON_OAUTH2_AUDIENCE")
			scope = os.Getenv("CANTON_OAUTH2_SCOPE")
		}

		if tokenURL == "" {
			resp.Diagnostics.AddError(
				"Missing OAuth2 token URL",
				"oauth2.token_url must be set in provider config or CANTON_OAUTH2_TOKEN_URL environment variable when oauth2 is configured",
			)
		}
		if clientID == "" {
			resp.Diagnostics.AddError(
				"Missing OAuth2 client ID",
				"oauth2.client_id must be set in provider config or CANTON_OAUTH2_CLIENT_ID environment variable when oauth2 is configured",
			)
		}
		if clientSecret == "" {
			resp.Diagnostics.AddError(
				"Missing OAuth2 client secret",
				"oauth2.client_secret must be set in provider config or CANTON_OAUTH2_CLIENT_SECRET environment variable when oauth2 is configured",
			)
		}
		if resp.Diagnostics.HasError() {
			return
		}

		ccConfig := clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     tokenURL,
		}

		if audience != "" {
			ccConfig.EndpointParams = map[string][]string{
				"audience": {audience},
			}
		}

		if scope != "" {
			ccConfig.Scopes = []string{scope}
		}

		oauthToken, err := ccConfig.Token(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"OAuth2 token error",
				"Failed to obtain OAuth2 token: "+err.Error(),
			)
			return
		}

		token = oauthToken.AccessToken
		if token == "" {
			resp.Diagnostics.AddError(
				"OAuth2 token is empty",
				"The OAuth2 token endpoint returned a successful response but the access token is empty. Check your OAuth2 provider configuration.",
			)
			return
		}

		tflog.Info(ctx, "OAuth2 authentication configured")
	}

	client := damlclient.NewDamlClient(token, participantURL)
	binding, err := client.Build(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create Canton client",
			"Could not create gRPC client for "+participantURL+": "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Canton provider configured", map[string]interface{}{
		"participant_url": participantURL,
	})

	clients := &cantonClients{binding: binding}
	resp.DataSourceData = clients
	resp.ResourceData = clients
}

func (p *cantonProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPartyResource,
		NewUserResource,
		NewUserRightsResource,
	}
}

func (p *cantonProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func stringValueOrEnv(value types.String, envVar string) string {
	if !value.IsNull() && !value.IsUnknown() {
		return value.ValueString()
	}
	return os.Getenv(envVar)
}
