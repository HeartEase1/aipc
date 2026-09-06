package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// CodexModels serves the Codex models manifest for Codex clients.
//
// Codex CLI and the Codex desktop app refresh their model picker from
// GET {base_url}/models?client_version=... (custom provider mode) or
// GET /backend-api/codex/models (chatgpt_base_url mode). Both routes land
// here. ChatGPT manifests are proxied verbatim; custom API key manifests receive
// provider-compatibility normalization and use a short-lived, asynchronously
// revalidated cache to tolerate canceled client requests.
func (h *OpenAIGatewayHandler) CodexModels(c *gin.Context) {
	if c.Request.Context().Err() != nil {
		return
	}
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey.Group == nil {
		h.errorResponse(c, http.StatusUnauthorized, "invalid_request_error", "API key group is required")
		return
	}
	if apiKey.Group.Platform != service.PlatformOpenAI && apiKey.Group.Platform != service.PlatformComposite {
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "Codex models manifest is only available for OpenAI and Composite groups")
		return
	}
	ifNoneMatch := c.GetHeader("If-None-Match")
	if apiKey.Group.HasBlockedModels() {
		// The upstream ETag identifies the unfiltered manifest. Do not allow a
		// client-side 304 to retain models that this group now blocks.
		ifNoneMatch = ""
	}

	// 固定账号分支：开启后只用选定账号拉取 manifest，不经过调度器；
	// 全部不可用/全部失败时按 FallbackToScheduler 决定回退调度器或返回错误。
	if apiKey.Group.Platform == service.PlatformOpenAI &&
		apiKey.Group.CodexModelsManifestConfig.Enabled {
		pinnedManifest, pinnedAccount, pinnedErr := h.gatewayService.FetchPinnedCodexModelsManifest(
			c.Request.Context(),
			apiKey.Group,
			c.Query("client_version"),
		)
		if pinnedErr != nil {
			if c.Request.Context().Err() != nil {
				return
			}
			if !apiKey.Group.CodexModelsManifestConfig.FallbackToScheduler {
				if errors.Is(pinnedErr, service.ErrNoPinnedCodexModelsAccounts) {
					h.errorResponse(c, http.StatusServiceUnavailable, "upstream_error", "No available pinned OpenAI accounts")
					return
				}
				h.errorResponse(c, infraerrors.Code(pinnedErr), "upstream_error", infraerrors.Message(pinnedErr))
				return
			}
			// 回退开启：跌入下方调度器循环。
		} else {
			// 让 ops 错误日志携带实际拉取成功的首个固定账号。
			setOpsSelectedAccount(c, pinnedAccount.ID, pinnedAccount.Platform)
			body, err := filterModelListEnvelope(pinnedManifest.Body, "models", apiKey.Group, "slug", "id", "name")
			if err != nil {
				h.errorResponse(c, http.StatusInternalServerError, "api_error", "Failed to apply group model policy")
				return
			}
			if c.Request.Context().Err() != nil {
				return
			}
			if pinnedManifest.ETag != "" && !apiKey.Group.HasBlockedModels() {
				c.Header("ETag", pinnedManifest.ETag)
				if service.CodexModelsManifestETagMatches(ifNoneMatch, pinnedManifest.ETag) {
					c.Status(http.StatusNotModified)
					c.Writer.WriteHeaderNow()
					return
				}
			}
			c.Data(http.StatusOK, "application/json", body)
			return
		}
	}

	maxAccountSwitches := h.maxAccountSwitches
	if apiKey.Group.Platform == service.PlatformOpenAI {
		configured, available, err := h.gatewayService.BuildGroupConfiguredCodexModelsManifest(c.Request.Context(), apiKey.Group, "")
		if err != nil {
			h.errorResponse(c, http.StatusInternalServerError, "api_error", "Failed to build configured model catalog")
			return
		}
		if available {
			body, err := filterModelListEnvelope(configured.Body, "models", apiKey.Group, "slug", "id", "name")
			if err != nil {
				h.errorResponse(c, http.StatusInternalServerError, "api_error", "Failed to apply group model policy")
				return
			}
			writeCodexCatalog(c, body)
			return
		}
	}
	if maxAccountSwitches <= 0 {
		maxAccountSwitches = 3
	}
	failedAccountIDs := make(map[int64]struct{})
	switchCount := 0
	var lastUpstreamErr error

	for {
		account, err := h.gatewayService.SelectAccountForModelWithExclusions(c.Request.Context(), apiKey.GroupID, "", "", failedAccountIDs)
		if err != nil {
			if c.Request.Context().Err() != nil {
				return
			}
			if lastUpstreamErr != nil {
				h.errorResponse(c, infraerrors.Code(lastUpstreamErr), "upstream_error", infraerrors.Message(lastUpstreamErr))
				return
			}
			h.errorResponse(c, http.StatusServiceUnavailable, "upstream_error", "No available OpenAI accounts")
			return
		}
		// 让 ops 错误日志携带实际选中的上游账号，便于定位失效账号（#4544）。
		setOpsSelectedAccount(c, account.ID, account.Platform)

		manifest, err := h.gatewayService.FetchCodexModelsManifest(c.Request.Context(), account, c.Query("client_version"), "")
		if err != nil {
			if c.Request.Context().Err() != nil {
				return
			}
			if service.IsRetryableCodexModelsManifestError(err) && switchCount < maxAccountSwitches {
				failedAccountIDs[account.ID] = struct{}{}
				switchCount++
				lastUpstreamErr = err
				continue
			}
			h.errorResponse(c, infraerrors.Code(err), "upstream_error", infraerrors.Message(err))
			return
		}
		if c.Request.Context().Err() != nil {
			return
		}

		if err := h.gatewayService.CompleteAPIKeyCodexModelsManifestForClient(manifest, account); err != nil {
			h.errorResponse(c, http.StatusInternalServerError, "api_error", "Failed to complete model capabilities")
			return
		}
		if manifest.NotModified {
			c.Status(http.StatusNotModified)
			return
		}
		body, filterErr := filterModelListEnvelope(manifest.Body, "models", apiKey.Group, "slug", "id", "name")
		if filterErr != nil {
			h.errorResponse(c, http.StatusInternalServerError, "api_error", "Failed to apply group model policy")
			return
		}
		if !apiKey.Group.HasBlockedModels() && manifest.ETag != "" {
			c.Header("ETag", manifest.ETag)
			if service.CodexModelsManifestETagMatches(ifNoneMatch, manifest.ETag) {
				c.Status(http.StatusNotModified)
				return
			}
		}
		c.Data(http.StatusOK, "application/json", body)
		return
	}
}
