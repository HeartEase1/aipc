<template>
  <button
    type="button"
    class="channel-health-card group w-full text-left"
    :aria-label="t('channelMonitorV2.overview.openDetailsFor', { name: displayName })"
    @click="emit('select')"
  >
    <div class="flex min-w-0 items-start gap-3">
      <span
        class="grid h-11 w-11 shrink-0 place-items-center rounded-xl ring-1 ring-black/5 dark:ring-white/10"
        :class="[providerGradient(row.platform), providerTintClass]"
      >
        <ProviderIcon :provider="row.platform" :size="22" />
      </span>

      <div class="min-w-0 flex-1">
        <div class="flex min-w-0 items-center gap-2">
          <h3 class="truncate text-[15px] font-bold text-gray-900 dark:text-white">
            {{ displayName }}
          </h3>
          <span
            v-if="rate != null"
            class="shrink-0 rounded-md bg-gray-100 px-1.5 py-0.5 text-[10px] font-bold tabular-nums text-gray-500 dark:bg-dark-700 dark:text-gray-300"
          >
            {{ formatRate(rate) }}
          </span>
        </div>
        <span
          class="mt-1 inline-flex max-w-full items-center rounded-md px-1.5 py-0.5 text-[10px] font-semibold"
          :class="providerBadgeClass(row.platform)"
        >
          {{ providerLabel(row.platform) }}
        </span>
      </div>

      <span
        class="inline-flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-bold"
        :class="statusTone"
      >
        <i class="h-1.5 w-1.5 rounded-full bg-current" aria-hidden="true" />
        {{ statusLabel }}
      </span>
    </div>

    <div class="mt-5 grid grid-cols-3 divide-x divide-gray-100 dark:divide-dark-700">
      <div class="min-w-0 pr-3">
        <p class="truncate text-[11px] font-medium text-gray-400 dark:text-gray-500">
          {{ t('channelMonitorV2.metrics.successRate') }}
        </p>
        <p class="mt-1 text-base font-black tabular-nums text-gray-900 dark:text-white">
          {{ successRate }}
        </p>
      </div>
      <div class="min-w-0 px-3">
        <p class="truncate text-[11px] font-medium text-gray-400 dark:text-gray-500">
          {{ t('channelMonitorV2.metrics.cacheRate') }}
        </p>
        <p class="mt-1 text-base font-black tabular-nums text-gray-900 dark:text-white">
          {{ cacheRate }}
        </p>
      </div>
      <div class="min-w-0 pl-3">
        <p class="truncate text-[11px] font-medium text-gray-400 dark:text-gray-500">
          {{ t('channelMonitorV2.metrics.ttftP50') }}
        </p>
        <p class="mt-1 text-base font-black tabular-nums text-gray-900 dark:text-white">
          {{ ttft }}
        </p>
      </div>
    </div>

    <div class="mt-5 border-t border-gray-100 pt-4 dark:border-dark-700/80">
      <div class="mb-2 flex items-center justify-between gap-3 text-[11px] font-medium text-gray-400 dark:text-gray-500">
        <span>{{ t('channelMonitorV2.overview.recentTrend') }}</span>
        <span class="inline-flex items-center gap-1 text-gray-500 transition-colors group-hover:text-primary-600 dark:text-gray-400 dark:group-hover:text-primary-300">
          {{ t('channelMonitorV2.overview.viewDetails') }}
          <Icon name="chevronRight" size="xs" />
        </span>
      </div>
      <ChannelHealthTimeline :buckets="row.buckets" />
    </div>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import ProviderIcon from '@/components/user/monitor/ProviderIcon.vue'
