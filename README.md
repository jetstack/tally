# tally

Finds [OpenSSF Scorecard](https://github.com/ossf/scorecard) scores for packages
in a Software Bill of Materials.

## Usage

Generate an SBOM and then scan it with `tally`:

```
$ syft prom/prometheus -o cyclonedx-json > bom.json
$ tally -p my-gcp-project-id bom.json
Found 150 supported packages in BOM
Fetching repository information from deps.dev dataset...
Fetching scores from OpenSSF scorecard dataset...
REPOSITORY                                         SCORE
github.com/googleapis/google-cloud-go              9.3
github.com/imdario/mergo                           9.1
github.com/googleapis/gax-go                       8.9
github.com/kubernetes/api                          8.2
github.com/azure/go-autorest                       8.0
github.com/googleapis/go-genproto                  7.9
...
```

You can pipe the BOM directly to `tally` as well:

```
$ syft prom/prometheus -o cyclonedx-json | tally -p my-gcp-project -
```

Not all the repositories in the deps.dev data set have scorecard scores. You can
generate missing scores yourself by setting the `-g/--generate` flag. This
requires that the `GITHUB_TOKEN` environment variable is set to a valid token.

```
$ tally -p my-gcp-project-id -g bom.json
Found 150 supported packages in BOM
Fetching repository information from deps.dev dataset...
Fetching scores from OpenSSF scorecard dataset...
Generating missing scores...
```

## How it works

This tool queries the public BigQuery [deps.dev](https://deps.dev/data) and
[OpenSSF](https://github.com/ossf/scorecard#public-data) datasets in order to
associate components with Scorecard projects.

As such, it only supports the package types supported by [deps.dev](https://deps.dev/):

- NPM
- Go
- Maven
- PyPI
- Cargo

The datasets are public but you must still provide a valid Google Cloud project
that you have access to with the `--project-id/-p` flag.
