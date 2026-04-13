resource "canton_party" "treasury" {
  party_id_hint = "treasury"
}

resource "canton_party" "validator" {
  party_id_hint = "validator"
}

resource "canton_user" "treasury_service" {
  user_id       = "treasury-service"
  primary_party = canton_party.treasury.party_id
}

resource "canton_user_rights" "treasury_rights" {
  user_id = canton_user.treasury_service.user_id

  act_as  = [canton_party.treasury.party_id]
  read_as = [canton_party.validator.party_id]

  participant_admin       = false
  identity_provider_admin = false
}
