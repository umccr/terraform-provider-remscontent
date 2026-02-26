terraform {
  required_providers {
    remscontent = {
      source = "registry.terraform.io/umccr/remscontent"
    }
  }
}

provider "remscontent" {

}


resource "remscontent_form" "test_form" {
  title           = "Test Form"
  organization_id = "Collaborative Centre for Genomic Cancer Medicine"
}

# output "edu_coffees" {
#   value = remscontent_license.example_license.id
# }
