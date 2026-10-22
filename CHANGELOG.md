# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Authenticate to GitHub as a GitHub App (`github.app.id`, `github.app.installationID`, `github.app.privateKey`) instead of a personal access token. `GITHUB_KEY` still works as a fallback when no App values are set.
- `operations_github_last_complete_refresh_timestamp_seconds`, the time of the last refresh in which every count succeeded, to alert on stale issue counts. It appears after the first complete refresh, so pair an age alert with `absent()`.

### Changed

- All GitHub environment variables are now optional in the chart, so either authentication mode works.

### Fixed

- GitHub issue counts no longer drop out on the search rate limit (30 requests per minute). A background refresh starts every 5 minutes and sends at most 24 searches per minute; scrapes serve the last counts. A count that fails keeps its last value, a refresh that is still rate limited after one retry stops and is retried after one minute, as is a refresh that cannot list the team labels or fails right after a restart. Team label names are quoted in the search query. Every GitHub request has a 30s timeout. Secondary rate limits (429, or 403 with Retry-After) are retried too.
- Startup no longer exits when GitHub is rate limited: the team labels come from the labels API instead of the search API, and are listed on each refresh.

## [0.5.0] - 2026-01-05

### Fixed

- Updated dependencies.
- Fixed "Error return value of resp.Body.Close is not checked" where appropriate.

### Removed

- Remove PodSecurityPolicy resources from helm chart as they are being deprecated in favor of Pod Security Standards.
- Remove OpsGenie collector

## [0.4.4] - 2024-03-15

- Fix template change made in 0.4.3

## [0.4.3] - 2024-03-15

- Make resource requests and limits configurable.
- Use container image registry `gsoci.azurecr.io`


## [0.4.2] - 2024-01-11

## [0.4.1] - 2023-12-05

### Fixed

- Disable PSP if PSS is in enforced mode.

## [0.4.0] - 2023-12-04

### Added

- Add icon.

### Changed

- Change the registries to new ones.
- Address PSS errors.

## [0.3.0] - 2022-10-13

## [0.2.1] - 2022-08-10

## [0.2.0] - 2022-08-10

## [0.1.4] - 2022-07-27

## [0.1.3] - 2022-07-22

## [0.1.2] - 2022-07-20

- changed: first release!

[Unreleased]: https://github.com/giantswarm/operational-load-exporter/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/giantswarm/operational-load-exporter/compare/v0.4.4...v0.5.0
[0.4.4]: https://github.com/giantswarm/operational-load-exporter/compare/v0.4.3...v0.4.4
[0.4.3]: https://github.com/giantswarm/operational-load-exporter/compare/v0.4.2...v0.4.3
[0.4.2]: https://github.com/giantswarm/operational-load-exporter/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/giantswarm/operational-load-exporter/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/giantswarm/operational-load-exporter/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/giantswarm/operational-load-exporter/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/giantswarm/operational-load-exporter/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/giantswarm/operational-load-exporter/compare/v0.1.4...v0.2.0
[0.1.4]: https://github.com/giantswarm/operational-load-exporter/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/giantswarm/operational-load-exporter/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/giantswarm/operational-load-exporter/releases/tag/v0.1.2
