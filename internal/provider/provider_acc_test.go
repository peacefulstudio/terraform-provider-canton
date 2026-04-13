// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	damlclient "github.com/noders-team/go-daml/pkg/client"
	"golang.org/x/oauth2/clientcredentials"
)

// regexpAtLeastOne matches any integer >= 1 (used to assert non-empty lists).
var regexpAtLeastOne = regexp.MustCompile(`^[1-9]\d*$`)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"canton": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("CANTON_PARTICIPANT_URL") == "" {
		t.Fatal("CANTON_PARTICIPANT_URL must be set for acceptance tests")
	}
}

// testAccCantonClient builds a go-daml binding client using the same env vars
// as the provider. Used by CheckDestroy callbacks to verify server-side state.
func testAccCantonClient() (*damlclient.DamlBindingClient, error) {
	var token string
	if tokenURL := os.Getenv("CANTON_OAUTH2_TOKEN_URL"); tokenURL != "" {
		ccConfig := clientcredentials.Config{
			ClientID:     os.Getenv("CANTON_OAUTH2_CLIENT_ID"),
			ClientSecret: os.Getenv("CANTON_OAUTH2_CLIENT_SECRET"),
			TokenURL:     tokenURL,
		}
		if audience := os.Getenv("CANTON_OAUTH2_AUDIENCE"); audience != "" {
			ccConfig.EndpointParams = map[string][]string{"audience": {audience}}
		}
		if scope := os.Getenv("CANTON_OAUTH2_SCOPE"); scope != "" {
			ccConfig.Scopes = []string{scope}
		}
		tok, err := ccConfig.Token(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to obtain OAuth2 token: %w", err)
		}
		token = tok.AccessToken
	}

	client := damlclient.NewDamlClient(token, os.Getenv("CANTON_PARTICIPANT_URL"))
	binding, err := client.Build(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to build Canton client: %w", err)
	}
	return binding, nil
}
