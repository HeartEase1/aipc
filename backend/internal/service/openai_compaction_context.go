package service

import "context"

type openAIForwardModelContextKey struct{}

type openAIForwardModel struct {
	model                  string
	useCompactModelMapping bool
	requestedModel         string
}

// WithOpenAIForwardModel records the model present in the forwarded request
// body after channel mapping and whether the legacy /responses/compact-only
// model mapping applies. Native remote compaction v2 keeps this false, so
// channel restriction checks follow the same model chain used by Forward.
func WithOpenAIForwardModel(ctx context.Context, forwardModel string, useCompactModelMapping bool, requestedModels ...string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	value := openAIForwardModel{
		model:                  forwardModel,
		useCompactModelMapping: useCompactModelMapping,
	}
	if len(requestedModels) > 0 {
		value.requestedModel = requestedModels[0]
	}
	return context.WithValue(ctx, openAIForwardModelContextKey{}, value)
}

// Channel mappings may override an alias already explicitly allowed by an
// account. Keep that whitelist contract while scheduling on the forwarded model.
func openAIAccountSupportsSchedulingModel(ctx context.Context, account *Account, model string) bool {
	if account == nil {
		return false
	}
	if model == "" || account.IsModelSupported(model) {
		return true
	}
	forward, ok := openAIForwardModelFromContext(ctx)
	return ok && forward.model == model && forward.requestedModel != "" &&
		forward.requestedModel != model &&
		mappingSupportsRequestedModel(account.GetModelMapping(), forward.requestedModel)
}

func openAIForwardModelFromContext(ctx context.Context) (openAIForwardModel, bool) {
	if ctx == nil {
		return openAIForwardModel{}, false
	}
	forwardModel, ok := ctx.Value(openAIForwardModelContextKey{}).(openAIForwardModel)
	return forwardModel, ok
}
