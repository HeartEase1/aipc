import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { UserSubscription } from '@/types'
import SubscriptionsView from '../SubscriptionsView.vue'

const { getMySubscriptions, routerPush, showError } = vi.hoisted(() => ({
  getMySubscriptions: vi.fn(),
  routerPush: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/subscriptions', () => ({
  default: { getMySubscriptions },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    cachedPublicSettings: null,
    showError,
  }),
}))

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRouter: () => ({ push: routerPush }),
  }
})

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

function subscriptionFixture(
  id: number,
  groupId: number,
  dailyLimit: number | null,
): UserSubscription {
  return {
    id,
    user_id: 9,
    group_id: groupId,
    status: 'active',
    starts_at: '2026-08-01T00:00:00Z',
    expires_at: '2026-09-01T00:00:00Z',
    daily_usage_usd: 3,
    weekly_usage_usd: 0,
    monthly_usage_usd: 0,
    daily_window_start: '2026-08-16T00:00:00Z',
    weekly_window_start: null,
    monthly_window_start: null,
    created_at: '2026-08-01T00:00:00Z',
    updated_at: '2026-08-16T00:00:00Z',
    group: {
      id: groupId,
      name: `Group ${groupId}`,
      platform: 'openai',
      daily_limit_usd: dailyLimit,
      weekly_limit_usd: null,
      monthly_limit_usd: null,
      rate_multiplier: 1,
    },
  } as UserSubscription
}

describe('SubscriptionsView immediate reset entry', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getMySubscriptions.mockResolvedValue([
      subscriptionFixture(1, 10, 20),
      subscriptionFixture(2, 11, null),
    ])
  })

  it('shows reset only for quota-limited subscriptions and routes both actions explicitly', async () => {
    const wrapper = mount(SubscriptionsView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true,
        },
      },
    })
    await flushPromises()

    const renewButtons = wrapper.findAll('button').filter(button => button.text() === 'payment.renewNow')
    const restartButtons = wrapper.findAll('button').filter(button => button.text() === 'payment.restartNow')

    expect(renewButtons).toHaveLength(2)
    expect(restartButtons).toHaveLength(1)

    await renewButtons[0].trigger('click')
    expect(routerPush).toHaveBeenLastCalledWith({
      path: '/purchase',
      query: { tab: 'subscription', group: '10', action: 'extend' },
    })

    await restartButtons[0].trigger('click')
    expect(routerPush).toHaveBeenLastCalledWith({
      path: '/purchase',
      query: { tab: 'subscription', group: '10', action: 'restart' },
    })
  })
})
