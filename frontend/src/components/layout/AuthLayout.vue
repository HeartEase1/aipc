<template>
  <main class="auth-shell">
    <section class="auth-brand-panel" :aria-label="siteName">
      <template v-if="settingsLoaded">
        <div class="auth-brand-mark">
          <span class="auth-logo-frame auth-logo-frame-compact">
            <img :src="siteLogo || '/logo.svg'" :alt="siteName" class="auth-logo-image" />
          </span>
          <span class="auth-brand-name-compact">{{ siteName }}</span>
        </div>

        <div class="auth-brand-copy">
          <h1>{{ siteName }}</h1>
          <p>{{ siteSubtitle }}</p>
        </div>

        <p class="auth-copyright">
          &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
        </p>
      </template>
    </section>

    <section class="auth-form-panel">
      <div class="auth-form-wrap">
        <div v-if="settingsLoaded" class="auth-mobile-brand">
          <span class="auth-logo-frame">
            <img :src="siteLogo || '/logo.svg'" :alt="siteName" class="auth-logo-image" />
          </span>
          <div class="min-w-0">
            <h1>{{ siteName }}</h1>
            <p>{{ siteSubtitle }}</p>
          </div>
        </div>

        <div class="auth-card">
          <slot />
        </div>

        <div class="auth-footer">
          <slot name="footer" />
        </div>

        <p class="auth-mobile-copyright">
          &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
        </p>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() =>
  sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }),
)
const siteSubtitle = computed(
  () =>
    appStore.cachedPublicSettings?.site_subtitle ||
    'Subscription to API Conversion Platform',
)
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.auth-shell {
  --color-primary-50: 239 246 255;
  --color-primary-100: 219 234 254;
  --color-primary-200: 191 219 254;
  --color-primary-300: 147 197 253;
  --color-primary-400: 96 165 250;
  --color-primary-500: 59 130 246;
  --color-primary-600: 37 99 235;
  --color-primary-700: 29 78 216;
  --color-primary-800: 30 64 175;
  --color-primary-900: 30 58 138;
  --color-primary-950: 23 37 84;

  display: grid;
  min-height: 100vh;
  min-height: 100dvh;
  grid-template-columns: minmax(30rem, 1.04fr) minmax(32rem, 0.96fr);
  overflow-x: hidden;
  background: #f8fafc;
  color: #111827;
}

.auth-brand-panel {
  position: relative;
  display: grid;
  min-height: 100vh;
  min-height: 100dvh;
  grid-template-rows: auto minmax(0, 1fr) auto;
  overflow: hidden;
  border-right: 1px solid rgba(148, 163, 184, 0.2);
  background-color: #eef2f7;
  background-image:
    linear-gradient(rgba(100, 116, 139, 0.075) 1px, transparent 1px),
    linear-gradient(90deg, rgba(100, 116, 139, 0.075) 1px, transparent 1px);
  background-size: 56px 56px;
  padding: 3rem clamp(3rem, 6vw, 6rem);
}

.auth-brand-panel::after {
  position: absolute;
  inset: auto 0 0;
  height: 34%;
  background: linear-gradient(to top, rgba(191, 219, 254, 0.42), transparent);
  content: '';
  pointer-events: none;
}

.auth-brand-mark,
.auth-brand-copy,
.auth-copyright {
  position: relative;
  z-index: 1;
}

.auth-brand-mark {
  display: flex;
  align-items: center;
  gap: 0.875rem;
}

.auth-logo-frame {
  display: inline-flex;
  width: 3.5rem;
  height: 3.5rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.24);
  border-radius: 1rem;
  background: rgba(255, 255, 255, 0.86);
  box-shadow:
    0 16px 28px -18px rgba(15, 23, 42, 0.38),
    inset 0 1px 0 rgba(255, 255, 255, 0.88);
}

.auth-logo-frame-compact {
  width: 2.75rem;
  height: 2.75rem;
  border-radius: 0.75rem;
}

