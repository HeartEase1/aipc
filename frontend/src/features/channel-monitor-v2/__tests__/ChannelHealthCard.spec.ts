import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { MonitorCoverage, MonitorMatrixRow } from '@/api/channelMonitorV2'

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({
    locale: ref('zh-CN'),
    t: (key: string, params?: Record<string, unknown>) => {
      const labels: Record<string, string> = {
        'monitorCommon.providers.openai': 'OpenAI',
        'channelMonitorV2.metrics.successRate': '成功率',
        'channelMonitorV2.metrics.windowSuccessRate': '区间成功率',
        'channelMonitorV2.metrics.cacheRate': '缓存率',
        'channelMonitorV2.metrics.windowCacheRate': '区间缓存率',
        'channelMonitorV2.metrics.ttftP50': '首 Token P50',
        'channelMonitorV2.overview.recentTrend': '近期状态',
        'channelMonitorV2.overview.viewDetails': '查看详情',
        'channelMonitorV2.overview.status.healthy': '运行良好',
        'channelMonitorV2.overview.status.unknown': '暂无数据',
        'channelMonitorV2.overview.status.insufficient': '样本不足',
      }
      if (key === 'channelMonitorV2.overview.openDetailsFor') return `查看 ${params?.name} 的详细分析`
      if (key === 'channelMonitorV2.overview.noTrafficAt') return `${params?.time} · 暂无流量`
      if (key === 'channelMonitorV2.overview.insufficientSamplesAt') return `${params?.time} · 有流量，样本不足`
      if (key === 'channelMonitorV2.metrics.successRateValue') return `成功率 ${params?.value}`
      if (key === 'channelMonitorV2.metrics.healthSuccessRateValue') return `健康口径成功率 ${params?.value}`
      if (key === 'channelMonitorV2.metrics.cacheRateValue') return `缓存率 ${params?.value}`
      if (key === 'channelMonitorV2.metrics.ttftValue') return `首 Token ${params?.value}`
      return labels[key] || key
    },
  }),
}))

import ChannelHealthCard from '../ChannelHealthCard.vue'
import ChannelHealthTimeline from '../ChannelHealthTimeline.vue'

function row(overrides: Partial<MonitorMatrixRow> = {}): MonitorMatrixRow {
  return {
    platform: 'openai',
    group_id: 12,
    group_name: '全球高速',
    metrics: {
      success_requests: 95,
      error_requests: 5,
      request_count: 100,
      has_traffic: true,
      token_count: 1000,
      rpm: 10,
      tpm: 600,
      error_rate: 0.05,
      success_rate: 0.95,
      cache_rate: 0.72,
      cache_rate_numerator: 720,
      cache_rate_denominator: 1000,
      ttft: { sample_count: 100, p50_ms: 420, p95_ms: 900, avg_ms: 500 },
      duration: { sample_count: 100, p50_ms: 1200, p95_ms: 2200, avg_ms: 1400 },
    },
    health: {
      overall: 'healthy',
      error_rate: 'healthy',
      ttft: 'healthy',
      cache: 'healthy',
      score: 92,
      minimum_sample: 5,
    },
    buckets: [],
    ...overrides,
  }
}

function coverage(
  requestedStart = '2026-08-30T00:00:00.000Z',
  requestedEnd = '2026-08-30T01:30:00.000Z',
  bucketSeconds = 300,
  coverageStart = requestedStart,
  dataThrough = requestedEnd,
): MonitorCoverage {
  return {
    requested_start: requestedStart,
    requested_end: requestedEnd,
    coverage_start: coverageStart,
    data_through: dataThrough,
    computed_at: dataThrough,
    aggregation_lag_seconds: 0,
    coverage_complete: coverageStart === requestedStart,
    bucket_seconds: bucketSeconds,
  }
}

