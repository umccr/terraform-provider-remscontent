terraform {
  required_providers {
    remscontent = {
      source = "registry.terraform.io/umccr/remscontent"
    }
  }
}

data "remscontent_license" "example_attachment_license" {
  title = "Test Attachment License 2"
}

resource "remscontent_resource" "resource_1" {
  organization_id = "Collaborative Centre for Genomic Cancer Medicine"
  resource_ext_id = "uri-resource-1"
  licenses        = [data.remscontent_license.example_attachment_license.id]
  archived        = false
  enabled         = true
}
