import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import RedeemView from '../RedeemView.vue'
import BenefitGrantHistory from '@/components/user/BenefitGrantHistory.vue'

const { redeem, getHistory, refreshUser, fetchActiveSubscriptions, showError, showWarning, showSuccess } = vi.hoisted(() => ({
  redeem: vi.fn(),
  getHistory: vi.fn(),
  refreshUser: vi.fn(),
  fetchActiveSubscriptions: vi.fn(),
  showError: vi.fn(),
  showWarning: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api', () => ({
  redeemAPI: { redeem, getHistory },
  authAPI: { getPublicSettings: vi.fn().mockResolvedValue({}) },
}))
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ user: { balance: 10, concurrency: 2 }, refreshUser }),
}))
vi.mock('@/stores/subscriptions', () => ({
  useSubscriptionStore: () => ({ fetchActiveSubscriptions }),
}))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showWarning, showSuccess }),
}))
vi.mock('@/api/benefitGrants', () => ({
  default: { list: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 }) },
}))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

async function submitCode() {
  const wrapper = mount(RedeemView, {
    global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, Icon: true } },
  })
  await flushPromises()
  await wrapper.get('input#code').setValue(' REDEEM-CODE ')
  await wrapper.get('form').trigger('submit')
  await flushPromises()
  return wrapper
}

