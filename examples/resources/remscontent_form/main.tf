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
  internal_name   = "Test Form"
  external_title  = "Test Form"
  organization_id = "Collaborative Centre for Genomic Cancer Medicine"

  fields = [
    {
      "title" : "Title",
      "type" : "text",
      "optional" : false
    },

    {
      "title" : "ph.number",
      "type" : "phone-number",
    },
    {
      "title" : "Email",
      "type" : "email"
    },

    {
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

