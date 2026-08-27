# Upstream source

This directory vendors [CookSleep/gpt_image_playground](https://github.com/CookSleep/gpt_image_playground)
under its MIT license.

- Upstream release: `v0.7.3`
- Upstream commit: `0348f72`
- Imported: `2026-08-19`

AIPC-specific integration is intentionally limited to the hosted bridge, local-data isolation,
navigation branding, build output, and security controls. Keep upstream changes separate when
updating this snapshot so future merges remain reviewable.

## Local integration surface

Review these files carefully when importing a newer upstream release:

- `src/lib/hosted.ts` and `src/lib/hosted.test.ts`: same-origin host handshake, secret scrubbing,
  per-user storage names, and legacy service-worker cleanup.
- `src/main.tsx`, `src/App.tsx`, `src/components/Header.tsx`, and `src/hooks/useVersionCheck.ts`:
  hosted-only startup and removal of standalone settings, PWA, update, and donation behavior.
- `src/store.ts` and `src/lib/db.ts`: per-user persistence, API-key removal from storage/export,
  local conversation titles, bounded Agent automation, and propagation of the Agent stop signal.
- `src/lib/imageApiShared.ts`, `src/lib/openaiCompatibleImageApi.ts`, and `src/lib/falAiImageApi.ts`:
  cancellation of in-flight hosted Agent image requests across the supported providers.
- `src/index.css`, `src/vite-env.d.ts`, and `vite.config.ts`: iframe layout and nested build output.

The remaining source files should normally match upstream. Do not replace the files above without
reapplying and testing the AIPC changes.

## Updating upstream

1. Import a clean tagged upstream snapshot into a temporary directory and compare it with this one.
2. Apply upstream changes first, then reapply the small integration surface listed above.
3. Keep upstream dependency versions and regenerate `package-lock.json` only when upstream changes
   them; do not independently upgrade this vendored application during a routine sync.
4. Run `npm test`, `npm run build`, the main Vue tests/typecheck/build, and the Go embed/security-header
   tests before release. Confirm the built files exist under
   `backend/internal/web/dist/playground-app/`.
