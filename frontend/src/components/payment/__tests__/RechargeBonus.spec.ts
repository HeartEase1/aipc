import {afterEach,beforeEach,describe,expect,it,vi} from 'vitest'
import {enableAutoUnmount,flushPromises,mount} from '@vue/test-utils'
import RechargeBonusCard from '../RechargeBonusCard.vue'
import RechargeBonusOrder from '../RechargeBonusOrder.vue'
import {bonusOrderCSV} from '../bonusExport'
import type {PaymentOrder} from '@/types/payment'
const api=vi.hoisted(()=>({post:vi.fn()}))
vi.mock('@/api/client',()=>({apiClient:api}))
enableAutoUnmount(afterEach)
const campaign={id:1,title:'充值赠送',copy:'<b>纯文本活动</b>',settlement_currency:'CNY',starts_at:'2026-09-01T00:00:00Z',ends_at:'2026-10-01T00:00:00Z',timezone:'Asia/Shanghai',frequency:'daily_first',eligible:true,poster_url:'https://example.com/poster.png',tiers:[{min_amount:'0',max_amount:'100',bonus_percent:'10'}]}
const dialog={props:['show','title'],template:'<section v-if="show" role="dialog"><h2>{{ title }}</h2><slot /><slot name="footer" /></section>'}
beforeEach(()=>{vi.useFakeTimers();vi.setSystemTime(new Date('2026-09-22T00:00:00Z'));api.post.mockReset()})
afterEach(()=>vi.useRealTimers())
describe('recharge bonus presentation',()=>{
 it('opens on server instruction and refreshes poster links and eligibility when reopened',async()=>{
  api.post.mockResolvedValue({data:{campaign,show_popup:true}})
  const w=mount(RechargeBonusCard,{props:{currency:'CNY'},global:{stubs:{BaseDialog:dialog}}});await flushPromises()
  expect(w.find('[role="dialog"]').exists()).toBe(true);expect(w.find('b').exists()).toBe(false)
  await w.get('img').trigger('error');expect(w.find('img').exists()).toBe(false);expect(w.text()).toContain('纯文本活动')
  await w.findAll('button').at(-1)!.trigger('click');expect(w.find('[role="dialog"]').exists()).toBe(false)
  api.post.mockResolvedValue({data:{campaign:{...campaign,eligible:false,poster_url:'https://example.com/renewed.png'},show_popup:false}})
  await w.get('button').trigger('click');await flushPromises();expect(w.find('[role="dialog"]').exists()).toBe(true);expect(api.post).toHaveBeenCalledTimes(2)
  expect(w.get('img').attributes('src')).toBe('https://example.com/renewed.png');expect(w.text()).toContain('资格已使用')
  vi.setSystemTime(new Date('2026-10-01T00:00:01Z'));await vi.advanceTimersByTimeAsync(1000);expect(w.find('button').exists()).toBe(false)
 })
 it('respects cross-device daily suppression and lets an ineligible user inspect rules',async()=>{
  api.post.mockResolvedValue({data:{campaign:{...campaign,eligible:false},show_popup:false}})
  const w=mount(RechargeBonusCard,{props:{currency:'CNY'},global:{stubs:{BaseDialog:dialog}}});await flushPromises();expect(w.find('[role="dialog"]').exists()).toBe(false);expect(w.text()).toContain('资格已使用');await w.get('button').trigger('click');expect(w.find('[role="dialog"]').exists()).toBe(true)
 })
 it('shows 1000 principal plus 100 bonus and exports actual values safely',()=>{
  const order={id:1,out_trade_no:'order1',amount:1000,pay_amount:90,original_amount:100,discount_amount:10,currency:'CNY',status:'COMPLETED',pricing:{bonus:{campaign_id:1,title:'=danger',percent:'10',multiplier:'10',expected:'100',confirmed:true,awarded:true,credited:true,refunded:'25',reason:'获赠整笔不返利'}}} as PaymentOrder
  const w=mount(RechargeBonusOrder,{props:{order}});expect(w.text()).toContain('$1100.00');expect(w.text()).toContain('$25.00');const csv=bonusOrderCSV([order]);expect(csv).toContain('"1000","100","100","1100","25"');expect(csv).toContain('"\'=danger"')
 })
})
