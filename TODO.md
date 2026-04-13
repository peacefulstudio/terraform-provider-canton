# TODO

### Add non-empty validators to data source lookup attributes (from PR #11 review)
- [ ] Add `stringvalidator.LengthAtLeast(1)` to `party_id` in `datasource_party.go` and `user_id` in `datasource_user.go` to give cleaner error messages when empty strings are passed
