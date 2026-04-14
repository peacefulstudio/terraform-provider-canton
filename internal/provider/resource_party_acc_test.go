// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPartyResource_basic(t *testing.T) {
	t.Parallel()
	hint := "acc-party-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	var partyID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		// Party delete is a no-op — verify the party still exists on the participant.
		CheckDestroy: func(s *terraform.State) error {
			if partyID == "" {
				return fmt.Errorf("party_id was never captured")
			}
			client, err := testAccCantonClient()
			if err != nil {
				return err
			}
			parties, err := client.PartyMng.GetParties(context.Background(), []string{partyID}, "")
			if err != nil {
				return fmt.Errorf("error checking party after destroy: %s", err)
			}
			if len(parties) == 0 {
				return fmt.Errorf("party %s was unexpectedly deleted from the participant", partyID)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testAccPartyResourceConfig(hint),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("canton_party.test", "party_id_hint", hint),
					resource.TestCheckResourceAttrSet("canton_party.test", "party_id"),
					resource.TestCheckResourceAttrSet("canton_party.test", "is_local"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["canton_party.test"]
						if !ok {
							return fmt.Errorf("canton_party.test not found in state")
						}
						partyID = rs.Primary.Attributes["party_id"]
						return nil
					},
				),
			},
			// ImportState — import the party by its allocated party_id.
			// party_id_hint is still ignored because Canton may modify the hint
			// during allocation, so the extracted prefix may not match exactly.
			{
				ResourceName:                         "canton_party.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "party_id",
				ImportStateVerifyIgnore:              []string{"party_id_hint"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["canton_party.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return rs.Primary.Attributes["party_id"], nil
				},
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
