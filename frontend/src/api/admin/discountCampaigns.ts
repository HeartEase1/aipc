import { apiClient } from '../client'

export type DiscountScheduleType = 'one_time' | 'weekly'
export type DiscountCampaignStatus = 'active' | 'upcoming' | 'ended' | 'disabled' | 'budget_exhausted'

export interface DiscountCampaign {
  id: number
  name: string
  enabled: boolean
  schedule_type: DiscountScheduleType
  timezone: string
  starts_at?: string
  ends_at?: string
  weekdays: number[]
  start_time?: string
  end_time?: string
  all_day: boolean
  discount_factor: string
  min_effective_multiplier?: string
  budget_cap?: string
  discount_spent: string
  created_by?: number
  updated_by?: number
  created_at: string
  updated_at: string
  status: DiscountCampaignStatus
}

export interface DiscountCampaignRequest {
  name: string
  enabled: boolean
  schedule_type: DiscountScheduleType
  timezone: string
  starts_at?: string
  ends_at?: string
  weekdays?: number[]
  start_time?: string
  end_time?: string
  all_day?: boolean
  discount_factor: string
  min_effective_multiplier?: string
  budget_cap?: string
}

export async function list(): Promise<DiscountCampaign[]> {
  const { data } = await apiClient.get<DiscountCampaign[]>('/admin/discount-campaigns')
  return data
}

export async function create(payload: DiscountCampaignRequest): Promise<DiscountCampaign> {
  const { data } = await apiClient.post<DiscountCampaign>('/admin/discount-campaigns', payload)
  return data
}

export async function update(id: number, payload: DiscountCampaignRequest): Promise<DiscountCampaign> {
  const { data } = await apiClient.put<DiscountCampaign>(`/admin/discount-campaigns/${id}`, payload)
  return data
}

export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/discount-campaigns/${id}`)
}

export default { list, create, update, remove }