describe('ChannelHealthCard', () => {
  beforeEach(() => vi.clearAllMocks())

  it('shows a concise group summary and emits selection', async () => {
    const wrapper = mount(ChannelHealthCard, {
      props: { row: row(), coverage: coverage(), rate: 1.25 },
      global: {
        stubs: {
          ProviderIcon: { template: '<span data-testid="provider-icon" />' },
          Icon: { template: '<span />' },
          ChannelHealthTimeline: { template: '<div data-testid="timeline" />' },
        },
      },
    })

    expect(wrapper.text()).toContain('全球高速')
    expect(wrapper.text()).toContain('OpenAI')
    expect(wrapper.text()).toContain('×1.25')
    expect(wrapper.text()).toContain('95.0%')
    expect(wrapper.text()).toContain('72.0%')
    expect(wrapper.text()).toContain('420ms')
    expect(wrapper.text()).toContain('运行良好')

    await wrapper.trigger('click')
    expect(wrapper.emitted('select')).toHaveLength(1)
  })

  it('does not present zero traffic as successful health data', () => {
    const base = row()
    const emptyRow = row({
      metrics: {
        ...base.metrics,
        success_requests: 0,
        error_requests: 0,
        request_count: 0,
        has_traffic: false,
        token_count: 0,
        rpm: 0,
        tpm: 0,
        error_rate: 0,
        success_rate: 0,
        cache_rate: 0,
        cache_rate_numerator: 0,
        cache_rate_denominator: 0,
        ttft: { sample_count: 0, p50_ms: null, p95_ms: null, avg_ms: null },
        duration: { sample_count: 0, p50_ms: null, p95_ms: null, avg_ms: null },
      },
      health: { ...base.health, overall: 'unknown', score: undefined },
    })
    const wrapper = mount(ChannelHealthCard, {
      props: { row: emptyRow, coverage: coverage() },
      global: {
        stubs: {
          ProviderIcon: true,
          Icon: true,
          ChannelHealthTimeline: true,
        },
      },
    })

    expect(wrapper.text()).toContain('暂无数据')
    expect(wrapper.text()).not.toContain('100.0%')
  })

  it('keeps health ratios visible when the user payload redacts request count', () => {
    const redactedRow = row({
      metrics: { ...row().metrics, request_count: 0, has_traffic: true },
    })
    const wrapper = mount(ChannelHealthCard, {
      props: { row: redactedRow, coverage: coverage() },
      global: {
        stubs: {
          ProviderIcon: true,
          Icon: true,
          ChannelHealthTimeline: true,
        },
      },
    })

    expect(wrapper.text()).toContain('运行良好')
    expect(wrapper.text()).toContain('95.0%')
    expect(wrapper.text()).toContain('72.0%')
    expect(wrapper.text()).toContain('420ms')
  })

  it('does not borrow a green bucket when aggregate health is still insufficient', () => {
    const base = row()
    const bucketedRow = row({
      health: { ...base.health, overall: 'unknown' },
      buckets: [{
        bucket_start: '2026-08-30T00:05:00.000Z',
        metrics: { ...base.metrics, request_count: 0, has_traffic: true },
        health: base.health,
      }],
    })
    const wrapper = mount(ChannelHealthCard, {
      props: { row: bucketedRow, coverage: coverage() },
      global: {
        stubs: {
          ProviderIcon: true,
          Icon: true,
          ChannelHealthTimeline: true,
        },
      },
    })

    expect(wrapper.text()).toContain('样本不足')
    expect(wrapper.text()).not.toContain('运行良好')
    expect(wrapper.text()).toContain('95.0%')
  })

  it('renders as a non-interactive summary when detailed analysis is disabled', async () => {
    const wrapper = mount(ChannelHealthCard, {
      props: { row: row(), coverage: coverage(), interactive: false },
      global: {
        stubs: {
          ProviderIcon: true,
          Icon: true,
          ChannelHealthTimeline: true,
        },
      },
    })

    expect(wrapper.element.tagName).toBe('ARTICLE')
    expect(wrapper.text()).not.toContain('查看详情')
    await wrapper.trigger('click')
    expect(wrapper.emitted('select')).toBeUndefined()
  })
})

