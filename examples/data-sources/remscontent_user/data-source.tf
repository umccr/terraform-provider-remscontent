terraform {
  required_providers {
    remscontent = {
      source = "registry.terraform.io/umccr/remscontent"
    }
  }
}

provider "remscontent" {

}

data "remscontent_user" "abcd" {
  email = "john@doe.com"
}



output "out" {
  value = data.remscontent_user.abcd.id
}
