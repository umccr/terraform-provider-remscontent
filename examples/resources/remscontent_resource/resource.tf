terraform {
  required_providers {
    remscontent = {
      source = "registry.terraform.io/umccr/remscontent"
    }
  }
}

provider "remscontent" {

}


resource "remscontent_resource" "resource_1" {
  organization_id = "Collaborative Centre for Genomic Cancer Medicine"
  resid           = "uri-resource-1"
  licenses        = [3]
  archived        = false
  enabled         = true
}

