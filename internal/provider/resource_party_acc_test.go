// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPartyResource_basic(t *testing.T) {
	t.Parallel()
	hint := "acc-party-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPartyResourceConfig(hint),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("canton_party.test", "party_id_hint", hint),
					resource.TestCheckResourceAttrSet("canton_party.test", "party_id"),
					resource.TestCheckResourceAttrSet("canton_party.test", "is_local"),
				),
			},
			// ImportState — import the party by its allocated party_id.
			// party_id_hint is ignored because import sets it to the full party ID,
			// which differs from the original hint used during allocation.
			{
				ResourceName:            "canton_party.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"party_id_hint"},
			},
		},
	})
}

func testAccPartyResourceConfig(hint string) string {
	return fmt.Sprintf(`
resource "canton_party" "test" {
  party_id_hint = %q
}
`, hint)
}
