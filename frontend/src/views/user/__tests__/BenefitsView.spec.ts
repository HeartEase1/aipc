import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import BenefitsView from '../BenefitsView.vue'

const { list } = vi.hoisted(() => ({ list: vi.fn() }))

vi.mock('@/api/benefitGrants', () => ({
  default: { list }
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string, values?: Record<string, string>) => values?.amount ? `${key}:${values.amount}` : key }) }
})

describe('BenefitsView', () => {
  beforeEach(() => {
    list.mockReset()
    list.mockResolvedValue({
      items: [{
        id: 7,
        batch_id: 2,
        grant_type: 'compensation',
        amount: '1.25000000',
        balance_after: '9.50000000',
        reason: 'service incident',
        title: 'Compensation received',
        content: 'content',
        created_at: '2026-07-28T00:00:00Z'
      }],
      total: 21,
      page: 1,
      page_size: 20,
      pages: 2
    })
  })

  it('loads the first history page and renders exact grant amounts', async () => {
    const wrapper = mount(BenefitsView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true,
          LoadingSpinner: true,
          Pagination: { props: ['page', 'total', 'pageSize'], template: '<button data-test="next" @click="$emit(\'update:page\', 2)">next</button>' }
        }
      }
    })
    await flushPromises()

    expect(list).toHaveBeenCalledWith(1, 20)
    expect(wrapper.text()).toContain('Compensation received')
    expect(wrapper.text()).toContain('service incident')
    expect(wrapper.text()).toContain('+$1.25')
    expect(wrapper.text()).toContain('benefits.balanceAfter:9.5')

    await wrapper.get('[data-test="next"]').trigger('click')
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith(2, 20)
  })
})
