# Terraform Provider REMS Content

A Terraform provider for managing the content of [REMS](https://github.com/CSCfi/rems) (Resource Entitlement Management System) instances. This provider does **not** install REMS itself — it manages the content of an existing REMS instance: forms, workflows, licences, catalogue items, categories, and more.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (for building from source)

## Using the Provider

Configure the provider with your REMS instance details. All attributes can alternatively be supplied via environment variables (`REMS_ENDPOINT`, `REMS_API_USER`, `REMS_API_KEY`, `REMS_LANGUAGE`).

```terraform
terraform {
  required_providers {
    remscontent = {
      source = "registry.terraform.io/umccr/remscontent"
    }
  }
}

provider "remscontent" {
  endpoint = "rems.example.org" # DNS name only, no https://
  api_user = "admin@example.org"
  api_key  = "my-secret-api-key"
  language = "en" # Localization language for all resources (e.g. "en", "fi")
}
```

See the [provider documentation](docs/index.md) for full configuration reference.

### Resources

| Resource | Documentation |
|---|---|
| `remscontent_form` | [docs/resources/form.md](docs/resources/form.md) |
| `remscontent_workflow` | [docs/resources/workflow.md](docs/resources/workflow.md) |
| `remscontent_license` | [docs/resources/license.md](docs/resources/license.md) |
| `remscontent_resource` | [docs/resources/resource.md](docs/resources/resource.md) |
| `remscontent_catalogue_item` | [docs/resources/catalogue_item.md](docs/resources/catalogue_item.md) |
| `remscontent_category` | [docs/resources/category.md](docs/resources/category.md) |
| `remscontent_blacklist` | [docs/resources/blacklist.md](docs/resources/blacklist.md) |

### Data Sources

| Data Source | Documentation |
|---|---|
| `remscontent_actor` | [docs/data-sources/actor.md](docs/data-sources/actor.md) |
| `remscontent_organization` | [docs/data-sources/organization.md](docs/data-sources/organization.md) |
| `remscontent_form` | [docs/data-sources/form.md](docs/data-sources/form.md) |
| `remscontent_workflow` | [docs/data-sources/workflow.md](docs/data-sources/workflow.md) |
| `remscontent_license` | [docs/data-sources/license.md](docs/data-sources/license.md) |
| `remscontent_catalogue_item` | [docs/data-sources/catalogue_item.md](docs/data-sources/catalogue_item.md) |
| `remscontent_category` | [docs/data-sources/category.md](docs/data-sources/category.md) |
| `remscontent_blacklist_user` | [docs/data-sources/blacklist_user.md](docs/data-sources/blacklist_user.md) |
| `remscontent_resource` | [docs/data-sources/resource.md](docs/data-sources/resource.md) |

## Usage Examples

End-to-end runnable examples are under [`examples/usage/`](examples/usage/):

| Example | Description |
|---|---|
| [`simple_form`](examples/usage/simple_form/) | Create REMS forms, including conditional field visibility |
| [`simple_license`](examples/usage/simple_license/) | Create licences (inline text and file attachment types) |
| [`simple_example`](examples/usage/simple_example/) | Full catalogue item setup — wires together organisation, licence, workflow, form, resource, category, and blacklist |

Each example directory contains a `README.md` with further details and a `main.tf` you can run directly.

## Building the Provider

```shell
git clone https://github.com/umccr/terraform-provider-remscontent
cd terraform-provider-remscontent
go install
```

This places the provider binary in `$GOPATH/bin`.

## Developing the Provider

To use a locally built provider before it is published to a registry, add a dev override to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/umccr/remscontent" = "/path/to/your/GOPATH/bin"
  }

  # All other providers install normally from their registries.
  direct {}
}
```

To regenerate documentation from source annotations:

```shell
make generate
```

To run the test suite:

```shell
make test
```

### Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.
