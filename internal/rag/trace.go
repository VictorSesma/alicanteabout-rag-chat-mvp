package rag

import "context"

type reqIDKey struct{}

// WithRequestID attaches a request ID to the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, reqIDKey{}, id)
}

// RequestID returns the request ID from context if present.
func RequestID(ctx context.Context) string {
	if v := ctx.Value(reqIDKey{}); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
