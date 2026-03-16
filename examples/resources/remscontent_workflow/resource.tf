terraform {
  required_providers {
    remscontent = {
      source = "registry.terraform.io/umccr/remscontent"
    }
  }
}

provider "remscontent" {}

variable "email" {}

data "remscontent_actor" "user_1" {
  email = var.email
}

data "remscontent_license" "example_attachment_license" {
  title = "Test Attachment License 2"
}

resource "remscontent_form" "example_form" {
  #  .... form args
}

resource "remscontent_workflow" "example_1" {
  title              = "Example Workflow 1"
  organization_id    = "Collaborative Centre for Genomic Cancer Medicine"
  type               = "workflow/default"
  licenses           = [data.remscontent_license.example_attachment_license.id]
  forms              = [resource.remscontent_form.example_form.id]
  handlers           = [data.remscontent_actor.user_1.id]
  anonymize_handling = true
  archived           = false
  enabled            = true
}
