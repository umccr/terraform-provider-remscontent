## This data source looks up a user by email and returns their user_id,
## for use in a workflow resource.
data "remscontent_actor" "user_1" {
  email = "user_1@email.com"
}
