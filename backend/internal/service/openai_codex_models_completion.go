package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func (s *OpenAIGatewayService) CompleteAPIKeyCodexModelsManifestForClient(manifest *CodexModelsManifest, account *Account) error {
	if manifest == nil || account == nil || !account.IsOpenAIApiKey() || manifest.NotModified || len(manifest.Body) == 0 {
		return nil
	}
	body := manifest.Body
	if len(manifest.upstreamSourceBody) > 0 {
		body = append([]byte(nil), manifest.upstreamSourceBody...)
		if manifest.convertedFromOpenAIModelList {
			body = convertOpenAIModelListToCodexManifest(body, account)
		}
	}
	var err error
	body, err = applySyncedAPIKeyCodexModelMetadata(body, account, false)
	if err != nil {
		return err
	}
	body, err = completeAPIKeyCodexModelsManifestMetadata(
		body,
		true,
		account,
	)
	if err != nil {
		return err
	}
	body, err = adjustAPIKeyCodexModelsManifest(body, account)
	if err != nil {
		return err
	}
	manifest.Body = body
	manifest.ETag = codexModelsManifestBodyETag(manifest.Body)
	return nil
}

func applySyncedAPIKeyCodexModelMetadata(body []byte, account *Account, overwriteLocalDefaults bool) ([]byte, error) {
	snapshot := account.GetUpstreamModelMetadataSnapshot()
	if snapshot == nil || len(snapshot.Models) == 0 {
		return body, nil
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode JSON object: %w", err)
	}
	var models []json.RawMessage
	if err := json.Unmarshal(envelope["models"], &models); err != nil {
		return nil, fmt.Errorf("decode top-level models array: %w", err)
	}

	changed := false
	for i, rawModel := range models {
		var model map[string]json.RawMessage
		if err := json.Unmarshal(rawModel, &model); err != nil || model == nil {
			continue
		}
		var slug string
		if err := json.Unmarshal(model["slug"], &slug); err != nil {
			continue
		}
		slug = strings.TrimSpace(slug)
		lookupModel := account.GetMappedModel(slug)
		metadata, ok := snapshot.Models[lookupModel]
		if !ok {
			continue
		}
		if lookupModel != slug {
			metadata.DisplayName = ""
			metadata.Description = ""
		}

		descriptor := newConfiguredCodexModelDescriptor(slug)
		applyUpstreamModelMetadataToCodexDescriptor(
			&descriptor,
			codexModelMetadataOverride{UpstreamModelMetadata: metadata},
		)
		descriptorBody, err := json.Marshal(descriptor)
		if err != nil {
			return nil, fmt.Errorf("encode synced model %q: %w", slug, err)
		}
		var syncedFields map[string]json.RawMessage
		if err := json.Unmarshal(descriptorBody, &syncedFields); err != nil {
			return nil, fmt.Errorf("decode synced model %q: %w", slug, err)
		}

		fields := make([]string, 0, 7)
		if strings.TrimSpace(metadata.DisplayName) != "" {
			fields = append(fields, "display_name")
		}
		if strings.TrimSpace(metadata.Description) != "" {
			fields = append(fields, "description")
		}
		if metadata.Reasoning != nil {
			fields = append(fields, "default_reasoning_level", "supported_reasoning_levels")
		}
		if len(normalizeCodexInputModalities(metadata.InputModalities)) > 0 {
			fields = append(fields, "input_modalities")
		}
		if metadata.ContextWindow > 0 {
			fields = append(fields, "context_window", "max_context_window")
		}
		if metadata.MaxOutputTokens > 0 {
			fields = append(fields, "max_output_tokens")
		}

		// List conversion has already applied live fields over account capabilities.
		modelChanged := applyCodexToolCapabilities(model, metadata.CodexToolCapabilities, false)
		for _, field := range fields {
			value, exists := syncedFields[field]
			if !exists {
				continue
			}
			current, currentExists := model[field]
			current = bytes.TrimSpace(current)
			if !overwriteLocalDefaults && currentExists && len(current) > 0 && !bytes.Equal(current, []byte("null")) {
				continue
			}
			if bytes.Equal(current, bytes.TrimSpace(value)) {
				continue
			}
			model[field] = value
			modelChanged = true
		}
		if !modelChanged {
			continue
		}
		encoded, err := json.Marshal(model)
		if err != nil {
			return nil, fmt.Errorf("encode model %q with synced metadata: %w", slug, err)
		}
		models[i] = encoded
		changed = true
	}
	if !changed {
		return body, nil
	}

	encodedModels, err := json.Marshal(models)
	if err != nil {
		return nil, fmt.Errorf("encode models with synced metadata: %w", err)
	}
	envelope["models"] = encodedModels
	updated, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("encode manifest with synced metadata: %w", err)
	}
	return updated, nil
}

