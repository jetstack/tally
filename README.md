# tally

Finds [OpenSSF Scorecard](https://github.com/ossf/scorecard) scores for packages
in a Software Bill of Materials.

This tool is currently under active development. There will be breaking changes
and how it works may change significantly as it matures.

## How it works

This tool queries the public BigQuery [deps.dev](https://deps.dev/data) and
[OpenSSF](https://github.com/ossf/scorecard#public-data) datasets in order to
associate components with Scorecard projects.

As such, it only supports the package types supported by
[deps.dev](https://deps.dev/):

- NPM
- Go
- Maven
- PyPI
- Cargo

## Usage

### Basic

Generate an SBOM in CycloneDX JSON format and then scan it with `tally`.

The datasets are public but you must still provide a valid Google Cloud project
that you have access to with the `--project-id/-p` flag.

```
$ syft prom/prometheus -o cyclonedx-json > bom.json
$ tally -p my-gcp-project-id bom.json
Found 150 supported packages in BOM
Fetching repository information from deps.dev dataset...
Fetching scores from OpenSSF scorecard dataset...
REPOSITORY                            SCORE
github.com/googleapis/google-cloud-go 9.3
github.com/imdario/mergo              9.1
github.com/googleapis/gax-go          8.9
github.com/kubernetes/api             8.2
github.com/azure/go-autorest          8.0
github.com/googleapis/go-genproto     7.9
...
```

You could also pipe the BOM directly to `tally`:

```
$ syft prom/prometheus -o cyclonedx-json | tally -p my-gcp-project -
```

### Generate missing scores

Not all the repositories in the deps.dev dataset have corresponding scores in
the scorecard dataset.

Tally will generate missing scores itself when the `-g/--generate` flag is set.

This requires that the `GITHUB_TOKEN` environment variable is set to a valid
token.

```
$ export GITHUB_TOKEN=<token>
$ tally -p my-gcp-project-id -g bom.json
Found 150 supported packages in BOM
Fetching repository information from deps.dev dataset...
Fetching scores from OpenSSF scorecard dataset...
Generating missing scores...
```

### Store generated scores in BigQuery

Generation can take a while, depending on the number of missing scores. To speed
up subsequent invocations, `tally` supports saving scores to a BigQuery dataset.

You can use [this terraform module](./terraform/dataset) to create the dataset.
Or use it as a reference if you're going to create the dataset another way.

Once it's set up, you can use it with the `-d/--dataset` flag:

```
$ tally -p my-gcp-project-id -g -d 'tally' bom.json
```

When `-d/--dataset` is set without `-g/--generate`, `tally` will query the
dataset for existing scores but won't generate any new ones.

### Output formats

The `-o/--output` flag can be used to modify the output format.

By default, `tally` will output each unique repository and its score:

```
REPOSITORY                            SCORE
github.com/googleapis/google-cloud-go 9.3
```

The `wide` output format will print additional package information, as well as
the date the score was generated:

```
SYSTEM PACKAGE                     VERSION REPOSITORY                            SCORE DATE
go     cloud.google.com/go/compute v1.3.0  github.com/googleapis/google-cloud-go 9.3   2022-06-27
```

The `json` output will print the full output in JSON format:

```
[
  {
    "type": "go",
    "name": "cloud.google.com/go/compute",
    "version": "v1.3.0",
    "repository": "github.com/googleapis/google-cloud-go",
    "score": 9.3,
    "date": "2022-06-27"
  },
  ...
]
```

### Print all packages

Not all packages will have a Scorecard score.

This is typically because the package's repository is either:

- Not in the deps.dev dataset
- Not in the Scorecard dataset
- Not hosted on Github

By default, `tally` will remove packages without a score from the output.

You can include all packages, regardless of whether they have a score or not, by
specifying the `-a/--all` flag.

### BOM formats

Specify the format of the target SBOM with the `-f/--format` flag.

The supported SBOM formats are:

- `cyclonedx-json`
- `cyclonedx-xml`
- `syft-json`
