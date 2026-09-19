import { apiClient } from '../client'

export interface MembershipTier {
  id: number
  name: string
  settlement_currency: string
  threshold_amount: string
  discount_percent: string
  sort_order: number
  enabled: boolean
}
export interface RechargePromotion {
  id: number
  kind: 'recharge' | 'first_recharge'
  name: string
  description: string
  settlement_currency: string
  enabled: boolean
  starts_at: string
  ends_at: string
  timezone: string
  min_amount: string | null
  max_amount: string | null
  discount_percent: string
  max_discount_amount: string | null
  budget_amount: string | null
  reserved_amount: string
  redeemed_amount: string
}
export interface MarketingConfig { settlement_currency: string; affiliate_commission_rate: string }
const root = '/admin/payment'
export const balanceMarketingAPI = {
  config: () => apiClient.get<MarketingConfig>(`${root}/balance-marketing`),
  saveConfig: (settlement_currency: string) => apiClient.put(`${root}/balance-marketing`, { settlement_currency }),
  tiers: () => apiClient.get<MembershipTier[]>(`${root}/membership-tiers`),
  promotions: () => apiClient.get<RechargePromotion[]>(`${root}/recharge-promotions`),
  saveTier: (data: Omit<MembershipTier, 'id'>, id?: number) => id
    ? apiClient.put(`${root}/membership-tiers/${id}`, data)
    : apiClient.post(`${root}/membership-tiers`, data),
  savePromotion: (data: Omit<RechargePromotion, 'id' | 'reserved_amount' | 'redeemed_amount'>, id?: number) => id
    ? apiClient.put(`${root}/recharge-promotions/${id}`, data)
    : apiClient.post(`${root}/recharge-promotions`, data),
  deleteTier: (id: number) => apiClient.delete(`${root}/membership-tiers/${id}`),
  deletePromotion: (id: number) => apiClient.delete(`${root}/recharge-promotions/${id}`),
}
