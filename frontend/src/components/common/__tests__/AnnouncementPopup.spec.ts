import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import AnnouncementPopup from '../AnnouncementPopup.vue'
import { useAnnouncementStore } from '@/stores/announcements'
import type { UserBenefitGrant } from '@/api/benefitGrants'

const announcementMarkdownStyles = readFileSync(
  resolve(process.cwd(), 'src/styles/announcement-markdown.css'),
  'utf8',
)

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const announcement = {
  id: 1,
  title: 'Preview announcement',
  content: '## Preview heading\n\n<div>HTML content</div><script>window.__xss = true</script>',
  status: 'draft' as const,
  notify_mode: 'popup' as const,
  targeting: { any_of: [] },
  created_at: '2026-07-24T07:30:00Z',
  updated_at: '2026-07-24T07:30:00Z',
}

const percentageBenefit = {
  id: 8,
  batch_id: 3,
  grant_type: 'compensation',
  grant_mode: 'percentage_24h',
  base_cost: '30.00000000',
  balance_base_cost: '20.00000000',
  subscription_base_cost: '10.00000000',
  percentage: '10.00000000',
  include_subscription: true,
  subscription_percentage: '5.00000000',
  amount: '2.50000000',
  balance_after: '12.50000000',
  reason: 'service incident',
  title: 'Compensation received',
  content: 'Your compensation has arrived.',
  window_start: '2026-07-21T07:30:00Z',
  window_end: '2026-07-24T07:30:00Z',
  created_at: '2026-07-24T07:30:00Z',
} satisfies UserBenefitGrant

