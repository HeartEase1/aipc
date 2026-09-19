<template>
  <LegacyAppShell v-if="consoleUiMode === 'legacy'" ref="shellRef">
    <slot />
  </LegacyAppShell>
  <ModernAppShell v-else ref="shellRef">
    <slot />
  </ModernAppShell>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useAppStore } from '@/stores'
import LegacyAppShell from './LegacyAppShell.vue'
import ModernAppShell from './ModernAppShell.vue'

const appStore = useAppStore()

// The public setting is intentionally read without making the layout depend
// on the settings API shape during bootstrap. Missing and unknown values are
// conservative: the existing console remains active until modern is explicit.
const consoleUiMode = computed<'legacy' | 'modern'>(() => {
  const mode = (appStore.cachedPublicSettings as { console_ui_mode?: unknown } | null)?.console_ui_mode
  return mode === 'modern' ? 'modern' : 'legacy'
})

const shellRef = ref<InstanceType<typeof LegacyAppShell> | InstanceType<typeof ModernAppShell> | null>(null)

function replayTour() {
  shellRef.value?.replayTour?.()
}

defineExpose({ replayTour })
</script>
