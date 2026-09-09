import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import MembershipBenefits from '../MembershipBenefits.vue'
import PaymentOfferBadge from '../PaymentOfferBadge.vue'
import type { MembershipSummary } from '@/api/payment'
import type { PaymentOrder } from '@/types/payment'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('membership and recharge offer display', () => {
  it.each(['eligible', 'reserved', 'used', 'ineligible'] as const)('renders a distinct first recharge state: %s', (status) => {
    const summary: MembershipSummary = {
      enabled: false, settlement_currency: 'CNY', current_amount: '0',
      current_discount_percent: '0', progress_percent: '0',
      first_recharge_eligible: status === 'eligible', first_recharge_status: status
    }
    const wrapper = mount(MembershipBenefits, { props: { summary }, global: { stubs: { BaseDialog: true, Icon: true } } })
    expect(wrapper.text()).toContain(`balanceMarketing.firstStates.${status}`)
    if (status === 'reserved') expect(wrapper.text()).not.toContain('balanceMarketing.firstStates.used')
  })

  it.each(['first_recharge', 'campaign', 'membership'])('uses the order snapshot offer: %s', (source) => {
    const order = { order_type: 'balance', status: 'CANCELLED', discount_source: source, discount_amount: 10, currency: 'CNY' } as PaymentOrder
    const wrapper = mount(PaymentOfferBadge, { props: { order } })
    expect(wrapper.text()).toContain(`balanceMarketing.sources.${source}`)
    expect(wrapper.text()).toContain('10.00')
  })

  it('does not label ordinary or subscription orders as discounted', () => {
    for (const order of [
      { order_type: 'balance', discount_source: '' },
      { order_type: 'subscription', discount_source: 'membership' }
    ]) {
      const wrapper = mount(PaymentOfferBadge, { props: { order: order as PaymentOrder } })
      expect(wrapper.text()).toBe('')
    }
  })
})
