terraform {
  required_providers {
    remscontent = {
      source = "registry.terraform.io/umccr/remscontent"
    }
  }
}

variable "email" {}

provider "remscontent" {

}

data "remscontent_actor" "user_1" {
  email = var.email
}

resource "remscontent_workflow" "example_1" {
  title              = "Example Workflow 1"
  organization_id    = "Collaborative Centre for Genomic Cancer Medicine"
  type               = "workflow/default"
  licenses           = [3]
  forms              = [17]
  handlers           = [data.remscontent_actor.user_1.id]
  anonymize_handling = true
  archived           = false
  enabled            = true
}
