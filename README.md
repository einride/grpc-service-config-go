protoc-gen-go-grpc-service-config
=================================

[gRPC service config](https://github.com/grpc/grpc/blob/master/doc/service_config.md) without service discovery, by bundling the service config with the generated code.

How to
======

Step 1: Add a service config JSON file to your package
------------------------------------------------------

There is assumed to be one JSON file per package, named according to: `<package>_grpc_service_config.json`.

For example `src/example/v1/example_grpc_service_config.json`:

```json
{
  "methodConfig": [
    {
      "name": [{ "service": "example.v1.ExampleService" }],
      "timeout": "10s",
      "retryPolicy": {
        "initialBackoff": "0.200s",
        "maxBackoff": "60s",
        "maxAttempts": 5,
        "backoffMultiplier": 1.3,
        "retryableStatusCodes": ["UNAVAILABLE"]
      }
    }
  ]
}
```

Step 2: Run the protoc plugin
-----------------------------

Use the required `path` option to tell the generator where to load JSON files from.

Use the optional `validate` option to validate that every package has a service config JSON, and that the format is valid.

```bash
protoc
  -I src \
  --go_out=gen/go \
  --go-grpc_out=gen/go \
  --go-grpc-service-config_out=gen/go \
  --go-grpc-service-config_opt=path=src \
  --go-grpc-service-config_opt=validate=true
```

Your generated code output will now have a Go file corresponding to every service config JSON file.

For example `gen/go/example/v1/example_grpc_service_config.json.go`:

```go
package examplev1

// ServiceConfig is the service config for all services in the package.
// Source: example_grpc_service_config.json.
const ServiceConfig = `{
  "methodConfig": [
    {
      "name": [{ "service": "example.v1.ExampleService" }],
      "timeout": "10s",
      "retryPolicy": {
        "initialBackoff": "0.200s",
        "maxBackoff": "60s",
        "maxAttempts": 5,
        "backoffMultiplier": 1.3,
        "retryableStatusCodes": ["UNAVAILABLE"]
      }
    }
  ]
}
`
```

Step 3: Use your bundled service config when dialing
----------------------------------------------------

```go
conn, err := grpc.DialContext(
	ctx,
	"example.com:443",
	grpc.WithDefaultServiceConfig(examplev1.ServiceConfig),
)
```
