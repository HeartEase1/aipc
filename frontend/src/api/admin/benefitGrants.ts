import { apiClient } from '../client'

export type BenefitGrantType = 'welfare' | 'compensation'
export type BenefitGrantMode = 'fixed' | 'percentage_24h'
export type BenefitGrantPercentagePeriod = '24h' | '72h' | '30d' | 'custom'
export type BenefitGrantAudience = 'all' | 'selected'
export type BenefitGrantStatus =
  | 'draft'
  | 'pending'
  | 'processing'
  | 'completed'
  | 'partially_failed'
  | 'failed'
  | 'expired'

export interface BenefitGrantBatch {
  id: number
  grant_type: BenefitGrantType
  grant_mode: BenefitGrantMode
  audience_type: BenefitGrantAudience
  fixed_amount?: string
  percentage?: string
  include_subscription: boolean
  subscription_percentage?: string
  min_amount?: string
  per_user_cap?: string
  total_budget_cap?: string
  reason: string
  notification_title: string
  notification_content: string
  window_start?: string
  window_end?: string
  status: BenefitGrantStatus
  eligible_count: number
  skipped_count: number
  success_count: number
  failed_count: number
  total_base_cost: string
  total_balance_base_cost: string
  total_subscription_base_cost: string
  total_amount: string
  total_balance_amount: string
  total_subscription_amount: string
  distributed_amount: string
  average_amount: string
  max_amount: string
  created_by?: number
  executed_by?: number
  expires_at: string
  started_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
  over_budget: boolean
}

export interface BenefitGrantItem {
  id: number
  batch_id: number
  user_id: number
  email: string
  username: string
  base_cost: string
  balance_base_cost: string
  subscription_base_cost: string
  amount: string
  balance_amount: string
  subscription_amount: string
  balance_before?: string
  balance_after?: string
  status: string
  error_message?: string
  processed_at?: string
  read_at?: string
  created_at: string
}

export interface BenefitGrantPreviewRequest {
  grant_type: BenefitGrantType
  grant_mode: BenefitGrantMode
  audience_type: BenefitGrantAudience
  user_ids?: number[]
  platform_ids?: number[]
  fixed_amount?: string
  percentage?: string
  include_subscription?: boolean
  subscription_percentage?: string
  percentage_period?: BenefitGrantPercentagePeriod
  custom_window_start?: string
  custom_window_end?: string
  min_amount?: string
  per_user_cap?: string
  total_budget_cap?: string
  reason: string
  notification_title: string
  notification_content: string
}

export interface BenefitGrantBatchList {
  items: BenefitGrantBatch[]
  total: number
  page: number
  page_size: number
  pages: number
}

export interface BenefitGrantBatchDetail {
  batch: BenefitGrantBatch
  items: BenefitGrantItem[]
  total: number
  page: number
  page_size: number
  pages: number
}

function idempotencyKey(): string {
  return typeof crypto !== 'undefined' && 'randomUUID' in crypto
    ? crypto.randomUUID()
    : `benefit-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export async function preview(payload: BenefitGrantPreviewRequest): Promise<BenefitGrantBatch> {
  const { data } = await apiClient.post<BenefitGrantBatch>('/admin/benefit-grants/preview', payload)
  return data
}

export async function execute(id: number): Promise<BenefitGrantBatch> {
  const { data } = await apiClient.post<BenefitGrantBatch>(
    `/admin/benefit-grants/${id}/execute`,
    {},
    { headers: { 'Idempotency-Key': idempotencyKey() } }
  )
  return data
}

export async function list(page = 1, pageSize = 20, status = ''): Promise<BenefitGrantBatchList> {
  const { data } = await apiClient.get<BenefitGrantBatchList>('/admin/benefit-grants', {
    params: { page, page_size: pageSize, status: status || undefined }
  })
  return data
}

export async function get(id: number, page = 1, pageSize = 50): Promise<BenefitGrantBatchDetail> {
  const { data } = await apiClient.get<BenefitGrantBatchDetail>(`/admin/benefit-grants/${id}`, {
    params: { page, page_size: pageSize }
  })
  return data
}

export async function retryFailed(id: number): Promise<BenefitGrantBatch> {
  const { data } = await apiClient.post<BenefitGrantBatch>(
    `/admin/benefit-grants/${id}/retry-failed`,
    {},
    { headers: { 'Idempotency-Key': idempotencyKey() } }
  )
  return data
}

export async function exportItems(id: number): Promise<Blob> {
  const response = await apiClient.get(`/admin/benefit-grants/${id}/export`, {
    responseType: 'blob'
  })
  return response.data
}

export default { preview, execute, list, get, retryFailed, exportItems }
