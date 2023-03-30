# tally

Finds [OpenSSF Scorecard](https://github.com/ossf/scorecard) scores for packages
in a Software Bill of Materials.

⚠️ This tool is currently under active development. There will be breaking changes
and how it works may change significantly as it matures.

## Usage

### Basic

Generate an SBOM in CycloneDX JSON format and then scan it with `tally`.

This uses the [public scorecard API](https://api.securityscorecards.dev/#/) to
fetch the latest score for each repository.

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

### Generate scores

The public API may not have a score for every discovered repository but `tally`
can generate these scores itself when the `-g/--generate` flag is
set.

Scores are generated from the `HEAD` of the repository.

This requires that the `GITHUB_TOKEN` environment variable is set to a valid
token.

```
$ export GITHUB_TOKEN=<token>
$ tally -g bom.json
Generating score for 'github.com/foo/bar' [--------->..] 68/72
```

This may take a while, depending on the number of missing scores.

If you'd like to generate all the scores yourself, you can disable fetching
scores from the API with `--api=false`.

### Cache

To speed up subsequent runs, `tally` will cache scorecard results to a local
database. You can disable the cache with `--cache=false`.

By default, `tally` will ignore results that were cached more than 7 days ago.
This window can be changed with the `--cache-duration` flag:

```
tally --cache-duration=20m bom.json
```

The cache is stored in the user's home cache directory, which is commonly
located in `~/.cache/tally/cache/`. This can be changed with the `--cache-dir`
flag.

### Fail on low scores

The return code will be set to 1 when a score is identified that is less than
or equal to the value of `--fail-on`:

```
$ tally --fail-on 3.5 bom.json
...
Error: found scores <= to 3.50
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
TYPE   PACKAGE                     REPOSITORY                            SCORE
golang cloud.google.com/go/compute github.com/googleapis/google-cloud-go 9.3
```

The `json` output will print the full report in JSON format:

```
$ tally -o json bom.json | jq -r .
{
  "results": [
    {
      "repository": "github.com/googleapis/google-http-java-client",
      "packages" : [
        {
          "type": "maven",
          "name": "com.google.http-client/google-http-client-jackson2"
        }
      ],
      "result": {
        "date": "2023-03-04",
        "repo": {
          "name": "github.com/googleapis/google-http-java-client",
          "commit": "4e889b702b8bbfb082b7a3234569dc173c1c286d"
        },
        "scorecard": {
          "version": "v4.8.0",
          "commit": "c40859202d739b31fd060ac5b30d17326cd74275"
        },
        "score": 7,
        "checks": [
          ...
        ]
      }
    },
    ...
  ]
}
```

### Print all

Not all packages will have a Scorecard score.

By default, `tally` will remove results without a score from the output when
using `-o short` or `-o wide`.

You can include all results, regardless of whether they have a score or not, by
specifying the `-a/--all` flag.

### BOM formats

Specify the format of the target SBOM with the `-f/--format` flag.

The supported SBOM formats are:

- `cyclonedx-json`
- `cyclonedx-xml`
- `syft-json`
