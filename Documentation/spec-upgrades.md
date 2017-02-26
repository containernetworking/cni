# CNI Plugin Versions, Compatibility, and Writing Plugins

## Specification Changes between CNI Specification Versions 0.2.0 and
0.3.0

Plugins which conform to the CNI 0.3.0 specification may elect to
return more expressive IPAM JSON that is not compatible with earlier
specification versions.  This requires updates to plugins that wish to
conform to the 0.3.0 specification. It **does not** require updates to
plugins that wish to only implement the 0.2.0 and earlier
specifications.

## Multi-version Compatible Plugins

Plugins can also choose to support multiple CNI versions. The plugin is
expected to return IPAM output conforming to the CNI specification
version passed in the plugin's configuration on stdin in the
`cniVersion` key. If that key is not present, then CNI specification
version 0.2.0 should be assumed and 0.2.0 conforming JSON returned.
Plugins should return all CNI specification versions they are
compatible with when given the `VERSION` command, so runtimes can
determine each plugins capabilities.

For example, if the plugin advertises only CNI specification version
0.2.0 support and receives a `cniVersion` key of "0.3.0" the plugin
must return an error. If the plugin advertises CNI specification
support for 0.2.0 and 0.3.0, and receives `cniVersion` key of "0.2.0",
the plugin must return IPAM JSON conforming to the CNI specification
version 0.2.0.

## libcni API and Go Types Changes in 0.3.0

With the 0.5.0 CNI reference plugins and libcni release a number of
important changes were made that affect both runtimes and plugin
authors. These changes make it easier to enhance the libcni and plugin
interfaces going forward. The largest changes are to the `types`
package, which now contains multiple versions of libcni types. The
`current` package contains the latest types, while types compatible
with previous libcni releases will be placed into versioned packages
like `types020` (representing the libcni types that conform to the CNI
specification version 0.2.0).

The `Result` type is now an interface instead of a plain struct. The
interface has new functions to convert the Result structure between
different versions of the libcni types. Plugins and consumers of libcni
should generally be written to use the Go types of the latest libcni
release, and convert between the latest libcni types and the CNI
specification type they are passed in the `cniVersion` key of plugin
configuration.

### Plugins

For example, say your plugins supports both CNI specification version
0.2.0 and 0.3.0. Your plugin receives configuration with a `cniVersion`
key of "0.2.0".  This means your plugin should return an IPAM structure
conforming to the CNI specification version 0.2.0. The easiest way to
code your plugin is to always use the most current libcni Result type
(from the `current` package) and then immediately before exiting,
convert that result to the requested CNI specification version result
(eg, `types020`) and print it to stdout.

```
import "github.com/containernetworking/cni/pkg/types"
import "github.com/containernetworking/cni/pkg/types/current"

result := &current.Result{}
<<< populate result here >>>
return types.PrintResult(result, << CNI version from stdin net
config>>)
```

or if your plugin internally runs IPAM and needs to process the result:

```
import "github.com/containernetworking/cni/pkg/types"
import "github.com/containernetworking/cni/pkg/types/current"

ipamResult, err := ipam.ExecAdd(n.IPAM.Type, args.StdinData)
result, err := current.NewResultFromResult(ipamResult)
<<< manipulate result here >>>
return types.PrintResult(result, << CNI version from stdin net
config>>)
```

### Runtimes

Since libcni functions like AddNetwork() now return a Result interface
rather than a plain structure, only a single additional step is
required to convert that object to a structure you can manipulate
internally. Runtimes should code to the most current Go types provided
by the libcni they vendor into their sources, and can convert from
whatever IPAM result version the plugin provides to the most current
version like so:

```
resultInterface, err := cninet.AddNetwork(netconf, rt)
<<< resultInterface could wrap an object of any CNI specification
version >>>
realResult := current.NewResultFromResult(resultInterface)
<<< realResult is a struct, not an interface >>>
```
