package service

import "context"

type apiKeyFastModeContextKey struct{}

// WithAPIKeyFastMode carries the authenticated key's Fast preference into the
// gateway without coupling service code to the HTTP middleware package.
func WithAPIKeyFastMode(ctx context.Context, enabled bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, apiKeyFastModeContextKey{}, enabled)
}

// APIKeyFastModeEnabled reports the preference attached by API key auth.
func APIKeyFastModeEnabled(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	enabled, _ := ctx.Value(apiKeyFastModeContextKey{}).(bool)
	return enabled
}

func apiKeyFastModeApplies(ctx context.Context, account *Account) bool {
	return APIKeyFastModeEnabled(ctx) && account != nil && account.Platform == PlatformOpenAI
}
