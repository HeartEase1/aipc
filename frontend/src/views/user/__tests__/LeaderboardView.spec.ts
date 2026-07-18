import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import LeaderboardView from '../LeaderboardView.vue'

const { getLeaderboard, updateLeaderboardParticipation, showError } = vi.hoisted(() => ({
  getLeaderboard: vi.fn(),
  updateLeaderboardParticipation: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/leaderboard', () => ({
  leaderboardAPI: { getLeaderboard, updateLeaderboardParticipation }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const leaderboard = {
  period: '24h' as const,
  start_at: '2026-07-16T00:00:00Z',
  end_at: '2026-07-17T00:00:00Z',
  participating: true,
  usage: {
    summary: { request_count: 12, total_tokens: 3456, actual_cost: 1.2 },
    current: { rank: 2, display_name: 'a***z@example.com', request_count: 5, total_tokens: 2000, actual_cost: 0.8 },
    entries: [{ rank: 1, display_name: 'Ada', request_count: 7, total_tokens: 3000, actual_cost: 1.1 }]
  },
  consumption: {
    summary: { request_count: 12, total_tokens: 3456, actual_cost: 1.2 },
    current: { rank: 3, display_name: 'Consumer', request_count: 5, total_tokens: 2000, actual_cost: 0.8 },
    entries: [{ rank: 1, display_name: 'Cost leader', request_count: 7, total_tokens: 3000, actual_cost: 1.1 }]
  },
  rebate: {
    summary: { invited_users: 2, rebate_count: 4, rebate_amount: 0.5 },
    current: { rank: 2, display_name: 'a***z@example.com', invited_users: 1, rebate_count: 2, rebate_amount: 0.2 },
    entries: [{ rank: 1, display_name: 'Ada', invited_users: 2, rebate_count: 2, rebate_amount: 0.3 }]
  }
}

function mountLeaderboardView() {
  return mount(LeaderboardView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' }
      }
    }
  })
}

function buttonByText(wrapper: ReturnType<typeof mount>, text: string) {
  const button = wrapper.findAll('button').find((candidate) => candidate.text() === text)
  if (!button) throw new Error(`button not found: ${text}`)
  return button
}

describe('LeaderboardView', () => {
  beforeEach(() => {
    getLeaderboard.mockReset()
    updateLeaderboardParticipation.mockReset()
    showError.mockReset()
    getLeaderboard.mockResolvedValue(leaderboard)
    updateLeaderboardParticipation.mockResolvedValue({ enabled: false })
  })

  it('loads the 24-hour leaderboard by default and reloads when the period changes', async () => {
    const wrapper = mountLeaderboardView()
    await flushPromises()

    expect(getLeaderboard).toHaveBeenCalledWith('24h')
    await buttonByText(wrapper, 'leaderboard.period72h').trigger('click')
    await flushPromises()
    expect(getLeaderboard).toHaveBeenLastCalledWith('72h')
  })

  it('renders the rank-zero current row and switches to the consumption tab', async () => {
    const wrapper = mountLeaderboardView()
    await flushPromises()

    expect(wrapper.text()).toContain('a***z@example.com')
    expect(wrapper.findAll('td').some((cell) => cell.text().startsWith('0'))).toBe(true)
    await buttonByText(wrapper, 'leaderboard.consumptionTab').trigger('click')
    expect(wrapper.text()).toContain('Consumer')
  })

  it('labels summary totals and explains the active ranking scope', async () => {
    const wrapper = mountLeaderboardView()
    await flushPromises()

    expect(wrapper.text()).toContain('leaderboard.totalRequests')
    expect(wrapper.text()).toContain('leaderboard.totalTokens')
    expect(wrapper.text()).toContain('leaderboard.totalCost')
    expect(wrapper.text()).toContain('leaderboard.summaryScope')
    expect(wrapper.text()).toContain('leaderboard.usageRule')
    expect(wrapper.text()).toContain('leaderboard.top20')
    expect(wrapper.text()).toContain('$1.20')
    expect(wrapper.text()).not.toContain('US$')

    await buttonByText(wrapper, 'leaderboard.rebateTab').trigger('click')
    expect(wrapper.text()).toContain('leaderboard.totalInvitedUsers')
    expect(wrapper.text()).toContain('leaderboard.totalRebateCount')
    expect(wrapper.text()).toContain('leaderboard.totalRebateAmount')
    expect(wrapper.text()).toContain('leaderboard.rebateRule')
  })

  it('updates participation and refreshes the current period', async () => {
    const wrapper = mountLeaderboardView()
    await flushPromises()

    await wrapper.get('input[type="checkbox"]').setValue(false)
    await flushPromises()

    expect(updateLeaderboardParticipation).toHaveBeenCalledWith(false)
    expect(getLeaderboard).toHaveBeenLastCalledWith('24h')
  })
})
