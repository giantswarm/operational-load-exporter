# operational-load-exporter

A Helm chart for operational-load-exporter

**Homepage:** <https://github.com/giantswarm/operational-load-exporter>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| image.registry | string | `"gsoci.azurecr.io"` |  |
| image.name | string | `"giantswarm/operational-load-exporter"` |  |
| image.tag | string | `""` |  |
| cortex.url | string | `""` |  |
| cortex.username | string | `""` |  |
| cortex.password | string | `""` |  |
| github.key | string | `""` | Legacy personal access token (already base64, written straight into the Secret's `data:`). Used only when the GitHub App values below are not all set. |
| github.app.id | string | `""` | The App ID. Accepts a string or a number. |
| github.app.installationID | string | `""` | The installation ID of the App on the giantswarm organization. Accepts a string or a number. |
| github.app.privateKey | string | `""` | The App private key, PEM text (not base64; the chart encodes it). |
| incidentio.key | string | `""` |  |
| resources.main.requests.cpu | string | `"2m"` |  |
| resources.main.requests.memory | string | `"20Mi"` |  |
| resources.main.limits.cpu | string | `"50m"` |  |
| resources.main.limits.memory | string | `"200Mi"` |  |
| resources.grafanaAgent.requests.cpu | string | `"8m"` |  |
| resources.grafanaAgent.requests.memory | string | `"50Mi"` |  |
| resources.grafanaAgent.limits.cpu | string | `"50m"` |  |
| resources.grafanaAgent.limits.memory | string | `"200Mi"` |  |