import { providerGradient, useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'
import type { MonitorMatrixRow } from '@/api/channelMonitorV2'
import { formatMonitorMs, formatMonitorPercent } from './monitorFormat'
import ChannelHealthTimeline from './ChannelHealthTimeline.vue'

const props = defineProps<{
  row: MonitorMatrixRow
  rate?: number | null
}>()

const emit = defineEmits<{
  (event: 'select'): void
}>()

const { t } = useI18n()
const { providerBadgeClass, providerLabel } = useChannelMonitorFormat()

const displayName = computed(() => props.row.group_name || `#${props.row.group_id || '-'}`)
const hasTraffic = computed(() => props.row.metrics.request_count > 0)
const state = computed(() => (hasTraffic.value ? props.row.health.overall : 'unknown'))
const successRate = computed(() =>
  hasTraffic.value ? formatMonitorPercent(1 - props.row.metrics.error_rate) : '-',
)
const cacheRate = computed(() =>
  hasTraffic.value ? formatMonitorPercent(props.row.metrics.cache_rate) : '-',
)
const ttft = computed(() => formatMonitorMs(props.row.metrics.ttft.p50_ms))

const statusLabel = computed(() => t(`channelMonitorV2.overview.status.${state.value}`))
const statusTone = computed(() => {
  if (state.value === 'healthy') {
    return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
  }
  if (state.value === 'warning') {
    return 'bg-amber-50 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
  }
  if (state.value === 'critical') {
    return 'bg-red-50 text-red-700 dark:bg-red-500/15 dark:text-red-300'
  }
  return 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-400'
})

const providerTintClass = computed(() => {
  const tones: Record<string, string> = {
    openai: 'text-emerald-600 dark:text-emerald-300',
    anthropic: 'text-orange-600 dark:text-orange-300',
    gemini: 'text-sky-600 dark:text-sky-300',
    grok: 'text-zinc-700 dark:text-zinc-200',
    antigravity: 'text-purple-600 dark:text-purple-300',
    kimi: 'text-pink-600 dark:text-pink-300',
    zhipu: 'text-indigo-600 dark:text-indigo-300',
    deepseek: 'text-teal-600 dark:text-teal-300',
  }
  return tones[props.row.platform] || 'text-gray-600 dark:text-gray-300'
})

function formatRate(value: number) {
  return `×${new Intl.NumberFormat(undefined, { maximumFractionDigits: 3 }).format(value)}`
}
</script>

<style scoped>
.channel-health-card {
  position: relative;
  display: block;
  min-height: 15rem;
  overflow: hidden;
  border: 1px solid rgb(229 231 235 / 0.82);
  border-radius: 1rem;
  background: rgb(255 255 255 / 0.86);
  padding: 1.125rem;
  box-shadow: 0 1px 2px rgb(15 23 42 / 0.04), 0 10px 26px rgb(15 23 42 / 0.045);
  backdrop-filter: blur(14px);
  transition: border-color 180ms ease, box-shadow 180ms ease, transform 180ms ease;
}

.channel-health-card::before {
  position: absolute;
  inset: 0 0 auto;
  height: 3px;
  content: '';
  background: linear-gradient(90deg, rgb(var(--color-primary-400) / 0.85), rgb(56 189 248 / 0.55));
  opacity: 0;
  transition: opacity 180ms ease;
}

.channel-health-card:hover {
  border-color: rgb(var(--color-primary-200) / 0.9);
  box-shadow: 0 16px 36px rgb(15 23 42 / 0.1);
  transform: translateY(-2px);
}

.channel-health-card:hover::before,
.channel-health-card:focus-visible::before {
  opacity: 1;
}

.channel-health-card:focus-visible {
  outline: 3px solid rgb(var(--color-primary-300) / 0.4);
  outline-offset: 2px;
}

:global(.dark) .channel-health-card {
  border-color: rgb(51 65 85 / 0.86);
  background: rgb(30 41 59 / 0.8);
  box-shadow: 0 10px 28px rgb(0 0 0 / 0.18);
}

:global(.dark) .channel-health-card:hover {
  border-color: rgb(var(--color-primary-500) / 0.45);
  box-shadow: 0 16px 38px rgb(0 0 0 / 0.28);
}

@media (prefers-reduced-motion: reduce) {
  .channel-health-card,
  .channel-health-card::before {
    transition: none;
  }

  .channel-health-card:hover {
    transform: none;
  }
}
</style>
