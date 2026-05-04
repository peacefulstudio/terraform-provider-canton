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

func TestAccPartyDataSource_basic(t *testing.T) {
	t.Parallel()
	hint := "acc-ds-party-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "canton_party" "source" {
  party_id_hint = %q
}

data "canton_party" "test" {
  party_id = canton_party.source.party_id
}
`, hint),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.canton_party.test", "party_id",
						"canton_party.source", "party_id",
					),
					// Locally allocated parties are always local to this participant.
					resource.TestCheckResourceAttr("data.canton_party.test", "is_local", "true"),
				),
			},
		},
	})
}

func TestAccUserDataSource_basic(t *testing.T) {
	t.Parallel()
	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	userID := "acc-ds-user-" + suffix

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
				Config: fmt.Sprintf(`
resource "canton_party" "source" {
  party_id_hint = "acc-ds-user-party-%s"
}

resource "canton_user" "source" {
  user_id       = "acc-ds-user-%s"
  primary_party = canton_party.source.party_id
}

data "canton_user" "test" {
  user_id = canton_user.source.user_id
}
`, suffix, suffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.canton_user.test", "user_id",
						"canton_user.source", "user_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.canton_user.test", "primary_party",
						"canton_user.source", "primary_party",
					),
				),
			},
		},
	})
}

func TestAccPartiesDataSource_basic(t *testing.T) {
	t.Parallel()
	hint := "acc-ds-parties-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "canton_party" "source" {
  party_id_hint = %q
}

data "canton_parties" "test" {
  depends_on = [canton_party.source]
}
`, hint),
				Check: resource.ComposeAggregateTestCheckFunc(
					// At least the party we just created should appear.
					resource.TestMatchResourceAttr(
						"data.canton_parties.test", "parties.#",
						regexpAtLeastOne,
					),
				),
			},
		},
	})
}
