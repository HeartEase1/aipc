import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import BenefitGrantsView from '../BenefitGrantsView.vue'

const { preview, showError, showSuccess } = vi.hoisted(() => ({
  preview: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    benefitGrants: {
      preview,
      execute: vi.fn(),
      list: vi.fn().mockResolvedValue({ items: [], total: 0 }),
      get: vi.fn(),
      retryFailed: vi.fn(),
      exportItems: vi.fn()
    },
    users: { list: vi.fn().mockResolvedValue({ items: [] }), getById: vi.fn() }
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ siteName: 'IPCAI', showError, showSuccess })
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} })
}))

vi.mock('@/composables/useStepUp', () => ({
  isStepUpCancelled: () => false,
  useStepUp: () => ({ run: (callback: () => unknown) => callback() })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const previewBatch = {
  id: 1,
  grant_type: 'welfare',
  grant_mode: 'fixed',
  audience_type: 'all',
  reason: 'reason',
  notification_title: 'title',
  notification_content: 'content',
  status: 'draft',
  eligible_count: 2,
  skipped_count: 0,
  success_count: 0,
  failed_count: 0,
  total_base_cost: '0',
  total_balance_base_cost: '0',
  total_subscription_base_cost: '0',
  total_amount: '2',
  total_balance_amount: '2',
  total_subscription_amount: '0',
  include_subscription: false,
  distributed_amount: '0',
  average_amount: '1',
  max_amount: '1',
  expires_at: '2026-07-28T00:10:00Z',
  created_at: '2026-07-28T00:00:00Z',
  updated_at: '2026-07-28T00:00:00Z',
  over_budget: false
}

function mountView() {
  return mount(BenefitGrantsView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        Icon: true,
        LoadingSpinner: true,
        Select: true,
        StatusBadge: true,
        Pagination: true,
        BaseDialog: { props: ['show'], template: '<section v-if="show"><slot /><slot name="footer" /></section>' },
        AnnouncementPopup: true,
        TotpStepUpDialog: true
      }
    }
  })
}

function buttonByText(wrapper: ReturnType<typeof mount>, text: string) {
  const button = wrapper.findAll('button').find((candidate) => candidate.text() === text)
  if (!button) throw new Error(`button not found: ${text}`)
  return button
}

describe('BenefitGrantsView', () => {
  beforeEach(() => {
    preview.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    preview.mockResolvedValue(previewBatch)
  })

  it('submits a fixed welfare preview with decimal strings', async () => {
    const wrapper = mountView()
    const amountInput = wrapper.get('input[placeholder="0.00000000"]')
    await amountInput.setValue('1.25000000')
    await wrapper.findAll('textarea')[0].setValue('welcome grant')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(preview).toHaveBeenCalledWith(expect.objectContaining({
      grant_type: 'welfare',
      grant_mode: 'fixed',
      audience_type: 'all',
      fixed_amount: '1.25000000',
      reason: 'welcome grant'
    }))
  })

  it('switches percentage mode to compensation and keeps percentage as a string', async () => {
    const wrapper = mountView()
    await buttonByText(wrapper, 'admin.benefitGrants.modes.percentage_24h').trigger('click')
    await wrapper.get('input[placeholder="10"]').setValue('25.5')
    await wrapper.findAll('textarea')[0].setValue('incident compensation')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(preview).toHaveBeenCalledWith(expect.objectContaining({
      grant_type: 'compensation',
      grant_mode: 'percentage_24h',
      percentage: '25.5',
      percentage_period: '24h',
      reason: 'incident compensation'
    }))
  })

  it('submits the selected percentage period', async () => {
    const wrapper = mountView()
    await buttonByText(wrapper, 'admin.benefitGrants.modes.percentage_24h').trigger('click')
    await wrapper.get('[data-testid="percentage-period-72h"]').trigger('click')
    await wrapper.findAll('textarea')[0].setValue('72-hour compensation')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(preview).toHaveBeenCalledWith(expect.objectContaining({
      grant_mode: 'percentage_24h',
      percentage_period: '72h'
    }))
  })

  it('submits subscription usage with an independent percentage', async () => {
    const wrapper = mountView()
    await buttonByText(wrapper, 'admin.benefitGrants.modes.percentage_24h').trigger('click')
    await wrapper.get('[data-testid="include-subscription"]').setValue(true)
    await wrapper.get('[data-testid="subscription-percentage"]').setValue('7.5')
    await wrapper.findAll('textarea')[0].setValue('subscription compensation')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(preview).toHaveBeenCalledWith(expect.objectContaining({
      include_subscription: true,
      subscription_percentage: '7.5',
      percentage: '10'
    }))
  })

  it('converts a custom local window to locked ISO timestamps', async () => {
    const wrapper = mountView()
    const start = '2026-07-01T08:30'
    const end = '2026-07-03T18:45'
    await buttonByText(wrapper, 'admin.benefitGrants.modes.percentage_24h').trigger('click')
    await wrapper.get('[data-testid="percentage-period-custom"]').trigger('click')
    await wrapper.get('[data-testid="custom-window-start"]').setValue(start)
    await wrapper.get('[data-testid="custom-window-end"]').setValue(end)
    await wrapper.findAll('textarea')[0].setValue('custom-window compensation')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(preview).toHaveBeenCalledWith(expect.objectContaining({
      percentage_period: 'custom',
      custom_window_start: new Date(start).toISOString(),
      custom_window_end: new Date(end).toISOString()
    }))
  })

  it('accepts deduplicated platform IDs for selected recipients', async () => {
    const wrapper = mountView()
    await wrapper.get('input[type="radio"][value="selected"]').setValue()
    await wrapper.get('[data-testid="platform-id-input"]').setValue('1024, 2048\n1024')
    await wrapper.get('input[placeholder="0.00000000"]').setValue('2.5')
    await wrapper.findAll('textarea')[1].setValue('platform ID grant')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(preview).toHaveBeenCalledWith(expect.objectContaining({
      audience_type: 'selected',
      platform_ids: [1024, 2048],
      reason: 'platform ID grant'
    }))
  })
})
