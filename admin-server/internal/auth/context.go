package auth

import "context"

type contextKey string

const (
	clientIPKey  contextKey = "admin_auth_client_ip"
	userAgentKey contextKey = "admin_auth_user_agent"
)

// WithClientInfo attaches client IP and User-Agent to the context.
func WithClientInfo(ctx context.Context, ip, userAgent string) context.Context {
	ctx = context.WithValue(ctx, clientIPKey, ip)
	return context.WithValue(ctx, userAgentKey, userAgent)
}

// ClientInfoFromContext extracts client IP and User-Agent from context.
func ClientInfoFromContext(ctx context.Context) (ip, userAgent string) {
	if ctx == nil {
		return "", ""
	}
	if v, ok := ctx.Value(clientIPKey).(string); ok {
		ip = v
	}
	if v, ok := ctx.Value(userAgentKey).(string); ok {
		userAgent = v
	}
	return
}
