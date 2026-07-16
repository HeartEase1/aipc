import { apiClient } from './client'

export type LeaderboardPeriod = '24h' | '72h' | '7d' | '30d'

export interface LeaderboardUsageEntry {
  rank: number
  display_name: string
  request_count: number
  total_tokens: number
  actual_cost: number
}

export interface LeaderboardRebateEntry {
  rank: number
  display_name: string
  invited_users: number
  rebate_count: number
  rebate_amount: number
}

export interface LeaderboardUsageBoard {
  summary: {
    request_count: number
    total_tokens: number
    actual_cost: number
  }
  entries: LeaderboardUsageEntry[]
  current?: LeaderboardUsageEntry
}

export interface LeaderboardRebateBoard {
  summary: {
    invited_users: number
    rebate_count: number
    rebate_amount: number
  }
  entries: LeaderboardRebateEntry[]
  current?: LeaderboardRebateEntry
}

export interface LeaderboardResponse {
  period: LeaderboardPeriod
  start_at: string
  end_at: string
  participating: boolean
  usage: LeaderboardUsageBoard
  consumption: LeaderboardUsageBoard
  rebate: LeaderboardRebateBoard
}

export async function getLeaderboard(period: LeaderboardPeriod): Promise<LeaderboardResponse> {
  const { data } = await apiClient.get<LeaderboardResponse>('/leaderboard', { params: { period } })
  return data
}

export async function updateLeaderboardParticipation(enabled: boolean): Promise<{ enabled: boolean }> {
  const { data } = await apiClient.put<{ enabled: boolean }>('/user/leaderboard-participation', {
    enabled
  })
  return data
}

export const leaderboardAPI = {
  getLeaderboard,
  updateLeaderboardParticipation
}
