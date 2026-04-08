terraform {
  required_providers {
    remscontent = {
      source = "registry.terraform.io/umccr/remscontent"
    }
  }
}

provider "remscontent" {}

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
