import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import BenefitGrantHistory from '../BenefitGrantHistory.vue'

const { list } = vi.hoisted(() => ({ list: vi.fn() }))

vi.mock('@/api/benefitGrants', () => ({
  default: { list }
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, values?: Record<string, string>) =>
        values?.amount ? `${key}:${values.amount}` : key
    })
  }
})

describe('BenefitGrantHistory', () => {
  beforeEach(() => {
    list.mockReset()
    list.mockResolvedValue({
      items: [
        {
          id: 7,
          batch_id: 2,
          grant_type: 'compensation',
          grant_mode: 'percentage_24h',
          base_cost: '20.00000000',
          balance_base_cost: '20.00000000',
          subscription_base_cost: '0.00000000',
          percentage: '6.25000000',
          include_subscription: false,
          amount: '1.25000000',
          balance_after: '9.50000000',
          reason: 'service incident',
          title: 'Compensation received',
          content: 'content',
          window_start: '2026-07-27T00:00:00Z',
          window_end: '2026-07-28T00:00:00Z',
          created_at: '2026-07-28T00:00:00Z'
        }
      ],
      total: 21,
      page: 1,
      page_size: 20,
      pages: 2
    })
  })

  it('loads grant records inside the redeem page section with calculation details', async () => {
    const wrapper = mount(BenefitGrantHistory, {
      props: {
        redeemHistory: [
          {
            id: 9,
            code: 'WELCOME-2026',
            type: 'balance',
            value: 10,
            status: 'used',
            used_at: '2026-07-29T00:00:00Z',
            created_at: '2026-07-29T00:00:00Z'
          }
        ]
      },
      global: {
        stubs: {
          Icon: true,
          LoadingSpinner: true,
          Pagination: {
            props: ['page', 'total', 'pageSize'],
            template:
              '<button data-test="next" @click="$emit(\'update:page\', 2)">next</button>'
          }
        }
      }
    })
    await flushPromises()

    expect(list).toHaveBeenCalledWith(1, 20)
    expect(wrapper.text()).toContain('Compensation received')
    expect(wrapper.text()).toContain('service incident')
    expect(wrapper.text()).toContain('+$1.25')
    expect(wrapper.text()).toContain('benefits.balanceAfter:9.5')
    expect(wrapper.text()).toContain('benefits.calculation.balanceSpending')
    expect(wrapper.text()).toContain('$20')
    expect(wrapper.text()).toContain('6.25%')
    expect(wrapper.text()).toContain('+$1.25')
    expect(wrapper.text()).toContain('benefits.calculation.actualAmount')
    expect(wrapper.text()).toContain('benefits.calculation.windowStart')
    expect(wrapper.text()).toContain('benefits.calculation.windowEnd')
    expect(wrapper.find('[data-testid="benefit-calculation-details"]').exists()).toBe(true)
    const activities = wrapper.findAll('article')
    expect(activities).toHaveLength(2)
    expect(activities[0].text()).toContain('redeem.balanceAddedRedeem')
    expect(activities[1].text()).toContain('Compensation received')
    expect(wrapper.get('[data-activity-kind="compensation"]').classes()).toContain('border-l-4')
    expect(wrapper.get('[data-activity-kind="compensation"]').classes()).toContain('bg-amber-50/45')
    expect(wrapper.get('[data-activity-kind="standard"]').classes()).not.toContain('border-l-4')

    await wrapper.get('[data-test="next"]').trigger('click')
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith(2, 20)
  })
})
