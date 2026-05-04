// Copyright (c) 2026 Peaceful Studio OÜ
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	damlclient "github.com/noders-team/go-daml/pkg/client"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// pingTimeout bounds the configure-time health check. Canton's VersionService
// responds instantly on a healthy participant; anything longer than a few
// seconds almost certainly means the connection, TLS, or auth is broken.
const pingTimeout = 10 * time.Second

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

	rawParticipantURL := stringValueOrEnv(config.ParticipantURL, "CANTON_PARTICIPANT_URL")
	useTLS, explicitScheme := participantScheme(rawParticipantURL)
	participantURL := normalizeParticipantURL(rawParticipantURL)
	if participantURL == "" {
		resp.Diagnostics.AddError(
			"Missing participant URL",
			"participant_url must be set in provider config or CANTON_PARTICIPANT_URL environment variable",
		)
		return
	}

	tokenSource, diags := resolveOAuth2TokenSource(ctx, config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if tokenSource != nil && !useTLS {
		// A bearer token over plaintext is a credential leak. Warn loudly
		// rather than refuse, because CI and local dev legitimately use http://
		// against a localhost participant that ignores the token anyway.
		msg := "participant_url is plaintext but an OAuth2 token is configured; the token will be sent unencrypted."
		if !explicitScheme {
			msg += " Prefix the URL with https:// (or http:// if you really mean plaintext) to silence this warning."
		}
		resp.Diagnostics.AddWarning("OAuth2 token sent over plaintext", msg)
	}

	dialOpts := []grpc.DialOption{}
	if useTLS {
		dialOpts = append(dialOpts,
			grpc.WithContextDialer(tlsDialer(&tls.Config{
				NextProtos: []string{"h2"},
				MinVersion: tls.VersionTLS12,
			})),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	if tokenSource != nil {
		dialOpts = append(dialOpts,
			grpc.WithUnaryInterceptor(tokenUnaryInterceptor(tokenSource)),
			grpc.WithStreamInterceptor(tokenStreamInterceptor(tokenSource)),
		)
	}

	grpcConn, err := grpc.NewClient(participantURL, dialOpts...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create Canton client",
			"Could not create gRPC client for "+participantURL+": "+err.Error(),
		)
		return
	}

	// NewConnection and NewDamlBindingClient are given nil for *DamlClient
	// because we've bypassed go-daml's own TLS/auth plumbing with hand-rolled
	// dial options. Connection.AdminGRPCConn() falls back to the main conn
	// when the admin conn is nil (go-daml client.go), so both nils are safe.
	conn := damlclient.NewConnection(nil, grpcConn, nil)
	binding := damlclient.NewDamlBindingClient(nil, conn)

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := binding.Ping(pingCtx); err != nil {
		resp.Diagnostics.AddError(
			"Canton participant unreachable",
			fmt.Sprintf("Health check against %s failed: %v. Verify the URL, TLS configuration, and OAuth2 credentials.", participantURL, err),
		)
		if closeErr := grpcConn.Close(); closeErr != nil {
			resp.Diagnostics.AddWarning(
				"Failed to close Canton client",
				fmt.Sprintf("The gRPC client created for %s could not be closed after a failed health check: %v", participantURL, closeErr),
			)
		}
		return
	}

	tflog.Info(ctx, "Canton provider configured", map[string]interface{}{
		"participant_url": participantURL,
		"tls":             useTLS,
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
	return []func() datasource.DataSource{
		NewPartyDataSource,
		NewUserDataSource,
		NewPartiesDataSource,
	}
}

// participantScheme reports whether the raw URL uses TLS (https://) and whether
// the scheme was explicit. Scheme detection is case-insensitive. A URL with no
// scheme is treated as plaintext-but-ambiguous so the caller can warn.
func participantScheme(raw string) (useTLS, explicit bool) {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.HasPrefix(lower, "https://"):
		return true, true
	case strings.HasPrefix(lower, "http://"):
		return false, true
	default:
		return false, false
	}
}

// normalizeParticipantURL strips an http(s) scheme prefix and any trailing
// slash so the result is a bare host:port suitable for gRPC dial. The scheme
// match is case-insensitive.
func normalizeParticipantURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	lower := strings.ToLower(trimmed)
	switch {
	case strings.HasPrefix(lower, "https://"):
		trimmed = trimmed[len("https://"):]
	case strings.HasPrefix(lower, "http://"):
		trimmed = trimmed[len("http://"):]
	}
	return strings.TrimRight(trimmed, "/")
}

func stringValueOrEnv(value types.String, envVar string) string {
	if !value.IsNull() && !value.IsUnknown() {
		return value.ValueString()
	}
	return os.Getenv(envVar)
}

// resolveOAuth2TokenSource builds a refreshing OAuth2 token source from either
// the provider's oauth2 block or CANTON_OAUTH2_* env vars. Returns nil when
// OAuth2 is not configured (anonymous connection).
func resolveOAuth2TokenSource(ctx context.Context, config cantonProviderModel) (oauth2.TokenSource, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	oauthBlockPresent := !config.OAuth2.IsNull() && !config.OAuth2.IsUnknown()
	oauthEnvPresent := os.Getenv("CANTON_OAUTH2_TOKEN_URL") != ""
	if !oauthBlockPresent && !oauthEnvPresent {
		return nil, nil
	}

	var tokenURL, clientID, clientSecret, audience, scope string
	if oauthBlockPresent {
		var oauthConfig oauth2Model
		diagnostics.Append(config.OAuth2.As(ctx, &oauthConfig, basetypes.ObjectAsOptions{})...)
		if diagnostics.HasError() {
			return nil, diagnostics
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
		diagnostics.AddError("Missing OAuth2 token URL",
			"oauth2.token_url must be set in provider config or CANTON_OAUTH2_TOKEN_URL environment variable when oauth2 is configured")
	}
	if clientID == "" {
		diagnostics.AddError("Missing OAuth2 client ID",
			"oauth2.client_id must be set in provider config or CANTON_OAUTH2_CLIENT_ID environment variable when oauth2 is configured")
	}
	if clientSecret == "" {
		diagnostics.AddError("Missing OAuth2 client secret",
			"oauth2.client_secret must be set in provider config or CANTON_OAUTH2_CLIENT_SECRET environment variable when oauth2 is configured")
	}
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	ccConfig := clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
	}
	if audience != "" {
		ccConfig.EndpointParams = map[string][]string{"audience": {audience}}
	}
	if scope != "" {
		ccConfig.Scopes = []string{scope}
	}

	// Fail fast: fetch an initial token so misconfiguration surfaces at
	// Configure time rather than on the first RPC. Reuse that token to seed
	// the refreshing source so the first RPC doesn't pay for another round trip.
	tok, err := ccConfig.Token(ctx)
	if err != nil {
		diagnostics.AddError("OAuth2 token error", "Failed to obtain OAuth2 token: "+err.Error())
		return nil, diagnostics
	}
	if tok.AccessToken == "" {
		diagnostics.AddError("OAuth2 token is empty",
			"The OAuth2 token endpoint returned a successful response but the access token is empty. Check your OAuth2 provider configuration.")
		return nil, diagnostics
	}

	tflog.Info(ctx, "OAuth2 authentication configured")
	return oauth2.ReuseTokenSource(tok, ccConfig.TokenSource(ctx)), diagnostics
}

// tlsDialer returns a gRPC context dialer that terminates TLS itself using the
// supplied config. gRPC sees the connection as plaintext (pair this dialer
// with insecure.NewCredentials()), which sidesteps grpc-go's post-1.67 ALPN
// enforcement — some edge proxies (Qovery, Envoy Gateway) don't advertise h2
// back on ALPN even when they do speak HTTP/2. Returned conns that negotiate a
// non-h2 ALPN (e.g. an HTTP/1.1 reverse proxy) are rejected because the first
// RPC would otherwise fail deep in the HTTP/2 framer.
func tlsDialer(base *tls.Config) func(ctx context.Context, addr string) (net.Conn, error) {
	return func(ctx context.Context, addr string) (net.Conn, error) {
		cfg := base.Clone()
		if cfg.ServerName == "" {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				host = addr
			}
			cfg.ServerName = host
		}
		d := &tls.Dialer{Config: cfg}
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("tls dial %s: %w", addr, err)
		}
		tlsConn := conn.(*tls.Conn)
		if alpn := tlsConn.ConnectionState().NegotiatedProtocol; alpn != "" && alpn != "h2" {
			_ = conn.Close()
			return nil, fmt.Errorf("server %s negotiated ALPN %q; gRPC requires h2", addr, alpn)
		}
		return conn, nil
	}
}

// tokenUnaryInterceptor attaches a fresh bearer token from ts to each unary
// RPC. Using a TokenSource (rather than a captured string) lets short-lived
// tokens refresh transparently mid-apply.
func tokenUnaryInterceptor(ts oauth2.TokenSource) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		authCtx, err := withBearer(ctx, ts)
		if err != nil {
			return err
		}
		return invoker(authCtx, method, req, reply, cc, opts...)
	}
}

func tokenStreamInterceptor(ts oauth2.TokenSource) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		authCtx, err := withBearer(ctx, ts)
		if err != nil {
			return nil, err
		}
		return streamer(authCtx, desc, cc, method, opts...)
	}
}

func withBearer(ctx context.Context, ts oauth2.TokenSource) (context.Context, error) {
	tok, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("oauth2 token refresh failed: %w", err)
	}
	if tok == nil {
		return nil, errors.New("oauth2 token refresh returned nil token")
	}
	if strings.TrimSpace(tok.AccessToken) == "" {
		return nil, errors.New("oauth2 token refresh returned empty access token")
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+tok.AccessToken), nil
}
