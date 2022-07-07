# tally

Finds [OpenSSF Scorecard](https://github.com/ossf/scorecard) scores for packages
in a Software Bill of Materials.

## Usage

Generate an SBOM and then scan it with `tally`:

```
$ syft prom/prometheus -o cyclonedx-json > bom.json
$ tally -p my-gcp-project-id bom.json
Found 281 supported packages in BOM
Retrieving scores from BigQuery...
SYSTEM PACKAGE                                                         VERSION                              REPOSITORY                                         SCORE DATE
GO     cloud.google.com/go/compute                                     v1.3.0                               github.com/googleapis/google-cloud-go              9.3   2022-06-27
GO     github.com/imdario/mergo                                        v0.3.12                              github.com/imdario/mergo                           9.1   2022-06-27
GO     github.com/googleapis/gax-go/v2                                 v2.1.1                               github.com/googleapis/gax-go                       8.9   2022-06-27
GO     k8s.io/api                                                      v0.22.7                              github.com/kubernetes/api                          8.2   2022-06-27
...
```

You can pipe the BOM directly to `tally` as well:
```
$ syft prom/prometheus -o cyclonedx-json | tally -p my-gcp-project -
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
