# dataset

A module for creating a BigQuery dataset for `tally`.

## Usage

Without any variables the module will create a dataset called `tally` in the
`US` location:

```hcl
provider "google" {
  project = "my-example-project"
}

module "tally_dataset" {
  source = "github.com/ribbybibby/tally//terraform/dataset"
}
```

You can configure the id, name and location of the dataset:

```hcl
module "tally_dataset" {
  source = "github.com/ribbybibby/tally//terraform/dataset"

  dataset_id    = "example"
  friendly_name = "example"
  location      = "EU"
}
```

Refer to [variables.tf](./variables.tf) for the full list of variables.
