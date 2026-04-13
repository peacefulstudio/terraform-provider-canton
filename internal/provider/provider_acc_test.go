// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
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
