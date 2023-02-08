# tally

Finds [OpenSSF Scorecard](https://github.com/ossf/scorecard) scores for packages
in a Software Bill of Materials.

⚠️ This tool is currently under active development. There will be breaking changes
and how it works may change significantly as it matures.

## Package types

It currently supports the following package types:

- NPM
- Go
- Maven
- PyPI
- Cargo

## Usage

### Basic

Generate an SBOM in CycloneDX JSON format and then scan it with `tally`.

```
$ syft prom/prometheus -o cyclonedx-json > bom.json
$ tally bom.json
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
$ syft prom/prometheus -o cyclonedx-json | tally -
```

### Generate missing scores

Tally may not have scores in its database for every discovered repository but
it can generate these missing scores itself when the `-g/--generate` flag is
set.

This requires that the `GITHUB_TOKEN` environment variable is set to a valid
token.

```
$ export GITHUB_TOKEN=<token>
$ tally -g bom.json
Generating score for 'github.com/foo/bar' [--------->..] 68/72
```

This may take a while, depending on the number of missing scores.

### Fail on low scores

The return code will be set to 1 when a score is identified that is less than
or equal to the value of `--fail-on`:

```
$ tally --fail-on 3.5 bom.json
...
error: found scores <= to 3.50
exit status 1
```

This will not consider packages `tally` has not been able to retrieve a score
for.

### Output formats

The `-o/--output` flag can be used to modify the output format.

By default, `tally` will output each unique repository and its score:

```
REPOSITORY                            SCORE
github.com/googleapis/google-cloud-go 9.3
```

The `wide` output format will print additional package information:
```
SYSTEM PACKAGE                     VERSION REPOSITORY                            SCORE
GO     cloud.google.com/go/compute v1.3.0  github.com/googleapis/google-cloud-go 9.3
```

The `json` output will print the full output in JSON format, including the
individual check scores:

```
$ tally -o json bom.json | jq -r .
[
  {
    "repository": "github.com/googleapis/google-http-java-client",
    "packages" : [
      {
        "system": "MAVEN",
        "name": "com.google.http-client:google-http-client-jackson2"
      }
    ],
    "score": {
      "score": 7.9,
      "checks": {
        "Binary-Artifacts": 10,
        "Branch-Protection": 8,
        "CII-Best-Practices": 0,
        "Code-Review": 10,
        "Dangerous-Workflow": 10,
        "Dependency-Update-Tool": 10,
        "Fuzzing": 0,
        "License": 10,
        "Maintained": 10,
        "Packaging": 0,
        "Pinned-Dependencies": 9,
        "SAST": 0,
        "Security-Policy": 10,
        "Signed-Releases": 0,
        "Token-Permissions": 0,
        "Vulnerabilities": 10
      }
    }
  },
  ...
]
```

### Print all

Not all packages will have a Scorecard score.

By default, `tally` will remove results without a score from the output.

You can include all results, regardless of whether they have a score or not, by
specifying the `-a/--all` flag.

### BOM formats

Specify the format of the target SBOM with the `-f/--format` flag.

The supported SBOM formats are:

- `cyclonedx-json`
- `cyclonedx-xml`
- `syft-json`

## Database

When `tally` runs for the first time, it pulls down a database from
`ghcr.io/jetstack/tally/db:v1` and caches it locally, typically in
`~/.cache/tally/db`.

It uses the data in this database to associate Scorecard scores with packages.

Every time `tally` runs it will check for a new version of the database and pull
it down if it finds one.
