terraform {
  required_providers {
    remscontent = {
      source = "registry.terraform.io/umccr/remscontent"
    }
  }
}

provider "remscontent" {

}

data "remscontent_organization" "example" {
  id = "Collaborative Centre for Genomic Cancer Medicine"
}


data "remscontent_license" "example_license" {
  title_en = "Example License"
}


output "edu_coffees" {
  value = data.remscontent_license.example_license.id
}
