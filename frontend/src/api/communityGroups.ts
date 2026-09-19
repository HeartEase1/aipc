import { apiClient } from './client'
import type { CommunityGroup } from '@/types'

export async function getCommunityGroups(): Promise<CommunityGroup[]> {
  const { data } = await apiClient.get<CommunityGroup[]>('/community-groups')
  return Array.isArray(data) ? data : []
}

export const communityGroupsAPI = {
  getCommunityGroups
}
