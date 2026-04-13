# TODO

### Add non-empty validators to data source lookup attributes (from PR #11 review)
- [ ] Add `stringvalidator.LengthAtLeast(1)` to `party_id` in `datasource_party.go` and `user_id` in `datasource_user.go` to give cleaner error messages when empty strings are passed

### Add CheckDestroy to acceptance tests (from PR #12 review)
- [ ] Add `CheckDestroy` callbacks to `canton_user` and `canton_user_rights` acceptance tests that query the participant to verify cleanup after `terraform destroy`

### Register provider on Terraform Registry
- [ ] Register `peacefulstudio/canton` on registry.terraform.io once first release is published
