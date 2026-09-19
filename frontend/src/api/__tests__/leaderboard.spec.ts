import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, put } = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn()
}))

vi.mock('../client', () => ({
  apiClient: { get, put }
}))

import { getLeaderboard, updateLeaderboardParticipation } from '@/api/leaderboard'

describe('leaderboard API', () => {
  beforeEach(() => {
    get.mockReset()
    put.mockReset()
  })

  it('requests the selected ranking period', async () => {
    get.mockResolvedValue({ data: { period: '72h' } })

    const result = await getLeaderboard('72h')

    expect(get).toHaveBeenCalledWith('/leaderboard', { params: { period: '72h' } })
    expect(result.period).toBe('72h')
  })

  it('updates the participation switch with the requested state', async () => {
    put.mockResolvedValue({ data: { enabled: false } })

    const result = await updateLeaderboardParticipation(false)

    expect(put).toHaveBeenCalledWith('/user/leaderboard-participation', { enabled: false })
    expect(result).toEqual({ enabled: false })
  })
})
