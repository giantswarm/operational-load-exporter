# operational-load-exporter

`operational-load-exporter` fetches operational data from multiple sources, and presents it to Prometheus for scraping. This is towards presenting engineering operational data over time.

## Features

`operational-load-exporter` supports the following data sources:

- GitHub
- incident.io

## Development

The following environment variables need to be set:

- GitHub credentials, either:
  - a GitHub App (preferred): `GITHUB_APP_ID`, `GITHUB_APP_INSTALLATION_ID` and `GITHUB_APP_PRIVATE_KEY` (PEM text). All three must be set; setting only some is an error. The exporter mints and refreshes installation tokens itself.
  - a token (fallback, used only when none of the App variables are set): `GITHUB_KEY` - a GitHub token with `repo` scope
- `INCIDENT_IO_KEY` - an incident.io API key

The GitHub App only needs the **Issues: read** repository permission on `giantswarm/giantswarm` (Repository metadata: read is implicit). The exporter only searches issues and labels there.

Each data source is modelled as a separate Prometheus Collector, and then registered with the default handler.
