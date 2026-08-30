import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { MonitorMatrixRow } from '@/api/channelMonitorV2'

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({
    locale: ref('zh-CN'),
    t: (key: string, params?: Record<string, unknown>) => {
      const labels: Record<string, string> = {
        'monitorCommon.providers.openai': 'OpenAI',
        'channelMonitorV2.metrics.successRate': '成功率',
        'channelMonitorV2.metrics.cacheRate': '缓存率',
        'channelMonitorV2.metrics.ttftP50': '首 Token P50',
        'channelMonitorV2.overview.recentTrend': '近期状态',
        'channelMonitorV2.overview.viewDetails': '查看详情',
        'channelMonitorV2.overview.status.healthy': '运行良好',
        'channelMonitorV2.overview.status.unknown': '暂无数据',
      }
      if (key === 'channelMonitorV2.overview.openDetailsFor') return `查看 ${params?.name} 的详细分析`
      if (key === 'channelMonitorV2.overview.noTrafficAt') return `${params?.time} · 暂无流量`
      if (key === 'channelMonitorV2.metrics.successRateValue') return `成功率 ${params?.value}`
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
      token_count: 1000,
      rpm: 10,
      tpm: 600,
      error_rate: 0.05,
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

describe('ChannelHealthCard', () => {
  beforeEach(() => vi.clearAllMocks())

  it('shows a concise group summary and emits selection', async () => {
    const wrapper = mount(ChannelHealthCard, {
      props: { row: row(), rate: 1.25 },
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
    const emptyRow = row({
      metrics: { ...row().metrics, request_count: 0, error_rate: 0, cache_rate: 0 },
      health: { ...row().health, overall: 'healthy' },
    })
    const wrapper = mount(ChannelHealthCard, {
      props: { row: emptyRow },
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

  it('renders as a non-interactive summary when detailed analysis is disabled', async () => {
    const wrapper = mount(ChannelHealthCard, {
      props: { row: row(), interactive: false },
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
  it('caps the compact pulse at the latest 24 buckets', () => {
    const base = row()
    const buckets = Array.from({ length: 30 }, (_, index) => ({
      bucket_start: new Date(Date.UTC(2026, 7, 30, 0, index)).toISOString(),
      metrics: base.metrics,
      health: base.health,
    }))
    const wrapper = mount(ChannelHealthTimeline, { props: { buckets } })

    expect(wrapper.findAll('.health-timeline__bar')).toHaveLength(24)
    expect(wrapper.find('.health-score9').exists()).toBe(true)
  })

  it('keeps sparse data in fixed-width 24-slot intervals', () => {
    const base = row()
    const buckets = Array.from({ length: 2 }, (_, index) => ({
      bucket_start: new Date(Date.UTC(2026, 7, 30, 0, index)).toISOString(),
      metrics: base.metrics,
      health: base.health,
    }))
    const wrapper = mount(ChannelHealthTimeline, { props: { buckets } })

    expect(wrapper.findAll('.health-timeline__bar')).toHaveLength(24)
    expect(wrapper.findAll('.health-timeline__missing')).toHaveLength(22)
  })

  it('uses cache rate to encode bar height without random decoration', () => {
    const base = row()
    const buckets = [0.2, 0.9].map((cacheRate, index) => ({
      bucket_start: new Date(Date.UTC(2026, 7, 30, 0, index)).toISOString(),
      metrics: { ...base.metrics, cache_rate: cacheRate },
      health: base.health,
    }))
    const wrapper = mount(ChannelHealthTimeline, { props: { buckets } })
    const bars = wrapper.findAll('.health-timeline__bar').slice(-2)
    const lowHeight = Number.parseFloat((bars[0].element as HTMLElement).style.height)
    const highHeight = Number.parseFloat((bars[1].element as HTMLElement).style.height)

    expect(lowHeight).toBeLessThan(highHeight)
    expect(highHeight).toBeLessThanOrEqual(100)
  })

  it('keeps cache height when user payload redacts request count', () => {
    const base = row()
    const wrapper = mount(ChannelHealthTimeline, {
      props: {
        buckets: [{
          bucket_start: new Date(Date.UTC(2026, 7, 30)).toISOString(),
          metrics: { ...base.metrics, request_count: 0, cache_rate: 0.9 },
          health: { ...base.health, overall: 'healthy' },
        }],
      },
    })
    const bar = wrapper.findAll('.health-timeline__bar').at(-1)

    expect((bar?.element as HTMLElement).style.height).toBe('92%')
    expect(bar?.attributes('title')).toContain('缓存率 90.0%')
    expect(bar?.attributes('title')).not.toContain('暂无流量')
  })
})
