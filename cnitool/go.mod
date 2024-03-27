module github.com/containernetworking/cni/cnitool

go 1.20

require (
	github.com/containernetworking/cni v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.8.0
)

require (
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)

replace github.com/containernetworking/cni => ../
