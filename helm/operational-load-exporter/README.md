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
| github.key | string | `""` |  |
| incidentio.key | string | `""` |  |
| resources.main.requests.cpu | string | `"2m"` |  |
| resources.main.requests.memory | string | `"20Mi"` |  |
| resources.main.limits.cpu | string | `"50m"` |  |
| resources.main.limits.memory | string | `"200Mi"` |  |
| resources.grafanaAgent.requests.cpu | string | `"8m"` |  |
| resources.grafanaAgent.requests.memory | string | `"50Mi"` |  |
| resources.grafanaAgent.limits.cpu | string | `"50m"` |  |
| resources.grafanaAgent.limits.memory | string | `"200Mi"` |  |