describe('AnnouncementPopup', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    document.body.innerHTML = ''
    document.body.style.overflow = ''
  })

  it('renders mixed Markdown and HTML inside the shared styled container', async () => {
    const store = useAnnouncementStore()
    store.currentPopup = {
      id: 1,
      title: 'Mixed content announcement',
      content: [
        '## Markdown heading',
        '',
        '<div><h3>HTML heading</h3><ul><li>HTML list item</li></ul></div>',
        '',
        '<table><thead><tr><th>Status</th></tr></thead><tbody><tr><td>OK</td></tr></tbody></table>',
        '<script>window.__announcementXss = true</script>',
      ].join('\n'),
      notify_mode: 'popup',
      created_at: '2026-07-24T07:30:00Z',
      updated_at: '2026-07-24T07:30:00Z',
    }

    const wrapper = mount(AnnouncementPopup)
    await wrapper.vm.$nextTick()

    const content = document.body.querySelector('.markdown-body')
    expect(content?.querySelector('h2')?.textContent).toBe('Markdown heading')
    expect(content?.querySelector('h3')?.textContent).toBe('HTML heading')
    expect(content?.querySelector('li')?.textContent).toBe('HTML list item')
    expect(content?.querySelector('table td')?.textContent).toBe('OK')
    expect(content?.querySelector('script')).toBeNull()

    wrapper.unmount()
  })

  it.each(['h2', 'h3', 'ul', 'li', 'blockquote', 'table', 'th', 'td', 'code'])(
    'loads a shared style rule for mixed-content <%s> elements',
    (element) => {
      expect(announcementMarkdownStyles).toContain(`.markdown-body ${element}`)
    },
  )

  it('previews an admin announcement without marking it as read', async () => {
    const store = useAnnouncementStore()
    const dismissPopup = vi.spyOn(store, 'dismissPopup')
    const wrapper = mount(AnnouncementPopup, {
      props: {
        announcement,
        preview: true,
      },
    })

    expect(document.body.textContent).toContain('Preview announcement')
    expect(document.body.querySelector('.markdown-body h2')?.textContent).toBe('Preview heading')
    expect(document.body.querySelector('.markdown-body script')).toBeNull()
    expect(document.body.textContent).toContain('common.close')

    const dismissButton = document.body.querySelector<HTMLButtonElement>(
      '[data-testid="announcement-popup-dismiss"]',
    )
    dismissButton?.click()
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('close')).toHaveLength(1)
    expect(dismissPopup).not.toHaveBeenCalled()

    await wrapper.setProps({ announcement: null })
    expect(document.body.style.overflow).toBe('')
    wrapper.unmount()
  })

  it('renders an external benefit notification without dismissing the announcement store', async () => {
    const store = useAnnouncementStore()
    const dismissPopup = vi.spyOn(store, 'dismissPopup')
    const wrapper = mount(AnnouncementPopup, {
      props: {
        announcement: percentageBenefit,
        benefitDetails: percentageBenefit,
        badgeLabel: 'Balance received',
        acknowledgeLabel: 'Got it',
      },
    })

    expect(document.body.textContent).toContain('Balance received')
    expect(document.body.textContent).toContain('Got it')
    expect(document.body.textContent).toContain('benefits.calculation.balanceSpending')
    expect(document.body.textContent).toContain('benefits.calculation.subscriptionSpending')
    expect(document.body.textContent).toContain('$20')
    expect(document.body.textContent).toContain('$10')
    expect(document.body.textContent).toContain('10%')
    expect(document.body.textContent).toContain('5%')
    expect(document.body.textContent).toContain('+$2.00')
    expect(document.body.textContent).toContain('+$0.50')
    expect(document.body.textContent).toContain('benefits.calculation.actualAmount')
    expect(document.body.textContent).toContain('benefits.calculation.window')
    expect(document.body.textContent).toContain('service incident')
    expect(document.body.querySelector('[data-testid="benefit-grant-reason"]')).not.toBeNull()
    expect(document.body.querySelector('[data-testid="benefit-calculation-details"]')).not.toBeNull()
    expect(document.body.querySelector('[data-testid="announcement-popup-panel"]')).not.toBeNull()
    expect(document.body.querySelector('[data-testid="announcement-popup-body"]')?.classList).toContain(
      'overflow-y-auto',
    )
    expect(document.body.querySelector('.markdown-body script')).toBeNull()

    document.body.querySelector<HTMLButtonElement>('[data-testid="announcement-popup-dismiss"]')?.click()
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('close')).toHaveLength(1)
    expect(dismissPopup).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('does not show calculation details for fixed-amount grants', () => {
    const fixedBenefit: UserBenefitGrant = {
      ...percentageBenefit,
      grant_mode: 'fixed',
      base_cost: '0.00000000',
      balance_base_cost: '0.00000000',
      subscription_base_cost: '0.00000000',
      percentage: undefined,
      include_subscription: false,
      subscription_percentage: undefined,
      window_start: undefined,
      window_end: undefined,
    }
    const wrapper = mount(AnnouncementPopup, {
      props: {
        announcement: fixedBenefit,
        benefitDetails: fixedBenefit,
      },
    })

    expect(document.body.querySelector('[data-testid="benefit-calculation-details"]')).toBeNull()
    wrapper.unmount()
  })

  it('does not duplicate a benefit reason already rendered by the notification template', () => {
    const benefitWithRenderedReason: UserBenefitGrant = {
      ...percentageBenefit,
      content: `Reason: ${percentageBenefit.reason}`,
    }
    const wrapper = mount(AnnouncementPopup, {
      props: {
        announcement: benefitWithRenderedReason,
        benefitDetails: benefitWithRenderedReason,
      },
    })

    expect(document.body.querySelector('[data-testid="benefit-grant-reason"]')).toBeNull()
    expect(document.body.textContent?.match(/service incident/g)).toHaveLength(1)
    wrapper.unmount()
  })

  it('keeps the existing user popup dismissal behavior', async () => {
    const store = useAnnouncementStore()
    store.currentPopup = announcement
    const dismissPopup = vi.spyOn(store, 'dismissPopup').mockResolvedValue()
    const wrapper = mount(AnnouncementPopup)

    const dismissButton = document.body.querySelector<HTMLButtonElement>(
      '[data-testid="announcement-popup-dismiss"]',
    )
    dismissButton?.click()
    await wrapper.vm.$nextTick()

    expect(dismissPopup).toHaveBeenCalledTimes(1)
    expect(wrapper.emitted('close')).toBeUndefined()
    wrapper.unmount()
  })
})
