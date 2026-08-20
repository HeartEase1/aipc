import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import SubscriptionProgressMini from '../SubscriptionProgressMini.vue'

const { subscriptionStore } = vi.hoisted(() => ({
  subscriptionStore: {
    activeSubscriptions: [] as Array<Record<string, unknown>>,
    hasActiveSubscriptions: false,
    fetchActiveSubscriptions: vi.fn(),
  },
}))

vi.mock('@/stores', () => ({
  useSubscriptionStore: () => subscriptionStore,
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        key === 'userSubscriptions.usedPercentage'
          ? `used ${params?.percentage}%`
          : key === 'subscriptionProgress.used'
            ? 'used'
          : key,
    }),
  }
})

describe('SubscriptionProgressMini', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    subscriptionStore.fetchActiveSubscriptions.mockResolvedValue([])
    subscriptionStore.hasActiveSubscriptions = true
    subscriptionStore.activeSubscriptions = [{
      id: 1,
      group_id: 10,
      daily_usage_usd: 3.2,
      weekly_usage_usd: 12,
      monthly_usage_usd: 45,
      expires_at: '2099-09-01T00:00:00Z',
      group: {
        id: 10,
        name: 'Pro plan',
        daily_limit_usd: 10,
        weekly_limit_usd: 50,
        monthly_limit_usd: 100,
      },
    }]
  })

  it('uses percentages for quick preview instead of requiring users to calculate from amounts', async () => {
    const wrapper = mount(SubscriptionProgressMini, {
      global: {
        stubs: {
          Icon: true,
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })
    await flushPromises()

    await wrapper.get('button').trigger('click')

    expect(wrapper.text()).toContain('used32%')
    expect(wrapper.text()).toContain('used24%')
    expect(wrapper.text()).toContain('used45%')
    expect(wrapper.text()).not.toContain('$3.20/$10.00')
    expect(wrapper.text()).not.toContain('$12.00/$50.00')
    expect(wrapper.text()).not.toContain('$45.00/$100.00')

    wrapper.unmount()
  })
})
