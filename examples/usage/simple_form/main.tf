terraform {
  required_providers {
    remscontent = {
      source = "registry.terraform.io/umccr/remscontent"
    }
  }
}

variable "email" {}
variable "REMS_ENDPOINT" {}
variable "REMS_API_KEY" {}
variable "REMS_API_USER" {}

provider "remscontent" {
  endpoint = var.REMS_ENDPOINT # DNS name only, no https://
  api_user = var.REMS_API_USER
  api_key  = var.REMS_API_KEY
  language = "en" # Localization language for all resources (e.g. "en", "fi")
}


resource "remscontent_form" "example-001" {
  internal_name   = "example-001"
  external_title  = "Example DEMO"
  organization_id = "Collaborative Centre for Genomic Cancer Medicine"

  fields = [
    {
      "id" : "1"
      "title" : "Title",
      "type" : "text",
      "optional" : false
    },
    # {
    #   "title" : "Email",
    #   "type" : "email"
    # },
    # {
    #   "title" : "Description",
    #   "type" : "texta"
    # },
  ]
}


resource "remscontent_form" "test_form" {
  internal_name   = "Test Form"
  external_title  = "Test Form"
  organization_id = "Collaborative Centre for Genomic Cancer Medicine"

  fields = [
    {
      "id" : "fld1"
      "title" : "Title",
      "type" : "text",
      "optional" : false
    },
    {
      "id" : "fld2"
      "title" : "ph.number",
      "type" : "phone-number",
    },
    {
      "id" : "fld3"
      "title" : "Email",
      "type" : "email"
    },
    {
      "id" : "fld4"
      "title" : "Date",
      "type" : "date"
    },
    {
      "id" : "attaching-5"
      "title" : "attach file?",
      "type" : "option",
      "options" : [
        {
          "key" : "y",
          "label" : "yes"
        },
        {
          "key" : "n",
          "label" : "no"
        }
      ]
    },
    {
      "id" : "fld6",
      "title" : "Attachment",
      "type" : "attachment",
      "visibility" : {
        "visibility_type" : "only-if",
        "field_id" : "attaching-5",
        "has_value" : ["n"]
      }
    },
  ]
}
