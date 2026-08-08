import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import DiscountCampaignsView from '../DiscountCampaignsView.vue'

const { list, create, update, remove, stepUpRun, showError, showSuccess } = vi.hoisted(() => ({
  list: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  remove: vi.fn(),
  stepUpRun: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    discountCampaigns: { list, create, update, remove }
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ showError, showSuccess })
}))

vi.mock('@/composables/useStepUp', () => ({
  isStepUpCancelled: () => false,
  useStepUp: () => ({ run: stepUpRun })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const existingCampaign = {
  id: 42,
  name: 'Sunday discount',
  description: 'Weekend balance discount',
  enabled: true,
  schedule_type: 'weekly',
  timezone: 'Asia/Shanghai',
  weekdays: [0],
  start_time: '22:00',
  end_time: '02:00',
  all_day: false,
  discount_factor: '0.900000',
  min_effective_multiplier: '0.5',
  budget_cap: '100',
  discount_spent: '12.5',
  created_at: '2026-08-01T00:00:00Z',
  updated_at: '2026-08-01T00:00:00Z',
  status: 'active'
} as const

function mountView() {
  return mount(DiscountCampaignsView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        Icon: true,
        BaseDialog: {
          props: ['show'],
          template: '<section v-if="show"><slot /><slot name="footer" /></section>'
        },
        ConfirmDialog: {
          props: ['show'],
          emits: ['confirm', 'cancel'],
          template: '<div v-if="show"><button data-testid="confirm-delete" @click="$emit(\'confirm\')">confirm</button></div>'
        },
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

async function openCreateForm(wrapper: ReturnType<typeof mount>) {
  await flushPromises()
  await buttonByText(wrapper, 'admin.discountCampaigns.create').trigger('click')
}

describe('DiscountCampaignsView', () => {
  beforeEach(() => {
    list.mockReset().mockResolvedValue([])
    create.mockReset().mockResolvedValue(existingCampaign)
    update.mockReset().mockResolvedValue(existingCampaign)
    remove.mockReset().mockResolvedValue(undefined)
    stepUpRun.mockReset().mockImplementation((callback: () => unknown) => callback())
    showError.mockReset()
    showSuccess.mockReset()
  })

  it('opens with a valid one-time schedule by default', async () => {
    const wrapper = mountView()
    await openCreateForm(wrapper)

    expect(buttonByText(wrapper, 'admin.discountCampaigns.scheduleTypes.one_time').classes()).toContain('bg-white')
    expect(wrapper.findAll('input[type="datetime-local"]')).toHaveLength(2)
    expect(wrapper.findAll('input[type="time"]')).toHaveLength(0)
  })

  it('converts the paid percentage and submits create through step-up verification', async () => {
    const wrapper = mountView()
    await openCreateForm(wrapper)
    await wrapper.get('input[maxlength="120"]').setValue('Launch discount')
    await wrapper.get('textarea[maxlength="500"]').setValue('Launch week offer')
    await wrapper.get('input[type="number"]').setValue('85')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(stepUpRun).toHaveBeenCalledTimes(1)
    expect(create).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Launch discount',
      description: 'Launch week offer',
      schedule_type: 'one_time',
      timezone: 'Asia/Shanghai',
      discount_factor: '0.850000',
      weekdays: [],
      all_day: false
    }))
  })

  it('keeps form switches aligned within their rows', async () => {
    const wrapper = mountView()
    await openCreateForm(wrapper)

    const enabledSwitch = wrapper.findAll('button[role="switch"]').at(-1)
    expect(enabledSwitch?.classes()).toContain('shrink-0')
    expect(enabledSwitch?.classes()).toContain('block')
  })

  it('submits selected weekdays and a cross-midnight weekly window', async () => {
    const wrapper = mountView()
    await openCreateForm(wrapper)
    await wrapper.get('input[maxlength="120"]').setValue('Night discount')
    await buttonByText(wrapper, 'admin.discountCampaigns.scheduleTypes.weekly').trigger('click')

    const weekdayInputs = wrapper.findAll('input[type="checkbox"]')
    await weekdayInputs[6].setValue(false)
    await weekdayInputs[0].setValue(true)
    await weekdayInputs[4].setValue(true)
    await wrapper.findAll('button[role="switch"]')[0].trigger('click')

    const timeInputs = wrapper.findAll('input[type="time"]')
    await timeInputs[0].setValue('22:00')
    await timeInputs[1].setValue('02:00')
    await wrapper.get('input[type="number"]').setValue('90')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(create).toHaveBeenCalledWith(expect.objectContaining({
      schedule_type: 'weekly',
      weekdays: [1, 5],
      start_time: '22:00',
      end_time: '02:00',
      all_day: false,
      discount_factor: '0.900000'
    }))
  })

  it('updates an existing campaign through step-up verification', async () => {
    list.mockResolvedValue([existingCampaign])
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('button[title="common.edit"]').trigger('click')
    await wrapper.get('input[maxlength="120"]').setValue('Updated discount')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(stepUpRun).toHaveBeenCalledTimes(1)
    expect(update).toHaveBeenCalledWith(42, expect.objectContaining({
      name: 'Updated discount',
      schedule_type: 'weekly',
      start_time: '22:00',
      end_time: '02:00'
    }))
  })

  it('deletes an existing campaign through step-up verification', async () => {
    list.mockResolvedValue([existingCampaign])
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('button[title="admin.discountCampaigns.delete"]').trigger('click')
    await wrapper.get('[data-testid="confirm-delete"]').trigger('click')
    await flushPromises()

    expect(stepUpRun).toHaveBeenCalledTimes(1)
    expect(remove).toHaveBeenCalledWith(42)
  })
})
