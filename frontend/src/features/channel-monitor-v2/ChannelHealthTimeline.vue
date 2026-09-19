<template>
  <div
    class="health-timeline"
    :aria-label="t('channelMonitorV2.overview.recentTrend')"
    :style="timelineStyle"
  >
    <span
      v-for="slot in visibleSlots"
      :key="slot.start"
      class="health-timeline__bar"
      :class="slot.bucket ? bucketClass(slot.bucket) : 'health-timeline__missing'"
      :style="slot.bucket ? bucketStyle(slot.bucket) : { height: '12%' }"
      :title="slot.bucket ? bucketTitle(slot.bucket) : noTrafficTitle(slot.start)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { MonitorCoverage, MonitorMatrixBucket } from '@/api/channelMonitorV2'
import {
  formatMonitorMs,
  formatMonitorPercent,
  healthScoreClass,
  monitorMetricHasTraffic,
  monitorMetricHealthSuccessRate,
  monitorMetricSuccessRate,
  scoreToBand,
} from './monitorFormat'

const props = defineProps<{
  buckets: MonitorMatrixBucket[]
  coverage: MonitorCoverage
}>()

const { t, locale } = useI18n()

// Product ranges currently resolve to 14-30 buckets. This guard prevents a
// malformed payload from creating an unbounded number of compact DOM nodes.
const MAX_COMPACT_SLOTS = 48

type TimelineSlot = { start: string; bucket?: MonitorMatrixBucket }

const visibleSlots = computed<TimelineSlot[]>(() => {
  const requestedStart = Date.parse(props.coverage.requested_start)
  const requestedEnd = Date.parse(props.coverage.requested_end || props.coverage.data_through)
  const coverageStart = Date.parse(props.coverage.coverage_start)
  const dataThrough = Date.parse(props.coverage.data_through)
  const bucketSeconds = Number(props.coverage.bucket_seconds)

  if (
    !Number.isFinite(requestedStart) ||
    !Number.isFinite(requestedEnd) ||
    !Number.isFinite(bucketSeconds) ||
    bucketSeconds <= 0 ||
    requestedStart >= requestedEnd
  ) {
    return []
  }

  const coveredStart = Number.isFinite(coverageStart)
    ? Math.max(requestedStart, coverageStart)
    : requestedStart
  const coveredEnd = Number.isFinite(dataThrough)
    ? Math.min(requestedEnd, dataThrough)
    : requestedEnd
  const bucketStep = bucketSeconds * 1000

  const bucketsByStart = new Map<number, MonitorMatrixBucket>()
  for (const bucket of props.buckets || []) {
    const bucketStart = Date.parse(bucket.bucket_start)
    if (
      Number.isFinite(bucketStart) &&
      bucketStart >= coveredStart &&
      bucketStart < coveredEnd &&
      Math.abs(bucketStart - Math.round(bucketStart / bucketStep) * bucketStep) <= 1 &&
      monitorMetricHasTraffic(bucket.metrics)
    ) {
      bucketsByStart.set(Math.round(bucketStart / bucketStep) * bucketStep, bucket)
    }
  }

  const starts: number[] = []
  for (
    let start = Math.ceil(coveredStart / bucketStep) * bucketStep;
    start < coveredEnd;
    start += bucketStep
  ) {
    starts.push(start)
  }

  return starts.slice(-MAX_COMPACT_SLOTS).map((start) => ({
    start: new Date(start).toISOString(),
    bucket: bucketsByStart.get(start),
  }))
})

const timelineStyle = computed(() => {
  const count = visibleSlots.value.length
  if (count === 0) return { gridTemplateColumns: 'none' }
  const compact = count < 12
  return {
    gridTemplateColumns: compact
      ? `repeat(${count}, minmax(8px, 18px))`
      : `repeat(${count}, minmax(4px, 1fr))`,
    gap: '2px',
    justifyContent: compact ? 'end' : 'stretch',
  }
})

function bucketClass(bucket: MonitorMatrixBucket) {
  if (bucket.health.overall === 'unknown' && bucket.health.score == null) {
    return `health-${scoreToBand(monitorMetricSuccessRate(bucket.metrics) * 100)}`
  }
  return healthScoreClass(bucket.health, 'overall', Math.max(1, bucket.metrics.request_count))
}

function bucketStyle(bucket: MonitorMatrixBucket) {
  if (!Number.isFinite(bucket.metrics.cache_rate)) {
    return { height: '18%' }
  }
  const cacheRate = Math.max(0, Math.min(1, bucket.metrics.cache_rate))
  return { height: `${Math.round(18 + cacheRate * 82)}%` }
}

function bucketTitle(bucket: MonitorMatrixBucket) {
  const time = formatBucketTime(bucket.bucket_start)
  const successRate = monitorMetricSuccessRate(bucket.metrics)
  const healthSuccessRate = monitorMetricHealthSuccessRate(bucket.metrics)
  const lines = [
    time,
    t('channelMonitorV2.metrics.successRateValue', {
      value: formatMonitorPercent(successRate),
    }),
  ]
  if (Math.abs(successRate - healthSuccessRate) >= 0.0005) {
    lines.push(t('channelMonitorV2.metrics.healthSuccessRateValue', {
      value: formatMonitorPercent(healthSuccessRate),
    }))
  }
  lines.push(
    t('channelMonitorV2.metrics.cacheRateValue', {
      value: formatMonitorPercent(bucket.metrics.cache_rate),
    }),
    t('channelMonitorV2.metrics.ttftValue', {
      value: formatMonitorMs(bucket.metrics.ttft.p50_ms),
    }),
  )
  return lines.join(' · ')
}

function noTrafficTitle(start: string) {
  return t('channelMonitorV2.overview.noTrafficAt', { time: formatBucketTime(start) })
}

function formatBucketTime(value: string) {
  return new Intl.DateTimeFormat(locale.value || undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}
</script>

<style scoped>
.health-timeline {
  display: grid;
  align-items: end;
  gap: 2px;
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
  background: #cbd5e1;
  opacity: 0.9;
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