describe('ChannelHealthTimeline', () => {
  it.each([
    ['90m', '2026-08-30T00:00:00.000Z', '2026-08-30T01:30:00.000Z', 300],
    ['24h', '2026-08-30T00:00:00.000Z', '2026-08-31T00:00:00.000Z', 3600],
    ['7d', '2026-08-24T00:00:00.000Z', '2026-08-31T00:00:00.000Z', 43200],
    ['30d', '2026-08-01T00:00:00.000Z', '2026-08-31T00:00:00.000Z', 86400],
  ])('hides empty %s intervals instead of drawing placeholder blocks', (_, start, end, seconds) => {
    const wrapper = mount(ChannelHealthTimeline, {
      props: { buckets: [], coverage: coverage(start, end, seconds) },
    })

    expect(wrapper.findAll('.health-timeline__bar')).toHaveLength(0)
    expect(wrapper.attributes('style')).toContain('grid-template-columns: none')
  })

  it('keeps every traffic bucket in chronological order without empty gaps', () => {
    const base = row()
    const buckets = [20, 5].map((minute) => ({
      bucket_start: `2026-08-30T00:${String(minute).padStart(2, '0')}:00.000Z`,
      metrics: base.metrics,
      health: base.health,
    })).reverse()
    const wrapper = mount(ChannelHealthTimeline, {
      props: {
        buckets,
        coverage: coverage(
          '2026-08-30T00:00:00.000Z',
          '2026-08-30T00:30:00.000Z',
          300,
        ),
      },
    })
    const bars = wrapper.findAll('.health-timeline__bar')

    expect(bars).toHaveLength(2)
    expect(bars[0].attributes('title').localeCompare(bars[1].attributes('title'))).toBeLessThan(0)
    expect(wrapper.findAll('.health-timeline__missing')).toHaveLength(0)
  })

  it('ignores buckets outside historical coverage, in the future, or off-grid', () => {
    const base = row()
    const bucket = (bucketStart: string) => ({
      bucket_start: bucketStart,
      metrics: base.metrics,
      health: base.health,
    })
    const wrapper = mount(ChannelHealthTimeline, {
      props: {
        buckets: [
          bucket('2026-08-29T23:55:00.000Z'),
          bucket('2026-08-30T00:05:00.000Z'),
          bucket('2026-08-30T00:20:00.000Z'),
          bucket('2026-08-30T00:16:00.000Z'),
          bucket('2026-08-30T00:25:00.000Z'),
          bucket('2026-08-30T00:30:00.000Z'),
        ],
        coverage: coverage(
          '2026-08-30T00:00:00.000Z',
          '2026-08-30T00:30:00.000Z',
          300,
          '2026-08-30T00:10:00.000Z',
          '2026-08-30T00:25:00.000Z',
        ),
      },
    })
    const bars = wrapper.findAll('.health-timeline__bar')

    expect(bars).toHaveLength(1)
    expect(bars[0].classes()).toContain('health-score9')
    expect(bars[0].attributes('title')).toContain('成功率')
  })

  it('caps malformed high-density traffic while keeping the newest 48 buckets', () => {
    const base = row()
    const buckets = Array.from({ length: 60 }, (_, index) => ({
      bucket_start: new Date(Date.UTC(2026, 7, 30, 0, index)).toISOString(),
      metrics: { ...base.metrics, cache_rate: index / 100 },
      health: base.health,
    }))
    const wrapper = mount(ChannelHealthTimeline, {
      props: {
        buckets,
        coverage: coverage(
          '2026-08-30T00:00:00.000Z',
          '2026-08-30T01:00:00.000Z',
          60,
        ),
      },
    })

    expect(wrapper.findAll('.health-timeline__bar')).toHaveLength(48)
    expect(wrapper.attributes('style')).toContain('repeat(48, minmax(4px, 1fr))')
    expect((wrapper.findAll('.health-timeline__bar')[0].element as HTMLElement).style.height).toBe('28%')
  })

  it('uses cache rate to encode bar height without random decoration', () => {
    const base = row()
    const buckets = [0.2, 0.9].map((cacheRate, index) => ({
      bucket_start: new Date(Date.UTC(2026, 7, 30, 0, index * 5)).toISOString(),
      metrics: { ...base.metrics, cache_rate: cacheRate },
      health: base.health,
    }))
    const wrapper = mount(ChannelHealthTimeline, {
      props: {
        buckets,
        coverage: coverage(
          '2026-08-30T00:00:00.000Z',
          '2026-08-30T00:10:00.000Z',
          300,
        ),
      },
    })
    const bars = wrapper.findAll('.health-timeline__bar')
    const lowHeight = Number.parseFloat((bars[0].element as HTMLElement).style.height)
    const highHeight = Number.parseFloat((bars[1].element as HTMLElement).style.height)

    expect(lowHeight).toBeLessThan(highHeight)
    expect(highHeight).toBeLessThanOrEqual(100)
  })

  it('keeps cache height when user payload redacts request count', () => {
    const base = row()
    const wrapper = mount(ChannelHealthTimeline, {
      props: {
        coverage: coverage(
          '2026-08-30T00:00:00.000Z',
          '2026-08-30T00:05:00.000Z',
          300,
        ),
        buckets: [{
          bucket_start: new Date(Date.UTC(2026, 7, 30)).toISOString(),
          metrics: { ...base.metrics, request_count: 0, has_traffic: true, cache_rate: 0.9 },
          health: { ...base.health, overall: 'healthy', score: undefined },
        }],
      },
    })
    const bar = wrapper.findAll('.health-timeline__bar').at(-1)

    expect((bar?.element as HTMLElement).style.height).toBe('92%')
    expect(bar?.classes()).toContain('health-healthy')
    expect(bar?.attributes('title')).toContain('缓存率 90.0%')
    expect(bar?.attributes('title')).not.toContain('暂无流量')
  })

  it('renders low-sample traffic distinctly and explains both success-rate semantics', () => {
    const base = row()
    const wrapper = mount(ChannelHealthTimeline, {
      props: {
        coverage: coverage(
          '2026-08-30T00:00:00.000Z',
          '2026-08-30T00:05:00.000Z',
          300,
        ),
        buckets: [{
          bucket_start: '2026-08-30T00:00:00.000Z',
          metrics: {
            ...base.metrics,
            request_count: 0,
            has_traffic: true,
            success_rate: 0.33,
            error_rate: 0,
            cache_rate: 0.6,
          },
          health: { ...base.health, overall: 'unknown', score: undefined },
        }],
      },
    })
    const bar = wrapper.find('.health-timeline__bar')

    expect(bar.classes()).toContain('health-insufficient')
    expect((bar.element as HTMLElement).style.height).toBe('67%')
    expect(bar.attributes('title')).toContain('有流量，样本不足')
    expect(bar.attributes('title')).toContain('成功率 33.0%')
    expect(bar.attributes('title')).toContain('健康口径成功率 100.0%')
  })
})