describe('RedeemView refresh after redemption', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    redeem.mockResolvedValue({ type: 'balance', value: 20, message: 'Code applied' })
    getHistory.mockResolvedValue({ items: [], total: 0 })
    refreshUser.mockResolvedValue({ balance: 30, concurrency: 2 })
    fetchActiveSubscriptions.mockResolvedValue([])
    vi.spyOn(console, 'error').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it.each(['balance', 'concurrency', 'subscription'])(
    'keeps a successful %s redemption when profile refresh fails', async (type) => {
      redeem.mockResolvedValue({ type, value: 20, message: 'Code applied' })
      refreshUser.mockRejectedValue({ status: 503, message: 'Service unavailable' })
      getHistory.mockResolvedValueOnce({ items: [], total: 0 }).mockResolvedValueOnce({ total: 1, items: [{
        id: 1, code: 'REDEEM-CODE', type, value: 20, used_at: '2026-03-08T00:00:00Z',
      }] })

      const wrapper = await submitCode()

      expect(redeem).toHaveBeenCalledWith('REDEEM-CODE')
      expect(showError).not.toHaveBeenCalled()
      expect(showWarning).toHaveBeenCalledWith('redeem.userRefreshFailed')
      expect(showSuccess).toHaveBeenCalledWith('redeem.codeRedeemSuccess')
      expect(wrapper.text()).toContain('Code applied')
      expect(wrapper.text()).not.toContain('redeem.failedToRedeem')
      expect((wrapper.get('input#code').element as HTMLInputElement).value).toBe('')
      expect((wrapper.get('input#code').element as HTMLInputElement).disabled).toBe(false)
      expect(getHistory).toHaveBeenCalledTimes(2)
      expect(wrapper.text()).toContain('REDEEM-C...')
      if (type === 'subscription') {
        expect(fetchActiveSubscriptions).toHaveBeenCalledWith(true)
      } else {
        expect(fetchActiveSubscriptions).not.toHaveBeenCalled()
      }
      wrapper.unmount()
    }
  )

  it('pages on the server, changes size, and resets page and total after redeeming', async () => {
    getHistory.mockResolvedValue({ items: [], total: 101 })
    const wrapper = mount(RedeemView, {
      global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, Icon: true } },
    })
    await flushPromises()
    const activity = wrapper.getComponent(BenefitGrantHistory)
    expect(getHistory).toHaveBeenLastCalledWith(1, 20)
    expect(activity.props()).toMatchObject({ redeemPage: 1, redeemPageSize: 20, redeemTotal: 101 })
    activity.vm.$emit('update:redeemPage', 2)
    await flushPromises()
    expect(getHistory).toHaveBeenLastCalledWith(2, 20)
    activity.vm.$emit('update:redeemPage', 1)
    await flushPromises()
    expect(getHistory).toHaveBeenLastCalledWith(1, 20)
    activity.vm.$emit('update:redeemPageSize', 50)
    await flushPromises()
    expect(getHistory).toHaveBeenLastCalledWith(1, 50)
    activity.vm.$emit('update:redeemPageSize', 100)
    await flushPromises()
    activity.vm.$emit('update:redeemPage', 2)
    await flushPromises()
    expect(getHistory).toHaveBeenLastCalledWith(2, 100)
    expect(activity.props()).toMatchObject({ redeemPage: 2, redeemPageSize: 100, redeemTotal: 101 })
    getHistory.mockResolvedValue({ items: [], total: 102 })
    await wrapper.get('input#code').setValue('NEW-CODE')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(getHistory).toHaveBeenLastCalledWith(1, 100)
    expect(activity.props()).toMatchObject({ redeemPage: 1, redeemPageSize: 100, redeemTotal: 102 })
    wrapper.unmount()
  })

  it('restores the loaded size and keeps rows and navigation usable after a size request fails', async () => {
    const item = { id: 1, code: 'OLD-ROWS', type: 'balance', value: 20, used_at: '2026-03-08T00:00:00Z' }
    getHistory.mockResolvedValue({ items: [item], total: 61 })
    const wrapper = mount(RedeemView, {
      global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, Icon: true } },
    })
    await flushPromises()
    const activity = wrapper.getComponent(BenefitGrantHistory)
    activity.vm.$emit('update:redeemPage', 2)
    await flushPromises()
    let rejectRequest!: (error: Error) => void
    getHistory.mockImplementationOnce(() => new Promise((_resolve, reject) => { rejectRequest = reject }))
    activity.vm.$emit('update:redeemPageSize', 50)
    expect(getHistory).toHaveBeenLastCalledWith(1, 50)
    expect(activity.props('redeemPaginationDisabled')).toBe(false)
    rejectRequest(new Error('Network error'))
    await flushPromises()
    expect(activity.props()).toMatchObject({ redeemPage: 2, redeemPageSize: 20, redeemTotal: 61 })
    expect(wrapper.text()).toContain('OLD-ROWS')
    expect(showError).toHaveBeenCalledWith('redeem.historyLoadFailed')
    activity.vm.$emit('update:redeemPage', 3)
    await flushPromises()
    expect(getHistory).toHaveBeenLastCalledWith(3, 20)
    activity.vm.$emit('update:redeemPageSize', 50)
    await flushPromises()
    expect(getHistory).toHaveBeenLastCalledWith(1, 50)
    expect(activity.props()).toMatchObject({ redeemPage: 1, redeemPageSize: 50, redeemTotal: 61 })
    wrapper.unmount()
  })

  it.each(['success', 'failure'])('ignores a stale history %s after a newer size request succeeds', async (outcome) => {
    let resolveOld!: (value: unknown) => void
    let rejectOld!: (error: Error) => void
    getHistory.mockImplementationOnce(() => new Promise((resolve, reject) => {
      resolveOld = resolve
      rejectOld = reject
    }))
    const wrapper = mount(RedeemView, {
      global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, Icon: true } },
    })
    getHistory.mockResolvedValue({ items: [{
      id: 2, code: 'NEW-ROWS', type: 'balance', value: 30, used_at: '2026-03-08T00:00:00Z',
    }], total: 61 })
    // Force overlapping requests to exercise responses arriving out of order.
    const activity = wrapper.getComponent(BenefitGrantHistory)
    activity.vm.$emit('update:redeemPageSize', 50)
    await flushPromises()
    activity.vm.$emit('update:redeemPage', 2)
    await flushPromises()
    expect(getHistory).toHaveBeenLastCalledWith(2, 50)
    if (outcome === 'success') {
      resolveOld({ items: [], total: 0 })
    } else {
      rejectOld(new Error('Stale network error'))
    }
    await flushPromises()
    expect(activity.props()).toMatchObject({ redeemPage: 2, redeemPageSize: 50, redeemTotal: 61 })
    expect(wrapper.text()).toContain('NEW-ROWS')
    expect(showError).not.toHaveBeenCalled()
    activity.vm.$emit('update:redeemPage', 1)
    await flushPromises()
    expect(getHistory).toHaveBeenLastCalledWith(1, 50)
    wrapper.unmount()
  })

  it('hides redemption pagination for empty history', async () => {
    const wrapper = await submitCode()
    expect(wrapper.getComponent(BenefitGrantHistory).props('redeemTotal')).toBe(0)
    expect(wrapper.find('[data-testid="redeem-history-pagination"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('finishes normally without a warning when profile refresh succeeds', async () => {
    const wrapper = await submitCode()

    expect(refreshUser).toHaveBeenCalledOnce()
    expect(showWarning).not.toHaveBeenCalled()
    expect(showError).not.toHaveBeenCalled()
    expect(showSuccess).toHaveBeenCalledWith('redeem.codeRedeemSuccess')
    expect((wrapper.get('input#code').element as HTMLInputElement).value).toBe('')
    wrapper.unmount()
  })

  it('preserves the existing subscription refresh warning after successful redemption', async () => {
    redeem.mockResolvedValue({ type: 'subscription', value: 20, message: 'Code applied' })
    fetchActiveSubscriptions.mockRejectedValue(new Error('Network Error'))
    const wrapper = await submitCode()

    expect(showWarning).toHaveBeenCalledWith('redeem.subscriptionRefreshFailed')
    expect(showError).not.toHaveBeenCalled()
    expect(showSuccess).toHaveBeenCalledWith('redeem.codeRedeemSuccess')
    expect(getHistory).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })

  it('keeps the code and reports failure when the redemption request itself fails', async () => {
    redeem.mockRejectedValue({ response: { data: { detail: 'Invalid code' } } })
    const wrapper = await submitCode()

    expect(showError).toHaveBeenCalledWith('redeem.redeemFailed')
    expect(wrapper.text()).toContain('Invalid code')
    expect(wrapper.text()).not.toContain('Code applied')
    expect((wrapper.get('input#code').element as HTMLInputElement).value).toBe(' REDEEM-CODE ')
    expect(refreshUser).not.toHaveBeenCalled()
    expect(fetchActiveSubscriptions).not.toHaveBeenCalled()
    expect(getHistory).toHaveBeenCalledOnce()
    expect(showSuccess).not.toHaveBeenCalled()
    expect(showWarning).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
