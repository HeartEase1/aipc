<template>
  <div class="space-y-5">
    <p class="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-200">{{ t('balanceMarketing.risk', { percent: config.affiliate_commission_rate }) }}</p>
    <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('balanceMarketing.scope') }}</p>
    <div v-if="mode === 'membership'" class="flex flex-wrap items-end gap-3 border-b border-gray-200 pb-5 dark:border-dark-600">
      <label class="block max-w-xs flex-1"><span class="input-label">{{ t('balanceMarketing.currency') }}</span><input v-model.trim="currency" class="input mt-1" maxlength="3" /></label>
      <button class="btn btn-primary" :disabled="busy" @click="saveCurrency">{{ t('common.save') }}</button>
      <p class="w-full text-xs text-gray-500">{{ t('balanceMarketing.currencyHint') }}</p>
    </div>
    <div class="flex flex-wrap gap-2">
      <button v-if="mode === 'membership'" class="btn btn-primary" :disabled="loading" @click="openTier()"><Icon name="plus" size="sm" />{{ t('balanceMarketing.createTier') }}</button>
      <button class="btn btn-secondary" :disabled="loading" @click="openPromotion()"><Icon name="plus" size="sm" />{{ t(mode === 'membership' ? 'balanceMarketing.createFirst' : 'balanceMarketing.createPromotion') }}</button>
      <button class="btn btn-secondary !px-3" :disabled="loading" :title="t('common.refresh')" @click="load"><Icon name="refresh" size="sm" /></button>
    </div>
    <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
    <p v-if="loading" class="py-8 text-sm text-gray-500">{{ t('common.loading') }}</p>
    <template v-else>
      <div v-if="mode === 'membership'" class="overflow-x-auto rounded-lg border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800">
        <table class="w-full min-w-[560px] text-left text-sm">
          <thead class="bg-gray-50 dark:bg-dark-900"><tr><th class="p-3">{{ t('balanceMarketing.tier') }}</th><th class="p-3">{{ t('balanceMarketing.threshold') }}</th><th class="p-3">{{ t('balanceMarketing.reduction') }}</th><th class="p-3">{{ t('common.status') }}</th><th class="p-3 text-right">{{ t('common.actions') }}</th></tr></thead>
          <tbody><tr v-for="item in tiers" :key="item.id" class="border-t dark:border-dark-600"><td class="p-3 font-medium">{{ item.name }}</td><td class="p-3">{{ item.settlement_currency }} {{ Number(item.threshold_amount) }}</td><td class="p-3">{{ Number(item.discount_percent) }}%</td><td class="p-3">{{ t(item.enabled ? 'balanceMarketing.enabled' : 'balanceMarketing.disabled') }}</td><td class="p-3 text-right"><button class="btn btn-secondary !p-2" :title="t('common.edit')" @click="openTier(item)"><Icon name="edit" size="sm" /></button><button class="btn btn-secondary ml-2 !p-2" :title="t('common.delete')" @click="removeTarget = { id: item.id, tier: true }"><Icon name="trash" size="sm" /></button></td></tr>
            <tr v-if="!tiers.length"><td colspan="5" class="p-8 text-center text-gray-500">{{ t('balanceMarketing.empty') }}</td></tr></tbody>
        </table>
      </div>
      <div class="overflow-x-auto rounded-lg border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800">
        <table class="w-full min-w-[720px] text-left text-sm">
          <thead class="bg-gray-50 dark:bg-dark-900"><tr><th class="p-3">{{ t('balanceMarketing.name') }}</th><th class="p-3">{{ t('balanceMarketing.reduction') }}</th><th class="p-3">{{ t('balanceMarketing.budget') }}</th><th class="p-3">{{ t('common.status') }}</th><th class="p-3 text-right">{{ t('common.actions') }}</th></tr></thead>
          <tbody><tr v-for="item in visiblePromotions" :key="item.id" class="border-t dark:border-dark-600"><td class="max-w-xs p-3"><p class="font-medium">{{ item.name }}</p><p v-if="item.description" class="mt-0.5 line-clamp-2 text-xs text-gray-500">{{ item.description }}</p><p class="text-xs text-gray-500">{{ date(item.starts_at) }} - {{ date(item.ends_at) }}</p><p class="text-xs text-gray-500">{{ item.timezone }}</p></td><td class="p-3">{{ Number(item.discount_percent) }}%</td><td class="p-3"><p>{{ item.settlement_currency }} {{ item.budget_amount ?? t('balanceMarketing.unlimited') }}</p><p class="text-xs text-gray-500">{{ t('balanceMarketing.reserved') }}: {{ Number(item.reserved_amount) }} / {{ t('balanceMarketing.used') }}: {{ Number(item.redeemed_amount) }}</p></td><td class="p-3">{{ t(item.enabled ? 'balanceMarketing.enabled' : 'balanceMarketing.disabled') }}</td><td class="p-3 text-right"><button class="btn btn-secondary !p-2" :title="t('common.edit')" @click="openPromotion(item)"><Icon name="edit" size="sm" /><span class="sr-only">{{ t('common.edit') }}</span></button><button class="btn btn-secondary ml-2 !p-2" :title="t('common.delete')" @click="removeTarget = { id: item.id, tier: false }"><Icon name="trash" size="sm" /><span class="sr-only">{{ t('common.delete') }}</span></button></td></tr>
            <tr v-if="!visiblePromotions.length"><td colspan="5" class="p-8 text-center text-gray-500">{{ t('balanceMarketing.empty') }}</td></tr></tbody>
        </table>
      </div>
    </template>
    <BaseDialog :show="dialog !== null" :title="t(dialog === 'tier' ? 'balanceMarketing.tier' : 'balanceMarketing.tabs.recharge')" size="lg" @close="!busy && (dialog = null)">
      <form id="marketing-form" class="space-y-4" @submit.prevent="save">
        <label class="block"><span class="input-label">{{ t('balanceMarketing.name') }}</span><input v-model.trim="form.name" class="input mt-1" required :maxlength="dialog === 'tier' ? 64 : 128" /></label>
        <div class="grid gap-4 sm:grid-cols-2">
          <label><span class="input-label">{{ t('balanceMarketing.currency') }}</span><input v-model.trim="form.settlement_currency" class="input mt-1" required maxlength="3" :disabled="!!editingId && dialog === 'promotion'" /></label>
          <label><span class="input-label">{{ t('balanceMarketing.reduction') }} (%)</span><input v-model="form.discount_percent" type="number" class="input mt-1" required :min="dialog === 'tier' ? 0 : 0.0001" max="99.9999" step="0.0001" /></label>
        </div>
        <p class="rounded-lg bg-primary-50 p-3 text-sm text-primary-800 dark:bg-primary-950/30 dark:text-primary-200">{{ t('balanceMarketing.preview', { fold: Number(((100 - Number(form.discount_percent)) / 10).toFixed(4)), amount: (100 - Number(form.discount_percent)).toFixed(2) }) }}</p>
        <label v-if="dialog === 'tier'" class="block"><span class="input-label">{{ t('balanceMarketing.threshold') }}</span><input v-model="form.threshold_amount" type="number" class="input mt-1" required min="0" step="0.00000001" /></label>
        <template v-else>
          <label class="block"><span class="input-label">{{ t('balanceMarketing.description') }}</span><textarea v-model="form.description" class="input mt-1" maxlength="4000" rows="2" /></label>
          <div class="grid gap-4 sm:grid-cols-2"><label><span class="input-label">{{ t('balanceMarketing.start') }}</span><input v-model="form.starts_at" type="datetime-local" class="input mt-1" required /></label><label><span class="input-label">{{ t('balanceMarketing.end') }}</span><input v-model="form.ends_at" type="datetime-local" class="input mt-1" required :min="form.starts_at" /></label></div>
          <p class="text-xs text-gray-500">{{ t('balanceMarketing.localTime', { timezone: browserTimezone }) }}</p>
          <label class="block"><span class="input-label">{{ t('balanceMarketing.timezone') }}</span><input v-model.trim="form.timezone" class="input mt-1" required /></label>
          <div class="grid gap-4 sm:grid-cols-2"><label v-for="field in amountFields" :key="field.key"><span class="input-label">{{ t(`balanceMarketing.${field.label}`) }}</span><input v-model="form[field.key]" type="number" class="input mt-1" min="0" step="0.001" :placeholder="t('balanceMarketing.unlimited')" /></label></div>
        </template>
        <div class="flex items-center justify-between gap-4"><span>{{ t('balanceMarketing.enabled') }}</span><Toggle v-model="form.enabled" :aria-label="t('balanceMarketing.enabled')" /></div>
        <p v-if="formError" role="alert" class="text-sm text-red-600">{{ formError }}</p>
      </form>
      <template #footer><button class="btn btn-secondary" :disabled="busy" @click="dialog = null">{{ t('common.cancel') }}</button><button form="marketing-form" type="submit" class="btn btn-primary" :disabled="busy">{{ t('common.save') }}</button></template>
    </BaseDialog>
    <ConfirmDialog :show="!!removeTarget" :title="t('common.delete')" :message="t('balanceMarketing.remove')" danger @cancel="removeTarget = null" @confirm="remove" />
    <TotpStepUpDialog :controller="stepUp" />
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { balanceMarketingAPI, type MembershipTier, type RechargePromotion } from '@/api/admin/balanceMarketing'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import { isStepUpCancelled, useStepUp } from '@/composables/useStepUp'
import { useAppStore } from '@/stores'
const props = defineProps<{ mode: 'recharge' | 'membership' }>()
const { t } = useI18n()
const app = useAppStore()
const stepUp = useStepUp()
const config = ref({ settlement_currency: 'CNY', affiliate_commission_rate: '0' })
const currency = ref('CNY')
const tiers = ref<MembershipTier[]>([])
const promotions = ref<RechargePromotion[]>([])
const visiblePromotions = computed(() => promotions.value.filter(p => p.kind === (props.mode === 'membership' ? 'first_recharge' : 'recharge')))
const loading = ref(false), busy = ref(false), error = ref(''), formError = ref('')
const dialog = ref<'tier' | 'promotion' | null>(null)
const editingId = ref<number>()
const removeTarget = ref<{ id: number; tier: boolean } | null>(null)
const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone
const emptyForm = () => ({ name: '', settlement_currency: config.value.settlement_currency, discount_percent: '0', threshold_amount: '0', enabled: false, description: '', starts_at: '', ends_at: '', timezone: browserTimezone, min_amount: '', max_amount: '', max_discount_amount: '', budget_amount: '', sort_order: 0 })
const form = reactive(emptyForm())
const amountFields = [{ key: 'min_amount', label: 'minimum' }, { key: 'max_amount', label: 'maximum' }, { key: 'max_discount_amount', label: 'cap' }, { key: 'budget_amount', label: 'budget' }] as const
const date = (value: string) => value ? new Date(value).toLocaleString() : '-'
const localDate = (value: string, timezone: string) => {
  if (!value) return ''
  const date = new Date(value)
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: timezone, year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).formatToParts(date)
  const get = (name: string) => parts.find((part) => part.type === name)?.value || '00'
  return `${get('year')}-${get('month')}-${get('day')}T${get('hour')}:${get('minute')}`
}
function zonedLocalToUtc(value: string, timezone: string): string {
  if (!value) return ''
  const [date, time] = value.split('T'); const [year, month, day] = date.split('-').map(Number); const [hour, minute] = time.split(':').map(Number)
  const naive = Date.UTC(year, month - 1, day, hour, minute)
  const parts = new Intl.DateTimeFormat('en-US', { timeZone: timezone, year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23' }).formatToParts(new Date(naive))
  const get = (name: string) => Number(parts.find((part) => part.type === name)?.value || 0)
  const represented = Date.UTC(get('year'), get('month') - 1, get('day'), get('hour'), get('minute'), get('second'))
  return new Date(naive - (represented - naive)).toISOString()
}
const message = (e: unknown) => (e as { message?: string })?.message || t('balanceMarketing.error')
async function load() {
  loading.value = true; error.value = ''
  try {
    config.value = (await balanceMarketingAPI.config()).data; currency.value = config.value.settlement_currency
    tiers.value = (await balanceMarketingAPI.tiers()).data
    promotions.value = (await balanceMarketingAPI.promotions()).data
  } catch (e) { error.value = message(e) } finally { loading.value = false }
}
function openTier(item?: MembershipTier) { Object.assign(form, emptyForm(), item || {}); editingId.value = item?.id; formError.value = ''; dialog.value = 'tier' }
function openPromotion(item?: RechargePromotion) {
  Object.assign(form, emptyForm(), item || {})
  for (const field of amountFields) form[field.key] = item?.[field.key] ?? ''
  form.timezone = item?.timezone || browserTimezone
  form.starts_at = localDate(item?.starts_at || '', form.timezone); form.ends_at = localDate(item?.ends_at || '', form.timezone)
  editingId.value = item?.id; formError.value = ''; dialog.value = 'promotion'
}
async function save() {
  if (busy.value) return
  busy.value = true; formError.value = ''
  try {
    if (dialog.value === 'tier') {
      const data = { name: form.name, settlement_currency: form.settlement_currency.toUpperCase(), threshold_amount: String(form.threshold_amount), discount_percent: String(form.discount_percent), sort_order: form.sort_order, enabled: form.enabled }
      await stepUp.run(() => balanceMarketingAPI.saveTier(data, editingId.value))
    } else {
      const data = { kind: props.mode === 'membership' ? 'first_recharge' as const : 'recharge' as const, name: form.name, description: form.description, settlement_currency: form.settlement_currency.toUpperCase(), enabled: form.enabled, starts_at: zonedLocalToUtc(form.starts_at, form.timezone), ends_at: zonedLocalToUtc(form.ends_at, form.timezone), timezone: form.timezone, discount_percent: String(form.discount_percent), min_amount: form.min_amount === '' ? null : String(form.min_amount), max_amount: form.max_amount === '' ? null : String(form.max_amount), max_discount_amount: form.max_discount_amount === '' ? null : String(form.max_discount_amount), budget_amount: form.budget_amount === '' ? null : String(form.budget_amount) }
      await stepUp.run(() => balanceMarketingAPI.savePromotion(data, editingId.value))
    }
    dialog.value = null; app.showSuccess(t('balanceMarketing.saved')); await load()
  } catch (e) { if (!isStepUpCancelled(e)) formError.value = message(e) } finally { busy.value = false }
}
async function saveCurrency() {
  if (busy.value) return
  busy.value = true
  try { await stepUp.run(() => balanceMarketingAPI.saveConfig(currency.value.toUpperCase())); app.showSuccess(t('balanceMarketing.saved')); await load() }
  catch (e) { if (!isStepUpCancelled(e)) error.value = message(e) } finally { busy.value = false }
}
async function remove() {
  if (!removeTarget.value || busy.value) return
  const item = removeTarget.value; busy.value = true
  try { await stepUp.run(() => item.tier ? balanceMarketingAPI.deleteTier(item.id) : balanceMarketingAPI.deletePromotion(item.id)); removeTarget.value = null; await load() }
  catch (e) { if (!isStepUpCancelled(e)) error.value = message(e) } finally { busy.value = false }
}
onMounted(load)
</script>
