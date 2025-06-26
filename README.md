# Terraform Provider for creating naming conventions

The standesamt provider is a provider for Terraform that allows you to create name strings that can be used as names for resources-  

## Get started with standesamt

ADD REFERENCE TO A DOCUMENTATION?

Also, there is a rich library of [examples](https://github.com/glueckkanja/terraform-provider-standesamt/tree/main/examples) to help you get started.

## Usage Example

The following example shows how to use the `build_name` function to create a name for an Azure resource group:

```hcl
terraform {
  required_providers {
    standesamt = {
      source  = "glueckkanja/standesamt"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
    }
  }
}

provider "standesamt" {
}

provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "example" {
  name     = "example-resources"
  location = "West Europe"
}

```

Further [usage documentation is available on the Terraform website](https://registry.terraform.io/providers/glueckkanja/standesamt/latest/docs).

## Developer Requirements

* [OpenTofu](https://opentofu.org/docs/intro/install/) version 1.8+
* [Go](https://golang.org/doc/install) version 1.23.x (to build the provider plugin)

### On Windows

If you're on Windows you'll also need:

* [Git Bash for Windows](https://git-scm.com/download/win)
* [Make for Windows](http://gnuwin32.sourceforge.net/packages/make.htm)

For *GNU32 Make*, make sure its bin path is added to PATH environment variable.*

For *Git Bash for Windows*, at the step of "Adjusting your PATH environment", please choose "Use Git and optional Unix tools from Windows Command Prompt".*

Or install via [Chocolatey](https://chocolatey.org/install) (`Git Bash for Windows` must be installed per steps above)

```powershell
choco install make golang terraform -y
refreshenv
```

You must run `Developing the Provider` commands in `bash` because `sh` scrips are invoked as part of these.

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.18+ is **required**). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

First clone the repository to: `$GOPATH/src/github.com/glueckkanja/terraform-provider-standesamt`:

```sh
mkdir -p $GOPATH/src/github.com/glueckkanja; cd $GOPATH/src/github.com/glueckkanja
git clone git@github.com:glueckkanja/terraform-provider-standesamt
cd $GOPATH/src/github.com/glueckkanja/terraform-provider-standesamt
```

Once inside the provider directory, you can run `make tools` to install the dependent tooling required to compile the provider.

At this point you can compile the provider by running `make build`, which will build the provider and put the provider binary in the `$GOPATH/bin` directory.

```sh
$ make build
...
$ $GOPATH/bin/terraform-provider-standesamt
...
```

You can also cross-compile if necessary:

```sh
GOOS=windows GOARCH=amd64 make build
```

In order to run the `Unit Tests` for the provider, you can run:

```sh
make test
```

The majority of tests in the provider are `Acceptance Tests` - which provisions real resources in Azure. It's possible to run the entire acceptance test suite by running `make testacc` - however it's likely you'll want to run a subset, which you can do using a prefix, by running:

```sh
make acctests TESTARGS='-run=<nameOfTheTest>' TESTTIMEOUT='60m'
```

* `<nameOfTheTest>` should be self-explanatory as it is the name of the test you want to run. An example could be `TestAccGenericResource_basic`. Since `-run` can be used with regular expressions you can use it to specify multiple tests like in `TestAccGenericResource_` to run all tests that match that expression

The following Environment Variables must be set in your shell prior to running acceptance tests:

* `ARM_CLIENT_ID`
* `ARM_CLIENT_SECRET`
* `ARM_SUBSCRIPTION_ID`
* `ARM_TENANT_ID`
* `ARM_ENVIRONMENT`
* `ARM_METADATA_HOST`
* `ARM_TEST_LOCATION`
* `ARM_TEST_LOCATION_ALT`
* `ARM_TEST_LOCATION_ALT2`

**Note:** Acceptance tests create real resources in Azure which often cost money to run.

## Generating Documentation

We use [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs) to automatically generate documentation for the provider.
Please ensure that the `MarkdownDescription` field is set in the schema for each function.

To generate the documentation run either:

```sh
make docs
```

or...

```sh
go generate ./...
```

### Guides

Guides should be stored in the `templates/guides` directory. They will be inclided in the documentation and copied to the `docs` directory by the `tfplugindocs` tool.

### Examples

The `examples/functions` directory contains examples for each function. The examples are used to generate the documentation for each resource and data source. The examples are written in HCL and must be called `function.tf`. These are then embedded into the documentation and are used to generate the `Example` section.

---

## Developer: Using the locally compiled Provider binary

After successfully compiling the standesamt Provider, you must [instruct OpenTofu to use your locally compiled provider binary](https://www.terraform.io/docs/commands/cli-config.html#development-overrides-for-provider-developers) instead of the official binary from the OpenTofu Registry.

For example, add the following to `~/.terraformrc` for a provider binary located in `/home/developer/go/bin`:

```
provider_installation {

  # Use /home/developer/go/bin as an overridden package directory
  # for the Azure/azapi provider. This disables the version and checksum
  # verifications for this provider and forces Terraform to look for the
  # azapi provider plugin in the given directory.
  dev_overrides {
    "glueckkanja/standesamt" = "/home/developer/go/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

## Testing with OpenTofu

There are some extra steps to test with OpenTofu. 

You can read more [here](https://github.com/orgs/opentofu/discussions/975)

Basically you need to set the following environment variables:

```shell
TF_ACC_TERRAFORM_PATH="/path/to/opentofu"
TF_ACC_PROVIDER_NAMESPACE="hashicorp"
TF_ACC_PROVIDER_HOST="registry.opentofu.org"
```
