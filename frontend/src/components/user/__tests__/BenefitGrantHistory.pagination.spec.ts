import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import BenefitGrantHistory from '../BenefitGrantHistory.vue'
import Pagination from '@/components/common/Pagination.vue'

const { list } = vi.hoisted(() => ({ list: vi.fn() }))

vi.mock('@/api/benefitGrants', () => ({
  default: { list },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

describe('BenefitGrantHistory redemption pagination', () => {
  beforeEach(() => {
    list.mockReset().mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
  })

  it('forwards server page and page-size changes to the parent view', async () => {
    const wrapper = mount(BenefitGrantHistory, {
      props: {
        redeemHistory: [],
        redeemPage: 1,
        redeemPageSize: 20,
        redeemTotal: 101,
      },
      global: {
        stubs: {
          Icon: true,
          LoadingSpinner: true,
          BenefitGrantCalculationDetails: true,
          Select: true,
        },
      },
    })
    await flushPromises()

    const section = wrapper.get('[data-testid="redeem-history-pagination"]')
    const pagination = section.getComponent(Pagination)
    expect(pagination.props()).toMatchObject({
      total: 101,
      page: 1,
      pageSize: 20,
      pageSizeOptions: [20, 50, 100],
    })

    pagination.vm.$emit('update:page', 2)
    pagination.vm.$emit('update:pageSize', 50)

    expect(wrapper.emitted('update:redeemPage')).toEqual([[2]])
    expect(wrapper.emitted('update:redeemPageSize')).toEqual([[50]])
  })

  it('blocks pagination events while redemption controls are disabled', async () => {
    const wrapper = mount(BenefitGrantHistory, {
      props: {
        redeemHistory: [],
        redeemPage: 1,
        redeemPageSize: 20,
        redeemTotal: 101,
        redeemPaginationDisabled: true,
      },
      global: {
        stubs: {
          Icon: true,
          LoadingSpinner: true,
          BenefitGrantCalculationDetails: true,
          Select: true,
        },
      },
    })
    await flushPromises()

    const section = wrapper.get('[data-testid="redeem-history-pagination"]')
    expect(section.attributes('aria-disabled')).toBe('true')
    const pagination = section.getComponent(Pagination)
    pagination.vm.$emit('update:page', 2)
    pagination.vm.$emit('update:pageSize', 50)

    expect(wrapper.emitted('update:redeemPage')).toBeUndefined()
    expect(wrapper.emitted('update:redeemPageSize')).toBeUndefined()
  })
})
