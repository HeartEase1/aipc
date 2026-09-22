<template>
 <div v-if="active" class="mt-4">
  <button type="button" class="w-full rounded-xl border border-amber-200 bg-amber-50 p-4 text-left text-amber-900 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-200" @click="reopen"><span class="block font-semibold">{{ campaign?.title }}</span><span class="mt-1 block text-xs">{{ campaign?.eligible ? '充值赠送 · 点击查看活动规则' : '本期资格已使用 · 点击查看活动规则' }}</span></button>
  <BaseDialog :show="open && active" :title="campaign?.title || '充值赠送'" width="wide" @close="open=false">
   <template v-if="campaign">
    <img v-if="campaign.poster_url && !imageFailed" :src="campaign.poster_url" :alt="campaign.title" class="mb-4 max-h-[45vh] w-full rounded-xl object-contain" @error="imageFailed=true" />
    <p class="whitespace-pre-wrap text-sm text-gray-700 dark:text-gray-200">{{ campaign.copy }}</p>
    <p class="mt-4 text-sm">{{ new Date(campaign.starts_at).toLocaleString() }} — {{ new Date(campaign.ends_at).toLocaleString() }}</p>
    <p class="mt-1 text-xs text-gray-500">每日规则按 {{ campaign.timezone }} 计算 · {{ frequency }}</p>
    <table class="my-4 w-full text-left text-sm"><thead><tr><th class="p-2">单笔充值面额（{{ campaign.settlement_currency }}）</th><th>赠送比例</th></tr></thead><tbody><tr v-for="(tier,i) in campaign.tiers" :key="i" class="border-t dark:border-dark-700"><td class="p-2">超过 {{ Number(tier.min_amount) }}{{ tier.max_amount ? `，至 ${Number(tier.max_amount)}（含）` : '，上限不限' }}</td><td>{{ Number(tier.bonus_percent) }}%</td></tr></tbody></table>
    <p v-if="!campaign.eligible" class="mb-3 text-amber-700 dark:text-amber-300">本期领取资格已使用。</p>
    <p class="text-xs leading-relaxed text-gray-500">赠送按充值面额及订单换算倍率计算，可与充值折扣叠加，不含手续费。首笔资格以符合档位的支付成功确认为准，未付款不占次数。有效订单锁定活动规则。获赠订单整笔不参与邀请返利；退款按比例收回赠送，退款不恢复次数。</p>
   </template>
   <template #footer><button class="btn btn-primary" @click="open=false">知道了</button></template>
  </BaseDialog>
 </div>
</template>
<script setup lang="ts">
import {computed,onUnmounted,ref,watch} from 'vue'
import {apiClient} from '@/api/client'
import type {RechargeBonusCampaign} from '@/api/admin/balanceMarketing'
import BaseDialog from '@/components/common/BaseDialog.vue'
const props=defineProps<{currency:string}>()
const campaign=ref<RechargeBonusCampaign|null>(null),open=ref(false),imageFailed=ref(false),now=ref(Date.now())
const active=computed(()=>!!campaign.value&&new Date(campaign.value.ends_at).getTime()>now.value)
const frequency=computed(()=>({every_payment:'每笔赠送',daily_first:'每日首笔符合条件的成功充值',campaign_first:'活动期间首笔符合条件的成功充值'}[campaign.value?.frequency||'every_payment']))
let version=0
async function reopen(){
 open.value=true
 const current=++version
 try{
  const {data}=await apiClient.post<{campaign:RechargeBonusCampaign|null;show_popup:boolean}>('/payment/recharge-bonus',null,{params:{currency:props.currency}})
  if(current!==version)return
  campaign.value=data.campaign;now.value=Date.now();imageFailed.value=false
 }catch{ /* Keep the available text if refreshing the poster or eligibility fails. */ }
}
const timer=setInterval(()=>{now.value=Date.now();if(!active.value)open.value=false},1000)
watch(()=>props.currency,async currency=>{
 const current=++version;campaign.value=null;open.value=false;imageFailed.value=false;if(!currency)return
 try{const {data}=await apiClient.post<{campaign:RechargeBonusCampaign|null;show_popup:boolean}>('/payment/recharge-bonus',null,{params:{currency}});if(current!==version)return;campaign.value=data.campaign;now.value=Date.now();open.value=data.show_popup}catch{ /* Activity display must not prevent checkout. */ }
},{immediate:true})
onUnmounted(()=>{version++;clearInterval(timer)})
</script>
