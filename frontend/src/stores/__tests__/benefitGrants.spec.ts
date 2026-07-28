import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useBenefitGrantStore } from '../benefitGrants'

const { list, markRead } = vi.hoisted(() => ({
  list: vi.fn(),
  markRead: vi.fn()
}))

vi.mock('@/api/benefitGrants', () => ({
  default: { list, markRead }
}))

const firstGrant = {
  id: 1,
  batch_id: 10,
  grant_type: 'welfare' as const,
  amount: '2.00000000',
  balance_after: '12.00000000',
  reason: 'welcome',
  title: 'Welcome grant',
  content: 'You received 2',
  created_at: '2026-07-28T00:00:00Z'
}

const secondGrant = { ...firstGrant, id: 2, title: 'Compensation', grant_type: 'compensation' as const }

describe('benefit grant notification store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    list.mockReset()
    markRead.mockReset()
    markRead.mockResolvedValue({ message: 'ok' })
  })

  it('shows unread grants in order and marks only the acknowledged item as read', async () => {
    vi.useFakeTimers()
    list.mockResolvedValueOnce({ items: [firstGrant, secondGrant], total: 2, page: 1, page_size: 20, pages: 1 })

    const store = useBenefitGrantStore()
    await store.fetchUnread(true)
    expect(store.currentPopup?.id).toBe(1)

    const dismiss = store.dismissPopup()
    await vi.runAllTimersAsync()
    await dismiss

    expect(markRead).toHaveBeenCalledWith(1)
    expect(store.currentPopup?.id).toBe(2)
    vi.useRealTimers()
  })

  it('loads the next unread page after the current queue is exhausted', async () => {
    list
      .mockResolvedValueOnce({ items: [firstGrant], total: 1, page: 1, page_size: 20, pages: 1 })
      .mockResolvedValueOnce({ items: [], total: 0, page: 1, page_size: 20, pages: 1 })

    const store = useBenefitGrantStore()
    await store.fetchUnread(true)
    await store.dismissPopup()

    expect(markRead).toHaveBeenCalledWith(1)
    expect(list).toHaveBeenCalledTimes(2)
    expect(store.currentPopup).toBeNull()
  })
})
