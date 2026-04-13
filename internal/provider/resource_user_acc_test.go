// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserResource_basic(t *testing.T) {
	t.Parallel()
	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	userID := "acc-user-" + suffix
	partyHint := "acc-user-party-" + suffix

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
				ResourceName:      "canton_user.test",
				ImportState:       true,
				ImportStateVerify: true,
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
