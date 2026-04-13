// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAccUserRightsResource_basic(t *testing.T) {
	t.Parallel()
	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	userID := "acc-rights-user-" + suffix

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		// The config also includes canton_user, so after full destroy the user
		// should be gone — confirming rights revocation + user deletion.
		CheckDestroy: func(s *terraform.State) error {
			client, err := testAccCantonClient()
			if err != nil {
				return err
			}
			_, err = client.UserMng.GetUser(context.Background(), userID)
			if err == nil {
				return fmt.Errorf("user %s still exists after destroy", userID)
			}
			if status.Code(err) != codes.NotFound {
				return fmt.Errorf("unexpected error checking user after destroy: %s", err)
			}
			return nil
		},
		Steps: []resource.TestStep{
			// Step 1: Create with act_as only.
			{
				Config: testAccUserRightsResourceConfig(suffix, "minimal"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("canton_user_rights.test", "act_as.#", "1"),
					resource.TestCheckResourceAttr("canton_user_rights.test", "read_as.#", "0"),
					resource.TestCheckResourceAttr("canton_user_rights.test", "participant_admin", "false"),
					resource.TestCheckResourceAttr("canton_user_rights.test", "identity_provider_admin", "false"),
				),
			},
			// Step 2: Grant — add read_as and both admin rights.
			{
				Config: testAccUserRightsResourceConfig(suffix, "full"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("canton_user_rights.test", "act_as.#", "1"),
					resource.TestCheckResourceAttr("canton_user_rights.test", "read_as.#", "1"),
					resource.TestCheckResourceAttr("canton_user_rights.test", "participant_admin", "true"),
					resource.TestCheckResourceAttr("canton_user_rights.test", "identity_provider_admin", "true"),
				),
			},
			// Step 3: Revoke — back to minimal (exercises the revoke path).
			{
				Config: testAccUserRightsResourceConfig(suffix, "minimal"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("canton_user_rights.test", "act_as.#", "1"),
					resource.TestCheckResourceAttr("canton_user_rights.test", "read_as.#", "0"),
					resource.TestCheckResourceAttr("canton_user_rights.test", "participant_admin", "false"),
					resource.TestCheckResourceAttr("canton_user_rights.test", "identity_provider_admin", "false"),
				),
			},
			// ImportState
			{
				ResourceName:                         "canton_user_rights.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "user_id",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["canton_user_rights.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return rs.Primary.Attributes["user_id"], nil
				},
			},
		},
	})
}

func testAccUserRightsResourceConfig(suffix, mode string) string {
	extras := ""
	if mode == "full" {
		extras = `
  read_as                 = [canton_party.reader.party_id]
  participant_admin       = true
  identity_provider_admin = true`
	}

	return fmt.Sprintf(`
resource "canton_party" "actor" {
  party_id_hint = "acc-rights-actor-%s"
}

resource "canton_party" "reader" {
  party_id_hint = "acc-rights-reader-%s"
}

resource "canton_user" "test" {
  user_id       = "acc-rights-user-%s"
  primary_party = canton_party.actor.party_id
}

resource "canton_user_rights" "test" {
  user_id = canton_user.test.user_id
  act_as  = [canton_party.actor.party_id]
%s
}
`, suffix, suffix, suffix, extras)
}
