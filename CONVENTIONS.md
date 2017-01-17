# Extension conventions                                                        
                                                                                 
The Container Network Interface (CNI) currently provides two methods for passing additional information to plugins without needing to update the spec. These are the `CNI_ARGS` environment variable and the `args` field in the JSON config.
                                                                                 
Establishing these conventions allows plugins to work across multiple runtimes. This helps both plugins and the runtimes.

## Plugins
* Plugin authors should aim to support these conventions where it makes sense for their plugin. This means they are more likely to "just work" with a wider range of runtimes.

## Runtimes
* Runtime authors should follow these conventions if they want to pass additional information to plugins. This will allow the extra information to be consumed by the widest range of plugins.

# Current conventions
Additional conventions can be created by creating PRs which modify this document.
                                                      
## CNI_ARGS
CNI_ARGS formed part of the original CNI spec and have been present since the initial release.
> `CNI_ARGS`: Extra arguments passed in by the user at invocation time. Alphanumeric key-value pairs separated by semicolons; for example, "FOO=BAR;ABC=123"

| Field  | Purpose| Spec and Example | Runtime implementations | Plugin Implementations |
| ------ | ------ | ------             | ------  | ------                  | ------                 |  
| IP     | Request a specific IP from IPAM plugins | IP=192.168.10.4 | *rkt* supports passing additional arguments to plugins and the [documentation](https://coreos.com/rkt/docs/latest/networking/overriding-defaults.html) suggests IP can be used. | host-local (since version v0.2.0) supports the field for IPv4 only - [documentation](https://github.com/containernetworking/cni/blob/master/Documentation/host-local.md#supported-arguments).|
                                                                                                                  
## "args" in network config
`args` in [network config](https://github.com/containernetworking/cni/blob/master/SPEC.md#network-configuration) were introduced as an optional field into the `0.1.0` CNI spec. The first CNI code release that it appeared in was `v0.4.0`. 
> args (dictionary): Optional additional arguments provided by the container runtime. For example a dictionary of labels could be passed to CNI plugins by adding them to a labels field under args.

`args` provide a way of providing more structured data than the flat strings that CNI_ARGS can support.

The conventions documented here are all namepaced under `cni` so they don't conflict with any existing `args`.

```json
{  
   "cniVersion":"0.2.0",
   "name":"net",
   "args":{  
      "cni":{  
         "example":"value"
      }
   },
   <REST OF CNI CONFIG HERE>
   "ipam":{  
     <IPAM CONFIG HERE>
   }
}
```

| Area  | Purpose| Spec and Example | Runtime implementations | Plugin Implementations |
| ------ | ------ | ------             | ------  | ------                  | ------                 |  
| labels | Pass`key=value` labels to plugins | <pre>"labels" : [<br />  { "key" : "app", "value" : "myapp" },<br />  { "key" : "env", "value" : "prod" }<br />] </pre> | none | none |
| port mappings | Pass mapping from ports on the host to ports in the container network namespace. | <pre>"port_mappings" : [<br />  { "host_port": 8080, "container_port": 80, "protocol": "tcp" },<br />  { "host_port": 8000, "container_port": 8001, "protocol": "udp" }<br />] </pre> | none | none |
