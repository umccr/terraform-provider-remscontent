# Simple Example — Creating Licences

This example demonstrates how to create **Licences** using the
`remscontent` Terraform provider, including both inline text and
file attachment licence types.

## Prerequisites

The following objects must already exist in REMS:

- Organisation: `Collaborative Centre for Genomic Cancer Medicine`
- Attachment file: `./license-en.txt`


## Resources Created

| Name | Type | Notes |
|------|------|-------|
| `example_license` | `text` | Inline text licence, active |
| `example_attachment_license` | `attachment` | File-based licence, active |
| `license_202603` | `attachment` | File-based licence, archived |
