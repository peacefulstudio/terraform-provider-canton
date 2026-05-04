// Copyright (c) 2026 Peaceful Studio OÜ
// SPDX-License-Identifier: Apache-2.0

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

func TestAccUserResource_basic(t *testing.T) {
	t.Parallel()
	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	userID := "acc-user-" + suffix
	partyHint := "acc-user-party-" + suffix

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		// Verify the user was actually deleted from the participant.
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
			{
				Config: testAccUserResourceConfig(userID, partyHint),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("canton_user.test", "user_id", userID),
					resource.TestCheckResourceAttrSet("canton_user.test", "primary_party"),
				),
			},
			// ImportState
			{
				ResourceName:                         "canton_user.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "user_id",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["canton_user.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return rs.Primary.Attributes["user_id"], nil
				},
			},
		},
	})
}

func testAccUserResourceConfig(userID, partyHint string) string {
	return fmt.Sprintf(`
resource "canton_party" "test" {
  party_id_hint = %q
}

resource "canton_user" "test" {
  user_id       = %q
  primary_party = canton_party.test.party_id
}
`, partyHint, userID)
}