func completeAPIKeyCodexModelsManifestMetadata(body []byte, completeAll bool, account *Account) ([]byte, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode JSON object: %w", err)
	}
	var models []json.RawMessage
	if err := json.Unmarshal(envelope["models"], &models); err != nil {
		return nil, fmt.Errorf("decode top-level models array: %w", err)
	}

	officialOpenAI := account != nil && isOfficialOpenAIModelsBaseURL(account.GetOpenAIBaseURL())
	changed := false
	if officialOpenAI {
		filtered := make([]json.RawMessage, 0, len(models))
		for _, rawModel := range models {
			var model struct {
				Slug string `json:"slug"`
			}
			if err := json.Unmarshal(rawModel, &model); err != nil || strings.TrimSpace(model.Slug) == "" {
				filtered = append(filtered, rawModel)
				continue
			}
			if !isOfficialOpenAICodexCatalogModel(model.Slug) {
				changed = true
				continue
			}
			filtered = append(filtered, rawModel)
		}
		models = filtered
	}
	for i, rawModel := range models {
		var model map[string]json.RawMessage
		if err := json.Unmarshal(rawModel, &model); err != nil || model == nil {
			continue
		}
		var slug string
		if err := json.Unmarshal(model["slug"], &slug); err != nil {
			continue
		}
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}

		completeDescriptor := completeAll || isDeepSeekCodexModel(slug)
		forceOfficialImage := officialOpenAI && isOpenAICodexImageInputModel(slug)
		if !completeDescriptor && !forceOfficialImage {
			continue
		}

		descriptor := newConfiguredCodexModelDescriptor(slug)
		descriptor.SupportsSearchTool = shouldForwardOpenAIResponsesViaRawChatCompletions(account)
		if accountCodexModelSupportsImageInput(account, slug) {
			descriptor.InputModalities = []string{"text", "image"}
		}
		if forceOfficialImage {
			descriptor.InputModalities = []string{"text", "image"}
			descriptor.SupportsImageDetailOriginal = true
		}
		defaultBody, err := json.Marshal(descriptor)
		if err != nil {
			return nil, fmt.Errorf("encode default model %q: %w", slug, err)
		}
		var defaults map[string]json.RawMessage
		if err := json.Unmarshal(defaultBody, &defaults); err != nil {
			return nil, fmt.Errorf("decode default model %q: %w", slug, err)
		}

		capabilityModel := slug
		if account != nil {
			capabilityModel = account.GetMappedModel(slug)
		}
		capabilities := accountCodexToolCapabilities(account, capabilityModel)
		modelChanged := applyCodexToolCapabilities(model, capabilities, false)
		if completeDescriptor {
			merged, err := mergeMissingCodexModelFields(model, defaults)
			if err != nil {
				return nil, fmt.Errorf("complete model %q: %w", slug, err)
			}
			modelChanged = merged || modelChanged
		}
		if forceOfficialImage {
			modalities, err := json.Marshal([]string{"text", "image"})
			if err != nil {
				return nil, fmt.Errorf("encode input modalities for model %q: %w", slug, err)
			}
			if !bytes.Equal(bytes.TrimSpace(model["input_modalities"]), modalities) {
				model["input_modalities"] = modalities
				modelChanged = true
			}
			imageDetailOriginal := json.RawMessage("true")
			if !bytes.Equal(bytes.TrimSpace(model["supports_image_detail_original"]), imageDetailOriginal) {
				model["supports_image_detail_original"] = imageDetailOriginal
				modelChanged = true
			}
		}
		if !modelChanged {
			continue
		}
		encoded, err := json.Marshal(model)
		if err != nil {
			return nil, fmt.Errorf("encode completed model %q: %w", slug, err)
		}
		models[i] = encoded
		changed = true
	}
	if !changed {
		return body, nil
	}

	encodedModels, err := json.Marshal(models)
	if err != nil {
		return nil, fmt.Errorf("encode top-level models array: %w", err)
	}
	envelope["models"] = encodedModels
	completed, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("encode JSON object: %w", err)
	}
	return completed, nil
}

func mergeMissingCodexModelFields(current, defaults map[string]json.RawMessage) (bool, error) {
	changed := false
	for key, defaultValue := range defaults {
		currentValue, exists := current[key]
		if exists && stringSliceContains(codexToolCapabilityFields, key) {
			continue
		}
		if !exists || (bytes.Equal(bytes.TrimSpace(currentValue), []byte("null")) &&
			!bytes.Equal(bytes.TrimSpace(defaultValue), []byte("null"))) {
			current[key] = defaultValue
			changed = true
			continue
		}

		var currentObject map[string]json.RawMessage
		var defaultObject map[string]json.RawMessage
		if err := json.Unmarshal(currentValue, &currentObject); err != nil || currentObject == nil {
			continue
		}
		if err := json.Unmarshal(defaultValue, &defaultObject); err != nil || defaultObject == nil {
			continue
		}
		nestedChanged, err := mergeMissingCodexModelFields(currentObject, defaultObject)
		if err != nil {
			return false, err
		}
		if !nestedChanged {
			continue
		}
		mergedValue, err := json.Marshal(currentObject)
		if err != nil {
			return false, fmt.Errorf("encode field %q: %w", key, err)
		}
		current[key] = mergedValue
		changed = true
	}
	return changed, nil
}
