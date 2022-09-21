# gRPC-Web Bridge for Caddy

This module implements a bridge from gRPC-Web clients to gRPC servers. It is similar to Envoy's `envoy.filters.http.grpc_web` filter. **It is EXPERIMENTAL and subject to change.**

To convert gRPC-Web requests to gRPC, simply add the `grpc_web` handler to your HTTP route. It should go before your `reverse_proxy` or any other handler that would expect a gRPC request.

## Installation

A new Caddy server with this module needs to be built to support GRPC-web calls. Build it with:

```
docker build -t <account_name>/caddy-grpc .
```

## Example

Caddyfile:

```
{
	order grpc_web before reverse_proxy
}

:5452 {
	grpc_web
	reverse_proxy h2c://127.0.0.1:50051
}
```

JSON:

```json
{
  "apps": {
    "http": {
      "servers": {
        "srv0": {
          "listen": [":5452"],
          "routes": [
            {
              "handle": [
                {
                  "handler": "grpc_web"
                },
                {
                  "handler": "reverse_proxy",
                  "transport": {
                    "protocol": "http",
                    "versions": ["h2c", "2"]
                  },
                  "upstreams": [
                    {
                      "dial": "127.0.0.1:50051"
                    }
                  ]
                }
              ]
            }
          ]
        }
      }
    }
  }
}
```

You can also specify the `websocket_ping` parameter to an interval value >= 1s for websocket keep-alive pings to be enabled.
