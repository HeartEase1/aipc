<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Compact Home Page -->
  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"
  >
    <header class="border-b border-gray-200 px-4 py-4 sm:px-6 dark:border-dark-800">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-lg object-contain"
          />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <button
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain"
        />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">{{ siteSubtitle }}</p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-8 inline-flex min-h-10 items-center justify-center rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <!-- Default Home Page -->
  <div
    v-else
    data-testid="default-home"
    class="relative flex min-h-screen flex-col overflow-hidden bg-[#030712]"
  >
    <!-- Background: subtle deep-space grid + radial glow -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div class="absolute inset-0 bg-[radial-gradient(ellipse_80%_50%_at_50%_-10%,rgba(56,189,248,0.08),transparent)]"></div>
      <div class="absolute inset-0 bg-[linear-gradient(rgba(56,189,248,0.03)_1px,transparent_1px),linear-gradient(90deg,rgba(56,189,248,0.03)_1px,transparent_1px)] bg-[size:72px_72px]"></div>
    </div>

    <!-- Header -->
    <header class="relative z-20 border-b border-white/[0.06] px-6 py-4 backdrop-blur-sm">
      <nav class="mx-auto flex max-w-7xl items-center justify-between">
        <!-- Logo + Name -->
        <div class="flex items-center gap-3">
          <div class="h-9 w-9 overflow-hidden rounded-xl shadow-lg shadow-sky-500/20">
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <span class="text-sm font-semibold text-white/90 hidden sm:inline">{{ siteName }}</span>
        </div>

        <!-- Nav Actions -->
        <div class="flex items-center gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="rounded-lg p-2 text-white/40 transition-colors hover:bg-white/5 hover:text-white/80"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <button
            @click="toggleTheme"
            class="rounded-lg p-2 text-white/40 transition-colors hover:bg-white/5 hover:text-white/80"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="inline-flex items-center gap-1.5 rounded-full bg-white/10 py-1 pl-1 pr-3 text-xs font-medium text-white transition-colors hover:bg-white/15 border border-white/10"
          >
            <span class="flex h-5 w-5 items-center justify-center rounded-full bg-gradient-to-br from-sky-400 to-sky-600 text-[10px] font-semibold text-white">{{ userInitial }}</span>
            {{ t('home.dashboard') }}
          </router-link>
          <router-link
            v-else
            to="/login"
            class="inline-flex items-center rounded-full border border-white/15 bg-white/8 px-4 py-1.5 text-xs font-medium text-white transition-colors hover:bg-white/15"
          >
            {{ t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <!-- Main Content -->
    <main class="relative z-10 flex-1 overflow-hidden">
      <!-- Hero: full-viewport split layout -->
      <div class="mx-auto flex max-w-7xl flex-col lg:flex-row lg:min-h-[640px]">
        <!-- Left: Text -->
        <div class="flex flex-1 flex-col justify-center px-6 py-12 text-center sm:px-8 lg:py-0 lg:text-left">
          <!-- Badge -->
          <div class="hero-badge mb-5 inline-flex justify-center lg:justify-start">
            <span class="rounded-full border border-sky-500/30 bg-sky-500/10 px-3 py-1 text-xs font-medium text-sky-400">
              {{ t('home.heroBadge') }}
            </span>
          </div>
          <h1 class="hero-title mb-4 text-3xl font-bold tracking-tight text-white sm:text-4xl md:text-5xl lg:text-6xl">
            {{ siteName }}
          </h1>
          <p class="hero-sub mb-8 max-w-lg text-sm leading-relaxed text-white/55 sm:text-base md:text-lg lg:mx-0 mx-auto">
            {{ siteSubtitle }}
          </p>
          <!-- Feature pills -->
          <div class="hero-pills mb-8 flex flex-wrap justify-center gap-2 lg:justify-start">
            <span class="inline-flex items-center gap-1.5 rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs text-white/65">
              <Icon name="swap" size="xs" class="text-sky-400" />{{ t('home.tags.subscriptionToApi') }}
            </span>
            <span class="inline-flex items-center gap-1.5 rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs text-white/65">
              <Icon name="shield" size="xs" class="text-emerald-400" />{{ t('home.tags.stickySession') }}
            </span>
            <span class="inline-flex items-center gap-1.5 rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs text-white/65">
              <Icon name="chart" size="xs" class="text-violet-400" />{{ t('home.tags.realtimeBilling') }}
            </span>
          </div>
          <div class="hero-actions flex flex-wrap justify-center gap-3 lg:justify-start">
            <router-link
              :to="isAuthenticated ? dashboardPath : '/login'"
              class="inline-flex items-center gap-2 rounded-full bg-sky-500 px-5 py-2.5 text-sm font-semibold text-white shadow-lg shadow-sky-500/30 transition-all hover:bg-sky-400 hover:shadow-sky-400/40"
            >
              {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
              <Icon name="arrowRight" size="sm" :stroke-width="2.5" />
            </router-link>
            <a
              v-if="docUrl"
              :href="docUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="inline-flex items-center gap-2 rounded-full border border-white/12 bg-white/6 px-5 py-2.5 text-sm font-medium text-white/70 transition-colors hover:bg-white/10 hover:text-white"
            >
              {{ t('home.docs') }}
            </a>
          </div>
        </div>

        <!-- Right: Globe — fills the full height, no box border -->
        <div class="hero-globe relative flex-1 lg:flex-[1.1]">
          <GlobeScene />
        </div>
      </div>

      <!-- Features Grid -->
      <div class="border-t border-white/[0.06] px-8 py-16">
        <div class="mx-auto max-w-7xl">
          <div class="mb-10 text-center">
            <h2 class="text-2xl font-bold text-white">{{ t('home.advantagesTitle') }}</h2>
            <p class="mt-2 text-sm text-white/45">{{ t('home.advantagesSubtitle') }}</p>
          </div>
          <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <!-- Feature 1 -->
            <div class="group rounded-2xl border border-white/8 bg-white/[0.04] p-6 backdrop-blur-sm transition-all hover:border-sky-500/30 hover:bg-white/[0.07]">
              <div class="mb-4 flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-sky-500 to-blue-600 shadow-lg shadow-sky-500/30 transition-transform group-hover:scale-110">
                <Icon name="server" size="md" class="text-white" />
              </div>
              <h3 class="mb-1.5 text-sm font-semibold text-white">{{ t('home.features.unifiedGateway') }}</h3>
              <p class="text-xs leading-relaxed text-white/45">{{ t('home.features.unifiedGatewayDesc') }}</p>
            </div>
            <!-- Feature 2 -->
            <div class="group rounded-2xl border border-white/8 bg-white/[0.04] p-6 backdrop-blur-sm transition-all hover:border-emerald-500/30 hover:bg-white/[0.07]">
              <div class="mb-4 flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-emerald-500 to-teal-600 shadow-lg shadow-emerald-500/30 transition-transform group-hover:scale-110">
                <svg class="h-5 w-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z"/></svg>
              </div>
              <h3 class="mb-1.5 text-sm font-semibold text-white">{{ t('home.features.multiAccount') }}</h3>
              <p class="text-xs leading-relaxed text-white/45">{{ t('home.features.multiAccountDesc') }}</p>
            </div>
            <!-- Feature 3 -->
            <div class="group rounded-2xl border border-white/8 bg-white/[0.04] p-6 backdrop-blur-sm transition-all hover:border-violet-500/30 hover:bg-white/[0.07]">
              <div class="mb-4 flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-violet-500 to-purple-600 shadow-lg shadow-violet-500/30 transition-transform group-hover:scale-110">
                <svg class="h-5 w-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z"/></svg>
              </div>
              <h3 class="mb-1.5 text-sm font-semibold text-white">{{ t('home.features.balanceQuota') }}</h3>
              <p class="text-xs leading-relaxed text-white/45">{{ t('home.features.balanceQuotaDesc') }}</p>
            </div>
            <!-- Feature 4: Recharge Rate -->
            <div v-if="paymentEnabled" class="group relative overflow-hidden rounded-2xl border border-amber-500/25 bg-amber-500/[0.06] p-6 backdrop-blur-sm transition-all hover:border-amber-400/40 hover:bg-amber-500/[0.10]">
              <div class="mb-4 flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-amber-400 to-orange-500 shadow-lg shadow-amber-500/30 transition-transform group-hover:scale-110">
                <svg class="h-5 w-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6v12m-3-2.818.879.659c1.171.879 3.07.879 4.242 0 1.172-.879 1.172-2.303 0-3.182C13.536 12.219 12.768 12 12 12c-.725 0-1.45-.22-2.003-.659-1.106-.879-1.106-2.303 0-3.182s2.9-.879 4.006 0l.415.33M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" /></svg>
              </div>
              <h3 class="mb-1.5 text-sm font-semibold text-white">{{ t('home.features.rechargeRate') }}</h3>
              <div class="my-2 flex items-baseline gap-1.5">
                <span class="text-2xl font-black text-amber-400">1¥</span>
                <span class="text-base font-bold text-white/30">=</span>
                <span class="text-2xl font-black text-orange-400">{{ rechargeMultiplierDisplay }}$</span>
              </div>
              <p class="text-xs leading-relaxed text-white/40">{{ t('home.features.rechargeRateDesc', { amount: rechargeMultiplierDisplay }) }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Supported Providers -->
      <div class="border-t border-white/[0.06] px-8 py-12">
        <div class="mx-auto max-w-7xl text-center">
          <h2 class="mb-2 text-xl font-bold text-white">{{ t('home.providers.title') }}</h2>
          <p class="mb-8 text-sm text-white/40">{{ t('home.providers.description') }}</p>
          <div class="flex flex-wrap items-center justify-center gap-3">
            <div class="flex items-center gap-2 rounded-xl border border-sky-500/20 bg-sky-500/8 px-4 py-2.5">
              <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-br from-orange-400 to-orange-500"><span class="text-[11px] font-bold text-white">C</span></div>
              <span class="text-sm font-medium text-white/80">{{ t('home.providers.claude') }}</span>
              <span class="rounded bg-sky-500/20 px-1.5 py-0.5 text-[10px] font-medium text-sky-400">{{ t('home.providers.supported') }}</span>
            </div>
            <div class="flex items-center gap-2 rounded-xl border border-sky-500/20 bg-sky-500/8 px-4 py-2.5">
              <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-br from-green-500 to-green-600"><span class="text-[11px] font-bold text-white">G</span></div>
              <span class="text-sm font-medium text-white/80">GPT</span>
              <span class="rounded bg-sky-500/20 px-1.5 py-0.5 text-[10px] font-medium text-sky-400">{{ t('home.providers.supported') }}</span>
            </div>
            <div class="flex items-center gap-2 rounded-xl border border-sky-500/20 bg-sky-500/8 px-4 py-2.5">
              <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-br from-blue-500 to-blue-600"><span class="text-[11px] font-bold text-white">G</span></div>
              <span class="text-sm font-medium text-white/80">{{ t('home.providers.gemini') }}</span>
              <span class="rounded bg-sky-500/20 px-1.5 py-0.5 text-[10px] font-medium text-sky-400">{{ t('home.providers.supported') }}</span>
            </div>
            <div class="flex items-center gap-2 rounded-xl border border-sky-500/20 bg-sky-500/8 px-4 py-2.5">
              <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-br from-rose-500 to-pink-600"><span class="text-[11px] font-bold text-white">A</span></div>
              <span class="text-sm font-medium text-white/80">{{ t('home.providers.antigravity') }}</span>
              <span class="rounded bg-sky-500/20 px-1.5 py-0.5 text-[10px] font-medium text-sky-400">{{ t('home.providers.supported') }}</span>
            </div>
            <div class="flex items-center gap-2 rounded-xl border border-white/8 bg-white/4 px-4 py-2.5 opacity-50">
              <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-br from-gray-500 to-gray-600"><span class="text-[11px] font-bold text-white">+</span></div>
              <span class="text-sm font-medium text-white/60">{{ t('home.providers.more') }}</span>
              <span class="rounded bg-white/10 px-1.5 py-0.5 text-[10px] font-medium text-white/40">{{ t('home.providers.soon') }}</span>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="relative z-10 border-t border-white/[0.06] px-8 py-6">
      <div class="mx-auto flex max-w-7xl flex-col items-center justify-between gap-3 text-center sm:flex-row">
        <p class="text-xs text-white/25">&copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</p>
        <div class="flex items-center gap-4">
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="text-xs text-white/25 transition-colors hover:text-white/60">{{ t('home.docs') }}</a>
          <a :href="githubUrl" target="_blank" rel="noopener noreferrer" class="text-xs text-white/25 transition-colors hover:text-white/60">GitHub</a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, defineAsyncComponent } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'

const GlobeScene = defineAsyncComponent(() => import('@/components/home/GlobeScene.vue'))

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const paymentEnabled = computed(() => appStore.cachedPublicSettings?.payment_enabled === true)
const rechargeMultiplierDisplay = computed(() => {
  const raw = Number(appStore.cachedPublicSettings?.payment_balance_recharge_multiplier ?? 1)
  const multiplier = Number.isFinite(raw) && raw > 0 ? raw : 1
  return multiplier.toFixed(8).replace(/\.?0+$/, '')
})

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

// Theme
const isDark = ref(document.documentElement.classList.contains('dark'))

// GitHub URL
const githubUrl = 'https://github.com/HeartEase1/aipc'

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

// Toggle theme
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

// Initialize theme
function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()

  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
@keyframes shimmer {
  0%   { transform: translateX(-100%); }
  100% { transform: translateX(200%); }
}

/* Hero入场动画 — "淡出倒放"：元素从模糊放大状态收缩清晰落位 */
@keyframes revealIn {
  from { opacity: 0; transform: translateY(-18px) scale(1.06); filter: blur(6px); }
  to   { opacity: 1; transform: translateY(0)     scale(1);    filter: blur(0);   }
}
@keyframes fadeIn {
  from { opacity: 0; }
  to   { opacity: 1; }
}

/* 地球：轻微横向落位；缩放入场由 Three.js 相机完成，避免容器被重复放大裁切。 */
@keyframes globeReveal {
  0%   { transform: translateX(10%); opacity: 0;   }
  18%  { opacity: 1; }
  100% { transform: translateX(0); opacity: 1;   }
}

.hero-badge   { animation: revealIn 0.65s cubic-bezier(0.16,1,0.3,1) both; animation-delay: 0.08s; }
.hero-title   { animation: revealIn 0.65s cubic-bezier(0.16,1,0.3,1) both; animation-delay: 0.22s; }
.hero-sub     { animation: revealIn 0.65s cubic-bezier(0.16,1,0.3,1) both; animation-delay: 0.36s; }
.hero-pills   { animation: revealIn 0.60s cubic-bezier(0.16,1,0.3,1) both; animation-delay: 0.48s; }
.hero-actions { animation: revealIn 0.60s cubic-bezier(0.16,1,0.3,1) both; animation-delay: 0.60s; }

/* 桌面端：地球从中央放大态收缩入位 */
@media (min-width: 1024px) {
  .hero-globe {
    animation: globeReveal 2.2s cubic-bezier(0.16,1,0.3,1) both;
    animation-delay: 0.05s;
  }
}
/* 移动端/平板：简单淡入即可 */
@media (max-width: 1023px) {
  .hero-globe {
    animation: fadeIn 1.0s ease both;
    animation-delay: 0.15s;
  }
}
</style>
