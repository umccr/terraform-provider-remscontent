# Provider configuration using explicit values.
# All attributes can alternatively be set via environment variables:
#   REMS_ENDPOINT, REMS_API_USER, REMS_API_KEY, REMS_LANGUAGE
provider "remscontent" {
  endpoint = "rems.example.org" # DNS name only, no https://
  api_user = "admin@example.org"
  api_key  = "my-secret-api-key"
  language = "en" # Localization language for all resources (e.g. "en", "fi")
}

