terraform {
  required_providers {
    remscontent = {
      source = "registry.terraform.io/umccr/remscontent"
    }
  }
}

provider "remscontent" {}

resource "remscontent_license" "example_license" {
  title           = "Test License"
  organization_id = "Collaborative Centre for Genomic Cancer Medicine"
  type            = "text"
  content         = "license test"
  archived        = false
  enabled         = true
}

resource "remscontent_license" "example_attachment_license" {
  title           = "Test Attachment License 2"
  organization_id = "Collaborative Centre for Genomic Cancer Medicine"
  type            = "attachment"
  path            = "./license-en.txt"
}
