package grpcweb

import (
	"net/http"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
)

func init() {
	caddy.RegisterModule(Handler{})
	httpcaddyfile.RegisterHandlerDirective("grpc_web", parseCaddyfile)
}

// Handler is an HTTP handler that bridges gRPC-Web <--> gRPC requests.
// This module is EXPERIMENTAL and subject to change.
type Handler struct {
	// Enable WebSocket keep-alive pinging. Default: 0 (no pinging).
	// Minimum to enable: 1s.
	WebSocketPing caddy.Duration `json:"websocket_ping,omitempty"`
}

// CaddyModule returns the Caddy module information.
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.grpc_web",
		New: func() caddy.Module { return new(Handler) },
	}
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// The grpcweb package does not have the most efficient API for our use case.
	// Here we've extracted a couple of its methods, IsGrpcWebRequest() and
	// IsGrpcWebSocketRequest(), to check if the request is even relevant. Only
	// then do we create a WrappedGrpcServer handler by wrapping our "next" handler,
	// which then basically performs the same checks. Not only do we allocate a new
	// WrappedGrpcServer struct for every request and repeat those checks,
	// the package is probably doing more than necessary (i.e. CORS support and
	// endpoint filtering, Origin checking, etc -- which are all things Caddy can be
	// configured to do; but maybe it's better for its own internal enforcement of
	// those? I dunno). All we really need to do is bridge gRPC-web requests and
	// gRPC-websockets requests. We can probably handle the rest. A more flexible API
	// would be nice though, so we don't have to re-create the wrapped server each
	// request. Thankfully, the WrappedGrpcServer struct doesn't actually carry any
	// state; it's more like a fancy config struct that wraps an HTTP handler, so
	// it should be easy to do. Sigh. I dunno, maybe this is fine. I just feel weird
	// not reusing the WrappedGrpcServer struct.
	// See https://github.com/improbable-eng/grpc-web/issues/1118.
	if isGRPCWeb(r) || isGRPCWebSocket(r) {
		var err error
		grpcWebBridge := grpcweb.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err = next.ServeHTTP(w, r)
		}), grpcweb.WithWebsocketPingInterval(time.Duration(h.WebSocketPing)))
		grpcWebBridge.ServeHTTP(w, r)
		return err
	}

	// pass-thru for all other requests
	return next.ServeHTTP(w, r)
}

// UnmarshalCaddyfile sets up h from Caddyfile tokens. Syntax:
//
// grpc_web {
//     websocket_ping <interval>
// }
//
func (h *Handler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if d.NextArg() {
			return d.ArgErr()
		}

		for nesting := d.Nesting(); d.NextBlock(nesting); {
			switch d.Val() {
			case "websocket_ping":
				if !d.NextArg() {
					return d.ArgErr()
				}
				dur, err := caddy.ParseDuration(d.Val())
				if err != nil {
					return d.Errf("bad interval value %s: %v", d.Val(), err)
				}
				h.WebSocketPing = caddy.Duration(dur)
				if d.NextArg() {
					return d.ArgErr()
				}
			default:
				return d.Errf("unknown subdirective '%s'", d.Val())
			}
		}
	}

	return nil
}

func isGRPCWeb(req *http.Request) bool {
	return req.Method == http.MethodPost && strings.HasPrefix(req.Header.Get("Content-Type"), "application/grpc-web")
}

func isGRPCWebSocket(req *http.Request) bool {
	return strings.ToLower(req.Header.Get("Upgrade")) == "websocket" && strings.ToLower(req.Header.Get("Sec-Websocket-Protocol")) == "grpc-websockets"
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var handler Handler
	err := handler.UnmarshalCaddyfile(h.Dispenser)
	return handler, err
}

// Interface guards
var (
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
	_ caddyfile.Unmarshaler       = (*Handler)(nil)
)
