terraform {
  required_providers {
    remscontent = {
      source = "registry.terraform.io/umccr/remscontent"
    }
  }
}

provider "remscontent" {}

resource "remscontent_resource" "resource_1" {
  # Define your resource here. See docs/resources/workflow.md for required arguments.
}

resource "remscontent_workflow" "workflow_1" {
  # Define your workflow here. See docs/resources/workflow.md for required arguments.
}

resource "remscontent_form" "form_1" {
  # Define your form here. See docs/resources/form.md for required arguments.
}

resource "remscontent_category" "category_1" {
  description = "description for category 1"
  title       = "category-01"
}

resource "remscontent_category" "category_2" {
  description = "description for category 2"
  title       = "category-01"
}

resource "remscontent_catalogue_item" "item1" {
  organization_id = data.remscontent_organization.ccgcm_org.id
  resource_id     = resource.remscontent_resource.resource_1.id
  workflow_id     = resource.remscontent_workflow.workflow_1.id
  form_id         = resource.remscontent_form.form_1.id
  localizations = {
    title   = "The title for catalogue item 1 - edit01"
    infourl = "url for catalogue item 1"
  }
  categories = [
    resource.remscontent_category.category_1.id,
    resource.remscontent_category.category_2.id,
  ]
}




