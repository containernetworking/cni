module github.com/containernetworking/cni/plugins/debug

go 1.21
toolchain go1.23.2

require (
	github.com/containernetworking/cni v1.2.3
	github.com/containernetworking/plugins v1.6.0
)

require (
	github.com/vishvananda/netns v0.0.4 // indirect
	golang.org/x/sys v0.26.0 // indirect
)

replace github.com/containernetworking/cni => ../..
