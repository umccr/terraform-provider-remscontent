terraform {
  required_providers {
    remscontent = {
      source = "registry.terraform.io/umccr/remscontent"
    }
  }
}

provider "remscontent" {

}

variable "email" {}

data "remscontent_blacklist_user" "user_1" {
  email = var.email
}

data "remscontent_organization" "ccgcm_org" {
  id = "Collaborative Centre for Genomic Cancer Medicine"
}

resource "remscontent_resource" "resource_1" {
  organization_id = data.remscontent_organization.ccgcm_org.id
  resid           = "uri-resource-1"
  licenses        = [3]
  archived        = false
  enabled         = true
}

resource "remscontent_blacklist" "user1" {
  resource_ext_id = resource.remscontent_resource.resource_1.resid
  user_id         = data.remscontent_blacklist_user.user_1.id
  comment         = "user_1 is bad"
}

resource "remscontent_catalogue_item" "item1" {
  organization_id = data.remscontent_organization.ccgcm_org.id
  resource_id     = resource.remscontent_resource.resource_1.id
  workflow_id     = 11
  form_id         = 17
  localizations = {
    title   = "The title for catalogue item 1 - edit01"
    infourl = "url for catalogue item 1"
  }
}





resource "remscontent_category" "category_3" {
  description = "description for category 3"
  title       = "category-03"
}


resource "remscontent_category" "category_2" {
  description = "description for category 2"
  title       = "category-02"
}
