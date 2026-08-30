<template>
  <div
    class="health-timeline"
    :aria-label="t('channelMonitorV2.overview.recentTrend')"
    :style="timelineStyle"
  >
    <span
      v-for="(slot, index) in timelineSlots"
      :key="slot ? `${slot.bucket_start}:${index}` : `empty:${index}`"
      class="health-timeline__bar"
      :class="slot ? bucketClass(slot) : 'health-timeline__missing'"
      :style="slot ? bucketStyle(slot) : undefined"
      :title="slot ? bucketTitle(slot) : undefined"
      :aria-hidden="slot ? undefined : 'true'"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { MonitorMatrixBucket } from '@/api/channelMonitorV2'
import {
  formatMonitorMs,
  formatMonitorPercent,
  healthScoreClass,
} from './monitorFormat'

const props = defineProps<{
  buckets: MonitorMatrixBucket[]
}>()

const { t, locale } = useI18n()

const visibleBuckets = computed(() => (props.buckets || []).slice(-24))
const timelineSlots = computed(() => [
  ...Array<null>(Math.max(0, 24 - visibleBuckets.value.length)).fill(null),
  ...visibleBuckets.value,
])
const timelineStyle = { gridTemplateColumns: 'repeat(24, minmax(0, 1fr))' }

function bucketClass(bucket: MonitorMatrixBucket) {
  return healthScoreClass(bucket.health, 'overall', bucket.metrics.request_count)
}

function bucketStyle(bucket: MonitorMatrixBucket) {
  if (bucket.health.overall === 'unknown' || !Number.isFinite(bucket.metrics.cache_rate)) {
    return { height: '18%' }
  }
  const cacheRate = Math.max(0, Math.min(1, bucket.metrics.cache_rate))
  return { height: `${Math.round(18 + cacheRate * 82)}%` }
}

function bucketTitle(bucket: MonitorMatrixBucket) {
  const time = new Intl.DateTimeFormat(locale.value || undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(bucket.bucket_start))
  if (bucket.health.overall === 'unknown') {
    return t('channelMonitorV2.overview.noTrafficAt', { time })
  }
  return [
    time,
    t('channelMonitorV2.metrics.successRateValue', {
      value: formatMonitorPercent(1 - bucket.metrics.error_rate),
    }),
    t('channelMonitorV2.metrics.cacheRateValue', {
      value: formatMonitorPercent(bucket.metrics.cache_rate),
    }),
    t('channelMonitorV2.metrics.ttftValue', {
      value: formatMonitorMs(bucket.metrics.ttft.p50_ms),
    }),
  ].join(' · ')
}
</script>

<style scoped>
.health-timeline {
  display: grid;
  align-items: end;
  gap: 3px;
  height: 2.25rem;
}

.health-timeline__bar {
  display: block;
  min-width: 2px;
  border-radius: 9999px;
  opacity: 0.82;
  transition: opacity 160ms ease, transform 160ms ease;
  transform-origin: bottom;
}

.health-timeline__bar:hover {
  opacity: 1;
  transform: translateY(-1px) scaleY(1.06);
}

.health-score10 { background: #16a34a; }
.health-score9  { background: #22c55e; }
.health-score8  { background: #4ade80; }
.health-score7  { background: #a3e635; }
.health-score6  { background: #facc15; }
.health-score5  { background: #fbbf24; }
.health-score4  { background: #f59e0b; }
.health-score3  { background: #f97316; }
.health-score2  { background: #fb7185; }
.health-score1  { background: #f87171; }
.health-score0,
.health-critical { background: #ef4444; }
.health-healthy { background: #22c55e; }
.health-warning { background: #f59e0b; }
.health-unknown { background: #d1d5db; }
.health-timeline__missing {
  height: 18%;
  background: rgb(226 232 240 / 0.72);
  opacity: 0.62;
}

:global(.dark) .health-timeline__missing,
:global(.dark) .health-unknown {
  background: #475569;
}

@media (prefers-reduced-motion: reduce) {
  .health-timeline__bar {
    transition: none;
  }
}
</style>
