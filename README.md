# operational-load-exporter

`operational-load-exporter` fetches operational data from multiple sources, and presents it to Prometheus for scraping. This is towards presenting engineering operational data over time.

## Features

`operational-load-exporter` supports the following data sources:

- GitHub
- incident.io
- Opsgenie

## Development

The following environment variables need to be set:

- `GITHUB_KEY` - a GitHub token with `repo` scope
- `INCIDENT_IO_KEY` - an incident.io API key
- `OPSGENIE_KEY` - an OpsGenie API key

Each data source is modelled as a separate Prometheus Collector, and then registered with the default handler.