.auth-logo-image {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.auth-brand-name-compact {
  overflow: hidden;
  color: #475569;
  font-size: 1rem;
  font-weight: 600;
  letter-spacing: 0;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.auth-brand-copy {
  align-self: center;
  max-width: 38rem;
  padding: 4rem 0;
}

.auth-brand-copy h1 {
  overflow-wrap: anywhere;
  color: #0f172a;
  font-size: 3.5rem;
  font-weight: 750;
  letter-spacing: 0;
  line-height: 1.08;
}

.auth-brand-copy p {
  max-width: 34rem;
  margin-top: 1.25rem;
  color: #64748b;
  font-size: 1.125rem;
  line-height: 1.75;
}

.auth-copyright,
.auth-mobile-copyright {
  color: #94a3b8;
  font-size: 0.75rem;
  letter-spacing: 0;
}

.auth-form-panel {
  display: flex;
  min-width: 0;
  min-height: 100vh;
  min-height: 100dvh;
  align-items: center;
  justify-content: center;
  padding: 3rem clamp(2rem, 5vw, 5rem);
  background: #f8fafc;
}

.auth-form-wrap {
  width: min(100%, 29rem);
  margin-block: auto;
}

.auth-mobile-brand,
.auth-mobile-copyright {
  display: none;
}

.auth-card {
  width: 100%;
  border: 1px solid rgba(148, 163, 184, 0.24);
  border-radius: 1.5rem;
  background: rgba(255, 255, 255, 0.92);
  box-shadow:
    0 30px 64px -36px rgba(15, 23, 42, 0.42),
    0 12px 28px -20px rgba(15, 23, 42, 0.26),
    inset 0 1px 0 rgba(255, 255, 255, 0.9);
  padding: 2.25rem;
  animation: auth-card-enter 360ms cubic-bezier(0.22, 1, 0.36, 1) both;
}

.auth-footer {
  margin-top: 1.5rem;
  text-align: center;
  font-size: 0.875rem;
}

/* Authentication pages do not use the console shell, so bridge the old
   teal primary utilities to the same blue action color used by modern UI. */
.auth-card :deep(.btn-primary) {
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: #2563eb;
  color: #ffffff;
  box-shadow: 0 10px 24px -12px rgba(37, 99, 235, 0.62);
}

.auth-card :deep(.btn-primary:hover:not(:disabled)) {
  background: #1d4ed8;
  box-shadow: 0 13px 28px -13px rgba(37, 99, 235, 0.72);
}

.auth-card :deep(.input:focus) {
  border-color: #2563eb;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.14);
}

.auth-shell :deep(.text-primary-600),
.auth-footer :deep(.text-primary-600) {
  color: #2563eb !important;
}

.auth-shell :deep(.hover\:text-primary-500:hover),
.auth-footer :deep(.hover\:text-primary-500:hover) {
  color: #1d4ed8 !important;
}

@keyframes auth-card-enter {
  from {
    opacity: 0;
    transform: translateY(10px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

:global(html.dark .auth-shell),
:global(html.dark .auth-form-panel) {
  background: #080d15;
  color: #f8fafc;
}

:global(html.dark .auth-brand-panel) {
  border-right-color: rgba(100, 116, 139, 0.22);
  background-color: #101722;
  background-image:
    linear-gradient(rgba(148, 163, 184, 0.07) 1px, transparent 1px),
    linear-gradient(90deg, rgba(148, 163, 184, 0.07) 1px, transparent 1px);
}

:global(html.dark .auth-brand-panel::after) {
  background: linear-gradient(to top, rgba(14, 116, 144, 0.18), transparent);
}

:global(html.dark .auth-logo-frame) {
  border-color: rgba(100, 116, 139, 0.3);
  background: rgba(30, 41, 59, 0.88);
  box-shadow:
    0 18px 32px -20px rgba(0, 0, 0, 0.8),
    inset 0 1px 0 rgba(255, 255, 255, 0.06);
}

:global(html.dark .auth-brand-name-compact),
:global(html.dark .auth-brand-copy p) {
  color: #94a3b8;
}

:global(html.dark .auth-brand-copy h1) {
  color: #f8fafc;
}

:global(html.dark .auth-card) {
  border-color: rgba(100, 116, 139, 0.3);
  background: rgba(15, 23, 42, 0.9);
  box-shadow:
    0 32px 68px -34px rgba(0, 0, 0, 0.9),
    0 14px 28px -24px rgba(0, 0, 0, 0.75),
    inset 0 1px 0 rgba(255, 255, 255, 0.05);
}

:global(html.dark .auth-card .btn-primary) {
  background: #3b82f6;
  color: #ffffff;
  box-shadow: 0 12px 26px -13px rgba(59, 130, 246, 0.68);
}

:global(html.dark .auth-card .btn-primary:hover:not(:disabled)) {
  background: #60a5fa;
  color: #0b1f45;
}

:global(html.dark .auth-shell .dark\:text-primary-400),
:global(html.dark .auth-footer .dark\:text-primary-400) {
  color: #60a5fa !important;
}

@media (max-width: 1023px) {
  .auth-shell {
    display: block;
    background-color: #eef2f7;
    background-image:
      linear-gradient(rgba(100, 116, 139, 0.06) 1px, transparent 1px),
      linear-gradient(90deg, rgba(100, 116, 139, 0.06) 1px, transparent 1px);
    background-size: 48px 48px;
  }

  .auth-brand-panel {
    display: none;
  }

  .auth-form-panel {
    min-height: 100vh;
    min-height: 100dvh;
    padding: 2rem 1.5rem;
    background: transparent;
  }

  .auth-mobile-brand {
    display: flex;
    max-width: 100%;
    align-items: center;
    gap: 0.875rem;
    margin-bottom: 1.5rem;
  }

  .auth-mobile-brand h1 {
    overflow: hidden;
    color: #0f172a;
    font-size: 1.25rem;
    font-weight: 700;
    letter-spacing: 0;
    line-height: 1.3;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .auth-mobile-brand p {
    display: -webkit-box;
    overflow: hidden;
    margin-top: 0.2rem;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
    color: #64748b;
    font-size: 0.75rem;
    line-height: 1.45;
  }

  .auth-mobile-copyright {
    display: block;
    margin-top: 1.75rem;
    text-align: center;
  }

  :global(html.dark .auth-shell) {
    background-color: #0b111a;
    background-image:
      linear-gradient(rgba(148, 163, 184, 0.055) 1px, transparent 1px),
      linear-gradient(90deg, rgba(148, 163, 184, 0.055) 1px, transparent 1px);
  }

  :global(html.dark .auth-mobile-brand h1) {
    color: #f8fafc;
  }

  :global(html.dark .auth-mobile-brand p) {
    color: #94a3b8;
  }
}

@media (max-width: 479px) {
  .auth-form-panel {
    padding: 1.25rem 1rem 1.75rem;
  }

  .auth-logo-frame {
    width: 3rem;
    height: 3rem;
    border-radius: 0.875rem;
  }

  .auth-card {
    border-radius: 1.125rem;
    padding: 1.5rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .auth-card {
    animation: none;
  }
}
</style>
