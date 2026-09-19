<template>
  <div
    class="modern-console-shell"
    :class="{ 'modern-sidebar-collapsed': sidebarCollapsed }"
    data-console-shell="modern"
  >
    <div class="modern-console-backdrop" aria-hidden="true"></div>

    <AppSidebar />

    <div class="modern-console-content">
      <AppHeader />

      <main class="modern-console-main">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import '@/styles/modern-console.css'
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)

  // Portals (dialogs, selects, announcements and toasts) are teleported to
  // body. Marking the document lets the modern token layer style those
  // surfaces without changing the legacy shell or the public landing page.
  if (typeof document !== 'undefined') {
    document.documentElement.dataset.consoleUiMode = 'modern'
  }
})

onBeforeUnmount(() => {
  if (typeof document !== 'undefined' && document.documentElement.dataset.consoleUiMode === 'modern') {
    delete document.documentElement.dataset.consoleUiMode
  }
})

defineExpose({ replayTour })
</script>
