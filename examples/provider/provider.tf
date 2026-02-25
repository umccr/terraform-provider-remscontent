terraform {
  required_providers {
    remscontent = {
      source = "registry.terraform.io/umccr/remscontent"
    }
  }
}

provider "remscontent" {

  # should organization be set at the provider level rather than included in every resource??
  # organization_id = "Collaborative Centre for Genomic Cancer Medicine"
}

resource "remscontent_form" "xyz_form" {
  organization_id = "Collaborative Centre for Genomic Cancer Medicine"
  title           = "Access to XYZ data"

  fields = [
    provider::remscontent::form_field_header("hdr_applicant", { en : "Applicant", fi : "Blah" }),
    provider::remscontent::form_field_header("hdr_purpose", { en : "Purpose" }),
    // provider::remscontent::form_field_label({ en : "Yes" }),
  ]
}
