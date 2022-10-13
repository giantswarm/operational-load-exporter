module github.com/giantswarm/operational-load-exporter

go 1.17

require (
	github.com/google/go-github v17.0.0+incompatible
	github.com/opsgenie/opsgenie-go-sdk-v2 v1.2.13
	github.com/prometheus/client_golang v1.12.2
	golang.org/x/oauth2 v0.0.0-20220223155221-ee480838109b
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.1 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)

replace (
    golang/golang.org/x/text v0.3.7 => golang/golang.org/x/text v0.3.8
)
