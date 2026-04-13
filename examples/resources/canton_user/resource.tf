resource "canton_party" "treasury" {
  party_id_hint = "treasury"
}

resource "canton_user" "treasury_service" {
  user_id       = "treasury-service"
  primary_party = canton_party.treasury.party_id
}
