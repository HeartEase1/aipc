<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, useId } from 'vue'

defineProps<{ label: string }>()
const id = useId()
const trigger = ref<HTMLButtonElement | null>(null)
const panel = ref<HTMLElement | null>(null)
const visible = ref(false)
const position = ref({ left: '0px', top: '0px' })

async function open() {
  visible.value = true
  await nextTick()
  if (!trigger.value || !panel.value || !visible.value) return
  const anchor = trigger.value.getBoundingClientRect()
  const box = panel.value.getBoundingClientRect()
  const left = Math.max(8, Math.min(anchor.left, window.innerWidth - box.width - 8))
  const below = anchor.bottom + 8
  const top = below + box.height <= window.innerHeight - 8 ? below : Math.max(8, anchor.top - box.height - 8)
  position.value = { left: `${left}px`, top: `${top}px` }
}
function close() { visible.value = false }
function onKey(event: KeyboardEvent) { if (event.key === 'Escape') close() }
onMounted(() => {
  window.addEventListener('scroll', close, true)
  window.addEventListener('resize', close)
  window.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => {
  window.removeEventListener('scroll', close, true)
  window.removeEventListener('resize', close)
  window.removeEventListener('keydown', onKey)
})
</script>

<template>
  <button ref="trigger" type="button" class="inline-flex items-center gap-1 rounded text-left focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500" :aria-label="label" :aria-describedby="visible ? id : undefined" @mouseenter="open" @mouseleave="close" @focus="open" @blur="close" @click.stop="open">
    <slot name="trigger" />
  </button>
  <Teleport to="body">
    <div v-if="visible" :id="id" ref="panel" role="tooltip" data-testid="usage-info-popover" class="pointer-events-none fixed z-[100000020] w-80 max-w-[calc(100vw-1rem)] rounded-xl border border-gray-200 bg-white p-4 text-xs leading-5 text-gray-700 shadow-xl dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200" :style="position">
      <slot />
    </div>
  </Teleport>
</template>
