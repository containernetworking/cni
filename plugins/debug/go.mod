module github.com/containernetworking/cni/plugins/debug

go 1.21

require (
	github.com/containernetworking/cni v1.1.2
	github.com/containernetworking/plugins v1.4.0
)

require (
	github.com/vishvananda/netns v0.0.4 // indirect
	golang.org/x/sys v0.15.0 // indirect
)

replace github.com/containernetworking/cni => ../..
