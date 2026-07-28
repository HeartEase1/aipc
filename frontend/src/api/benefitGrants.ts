import { apiClient } from './client'
import type { BenefitGrantType } from './admin/benefitGrants'

export interface UserBenefitGrant {
  id: number
  batch_id: number
  grant_type: BenefitGrantType
  amount: string
  balance_after: string
  reason: string
  title: string
  content: string
  read_at?: string
  created_at: string
}

export interface UserBenefitGrantList {
  items: UserBenefitGrant[]
  total: number
  page: number
  page_size: number
  pages: number
}

export async function list(page = 1, pageSize = 20, unreadOnly = false): Promise<UserBenefitGrantList> {
  const { data } = await apiClient.get<UserBenefitGrantList>('/user/benefit-grants', {
    params: { page, page_size: pageSize, unread_only: unreadOnly ? 1 : undefined }
  })
  return data
}

export async function markRead(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.put<{ message: string }>(`/user/benefit-grants/${id}/read`)
  return data
}

export default { list, markRead }
