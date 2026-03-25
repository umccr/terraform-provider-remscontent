# Simple Example — Creating a Catalogue Item

This example demonstrates how to create a **Catalogue Item** using the
`remscontent` Terraform provider, with several supporting resources and
data source lookups.

## What This Example Does

| Step | Type | Name | Purpose |
|------|------|------|---------|
| 1 | `data` | `remscontent_organization` | Looks up an existing organisation by ID |
| 2 | `data` | `remscontent_license` | Looks up an existing licence by title |
| 3 | `data` | `remscontent_workflow` | Looks up an existing workflow by title |
| 4 | `data` | `remscontent_form` | Looks up an existing form by internal name |
| 5 | `data` | `remscontent_blacklist_user` | Looks up a user by email address |
| 6 | `resource` | `remscontent_resource` | Creates a new content resource under the organisation |
| 7 | `resource` | `remscontent_category` | Creates a new category |
| 8 | `resource` | `remscontent_blacklist` | Blacklists the looked-up user from the resource |
| 9 | `resource` | `remscontent_catalogue_item` | **Creates the catalogue item** wiring together the resource, workflow, and form |
