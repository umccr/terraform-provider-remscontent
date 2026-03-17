## This data source looks up a user by email and returns their user_id,
## for use in a blacklist entry. It does NOT check if the user is already blacklisted.
## It only finds users who are available to be blacklisted (not currently blacklisted).
## Use this when you know the user's email but not their user_id.
data "remscontent_blacklist_user" "user_1" {
  email = "user_1@email.com"
}
