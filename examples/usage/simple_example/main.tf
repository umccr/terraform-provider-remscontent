terraform {
  required_providers {
    remscontent = {
      source = "registry.terraform.io/umccr/remscontent"
    }
  }
}
variable "email" {}
variable "REMS_ENDPOINT" {}
variable "REMS_API_KEY" {}
variable "REMS_API_USER" {}

provider "remscontent" {
  endpoint = var.REMS_ENDPOINT # DNS name only, no https://
  api_user = var.REMS_API_USER
  api_key  = var.REMS_API_KEY
  language = "en" # Localization language for all resources (e.g. "en", "fi")
}


data "remscontent_blacklist_user" "user_1" {
  email = var.email
}

data "remscontent_organization" "ccgcm_org" {
  id = "Collaborative Centre for Genomic Cancer Medicine"
}

data "remscontent_license" "license_1" {
  title = "Test License"
}

resource "remscontent_resource" "resource_1" {
  organization_id = data.remscontent_organization.ccgcm_org.id
  resource_ext_id = "uri-resource-1"
  licenses        = [data.remscontent_license.license_1.id]
  archived        = false
  enabled         = true
}

resource "remscontent_blacklist" "user1" {
  resource_ext_id = resource.remscontent_resource.resource_1.resource_ext_id
  user_id         = data.remscontent_blacklist_user.user_1.id
  comment         = "user_1 is bad"
}

resource "remscontent_category" "category_2" {
  description   = "description for category 2"
  title         = "category-02"
  display_order = 2
}

resource "remscontent_catalogue_item" "item1" {
  organization_id = data.remscontent_organization.ccgcm_org.id
  resource_id     = resource.remscontent_resource.resource_1.id
  # The following is just putting id rather than using data/resource to fetch the id
  workflow_id = 11
  form_id     = 17
  localizations = {
    title   = "The title for catalogue item 1 - edit01"
    infourl = "url for catalogue item 1"
  }
}

