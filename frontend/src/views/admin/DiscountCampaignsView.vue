<template>
  <AppLayout>
    <div class="space-y-5">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex items-start gap-3 rounded-lg border border-emerald-200 bg-emerald-50/70 px-4 py-3 text-sm text-emerald-900 dark:border-emerald-900/60 dark:bg-emerald-950/25 dark:text-emerald-200">
          <Icon name="infoCircle" size="md" class="mt-0.5 shrink-0" />
          <div>
            <p class="font-semibold">{{ t('admin.discountCampaigns.safetyTitle') }}</p>
            <p class="mt-0.5 leading-5 opacity-80">{{ t('admin.discountCampaigns.safetyHint') }}</p>
          </div>
        </div>
        <button class="btn btn-primary shrink-0" @click="openCreate">
          <Icon name="plus" size="sm" />
          {{ t('admin.discountCampaigns.create') }}
        </button>
      </div>

      <div class="flex items-center justify-between gap-3">
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.discountCampaigns.hints.overlap') }}</p>
        <button class="btn btn-secondary !px-3" :disabled="loading" :title="t('common.refresh')" @click="loadCampaigns">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
        </button>
      </div>

      <div class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div v-if="loading" class="flex min-h-52 items-center justify-center">
          <Icon name="refresh" size="lg" class="animate-spin text-gray-400" />
        </div>
        <div v-else-if="campaigns.length === 0" class="px-6 py-16 text-center">
          <Icon name="calendar" size="xl" class="mx-auto text-gray-300 dark:text-dark-500" />
          <p class="mt-3 text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('admin.discountCampaigns.empty') }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.discountCampaigns.emptyHint') }}</p>
        </div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50/80 text-left text-xs font-medium uppercase text-gray-500 dark:bg-dark-900/50 dark:text-gray-400">
              <tr>
                <th class="px-4 py-3">{{ t('admin.discountCampaigns.columns.campaign') }}</th>
                <th class="px-4 py-3">{{ t('admin.discountCampaigns.columns.schedule') }}</th>
                <th class="px-4 py-3">{{ t('admin.discountCampaigns.columns.discount') }}</th>
                <th class="px-4 py-3">{{ t('admin.discountCampaigns.columns.budget') }}</th>
                <th class="px-4 py-3">{{ t('admin.discountCampaigns.columns.status') }}</th>
                <th class="px-4 py-3 text-right">{{ t('admin.discountCampaigns.columns.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="campaign in campaigns" :key="campaign.id" class="hover:bg-gray-50/60 dark:hover:bg-dark-700/30">
                <td class="px-4 py-3">
                  <p class="font-medium text-gray-900 dark:text-white">{{ campaign.name }}</p>
                  <p class="mt-0.5 text-xs text-gray-500">#{{ campaign.id }} · {{ campaign.timezone }}</p>
                </td>
                <td class="px-4 py-3 text-gray-700 dark:text-gray-300">
                  <p>{{ scheduleLabel(campaign) }}</p>
                  <p v-if="campaign.schedule_type === 'weekly'" class="mt-0.5 text-xs text-gray-500">{{ campaign.all_day ? t('admin.discountCampaigns.allDay') : `${campaign.start_time} - ${campaign.end_time}` }}</p>
                </td>
                <td class="px-4 py-3">
                  <span class="inline-flex rounded-md bg-emerald-50 px-2 py-1 font-semibold text-emerald-700 dark:bg-emerald-900/25 dark:text-emerald-300">
                    {{ discountLabel(campaign.discount_factor) }}
                  </span>
                  <p v-if="campaign.min_effective_multiplier" class="mt-1 text-xs text-gray-500">≥ {{ formatAmount(campaign.min_effective_multiplier) }}x</p>
                </td>
                <td class="px-4 py-3 font-mono text-xs text-gray-700 dark:text-gray-300">
                  ${{ formatAmount(campaign.discount_spent) }} / {{ campaign.budget_cap ? `$${formatAmount(campaign.budget_cap)}` : t('admin.discountCampaigns.noLimit') }}
                </td>
                <td class="px-4 py-3">
                  <span :class="statusClass(campaign.status)" class="inline-flex rounded-full px-2 py-1 text-xs font-medium">
                    {{ t(`admin.discountCampaigns.statuses.${campaign.status}`) }}
                  </span>
                </td>
                <td class="px-4 py-3">
                  <div class="flex justify-end gap-1">
                    <button class="rounded-md p-2 text-gray-500 hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700" :title="t('common.edit')" @click="openEdit(campaign)">
                      <Icon name="edit" size="sm" />
                    </button>
                    <button class="rounded-md p-2 text-gray-500 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/30" :title="t('admin.discountCampaigns.delete')" @click="pendingDelete = campaign">
                      <Icon name="trash" size="sm" />
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <BaseDialog :show="formVisible" :title="editingID ? t('admin.discountCampaigns.edit') : t('admin.discountCampaigns.create')" width="wide" @close="closeForm">
        <form id="discount-campaign-form" class="space-y-6" @submit.prevent="saveCampaign">
          <div class="grid gap-4 md:grid-cols-2">
            <div class="md:col-span-2">
              <label class="input-label">{{ t('admin.discountCampaigns.fields.name') }}</label>
              <input v-model.trim="form.name" class="input mt-1" maxlength="120" required />
            </div>

            <div>
              <label class="input-label">{{ t('admin.discountCampaigns.fields.scheduleType') }}</label>
              <div class="mt-1 grid grid-cols-2 rounded-lg bg-gray-100 p-1 dark:bg-dark-900">
                <button v-for="type in scheduleTypes" :key="type" type="button" class="rounded-md px-3 py-2 text-sm font-medium transition-colors" :class="form.schedule_type === type ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-700 dark:text-primary-300' : 'text-gray-500 dark:text-gray-400'" @click="form.schedule_type = type">
                  {{ t(`admin.discountCampaigns.scheduleTypes.${type}`) }}
                </button>
              </div>
            </div>
            <div>
              <label class="input-label">{{ t('admin.discountCampaigns.fields.timezone') }}</label>
              <input v-model.trim="form.timezone" class="input mt-1" required placeholder="Asia/Shanghai" />
            </div>

            <template v-if="form.schedule_type === 'one_time'">
              <div>
                <label class="input-label">{{ t('admin.discountCampaigns.fields.startsAt') }}</label>
                <input v-model="form.starts_at" type="datetime-local" class="input mt-1" required />
              </div>
              <div>
                <label class="input-label">{{ t('admin.discountCampaigns.fields.endsAt') }}</label>
                <input v-model="form.ends_at" type="datetime-local" class="input mt-1" required />
              </div>
            </template>

            <template v-else>
              <fieldset class="md:col-span-2">
                <legend class="input-label">{{ t('admin.discountCampaigns.fields.weekdays') }}</legend>
                <div class="mt-2 flex flex-wrap gap-2">
                  <label v-for="day in weekdayOptions" :key="day.value" class="cursor-pointer rounded-md border px-3 py-2 text-sm transition-colors" :class="form.weekdays.includes(day.value) ? 'border-primary-400 bg-primary-50 text-primary-700 dark:border-primary-700 dark:bg-primary-950/30 dark:text-primary-300' : 'border-gray-200 text-gray-600 dark:border-dark-600 dark:text-gray-300'">
                    <input v-model="form.weekdays" :value="day.value" type="checkbox" class="sr-only" />
                    {{ t(`admin.discountCampaigns.weekdays.${day.key}`) }}
                  </label>
                </div>
              </fieldset>
              <div class="md:col-span-2 flex items-center justify-between rounded-lg border border-gray-200 px-4 py-3 dark:border-dark-600">
                <label class="font-medium text-gray-800 dark:text-gray-200">{{ t('admin.discountCampaigns.fields.allDay') }}</label>
                <button type="button" role="switch" :aria-checked="form.all_day" class="relative h-6 w-11 rounded-full transition-colors" :class="form.all_day ? 'bg-primary-600' : 'bg-gray-300 dark:bg-dark-600'" @click="form.all_day = !form.all_day">
                  <span class="absolute top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform" :class="form.all_day ? 'translate-x-5' : 'translate-x-0.5'" />
                </button>
              </div>
              <template v-if="!form.all_day">
                <div>
                  <label class="input-label">{{ t('admin.discountCampaigns.fields.startTime') }}</label>
                  <input v-model="form.start_time" type="time" class="input mt-1" required />
                </div>
                <div>
                  <label class="input-label">{{ t('admin.discountCampaigns.fields.endTime') }}</label>
                  <input v-model="form.end_time" type="time" class="input mt-1" required />
                </div>
                <p class="md:col-span-2 -mt-2 text-xs text-gray-500">{{ t('admin.discountCampaigns.hints.crossMidnight') }}</p>
              </template>
            </template>
          </div>

          <div class="grid gap-4 rounded-lg border border-gray-200 bg-gray-50/60 p-4 md:grid-cols-3 dark:border-dark-600 dark:bg-dark-900/40">
            <div>
              <label class="input-label">{{ t('admin.discountCampaigns.fields.discountPercent') }}</label>
              <div class="relative mt-1">
                <input v-model.number="form.discount_percent" type="number" min="0.01" max="99.99" step="0.01" class="input pr-9" required />
                <span class="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-gray-500">%</span>
              </div>
              <p class="mt-1.5 text-xs text-gray-500">{{ t('admin.discountCampaigns.hints.discountPercent') }}</p>
            </div>
            <div>
              <label class="input-label">{{ t('admin.discountCampaigns.fields.minMultiplier') }}</label>
              <input v-model.trim="form.min_effective_multiplier" inputmode="decimal" class="input mt-1" placeholder="0.5" />
              <p class="mt-1.5 text-xs text-gray-500">{{ t('admin.discountCampaigns.hints.minimum') }}</p>
            </div>
            <div>
              <label class="input-label">{{ t('admin.discountCampaigns.fields.budgetCap') }}</label>
              <input v-model.trim="form.budget_cap" inputmode="decimal" class="input mt-1" placeholder="100" />
              <p class="mt-1.5 text-xs text-gray-500">{{ t('admin.discountCampaigns.hints.budget') }}</p>
            </div>
          </div>

          <div class="flex items-center justify-between rounded-lg border border-gray-200 px-4 py-3 dark:border-dark-600">
            <div>
              <p class="font-medium text-gray-800 dark:text-gray-200">{{ t('admin.discountCampaigns.fields.enabled') }}</p>
              <p class="mt-0.5 text-xs text-gray-500">{{ form.enabled ? t('admin.discountCampaigns.enabled') : t('admin.discountCampaigns.disabled') }}</p>
            </div>
            <button type="button" role="switch" :aria-checked="form.enabled" class="relative h-6 w-11 rounded-full transition-colors" :class="form.enabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-dark-600'" @click="form.enabled = !form.enabled">
              <span class="absolute top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform" :class="form.enabled ? 'translate-x-5' : 'translate-x-0.5'" />
            </button>
          </div>
        </form>
        <template #footer>
          <button class="btn btn-secondary" :disabled="saving" @click="closeForm">{{ t('common.cancel') }}</button>
          <button form="discount-campaign-form" type="submit" class="btn btn-primary" :disabled="!formValid || saving">
            <Icon name="check" size="sm" />
            {{ saving ? t('admin.discountCampaigns.saving') : t('admin.discountCampaigns.save') }}
          </button>
        </template>
      </BaseDialog>

      <ConfirmDialog
        :show="!!pendingDelete"
        :title="t('admin.discountCampaigns.deleteTitle')"
        :message="t('admin.discountCampaigns.deleteConfirm', { name: pendingDelete?.name || '' })"
        :confirm-text="t('admin.discountCampaigns.delete')"
        danger
        @cancel="pendingDelete = null"
        @confirm="deleteCampaign"
      />
      <TotpStepUpDialog :controller="stepUp" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import { adminAPI } from '@/api/admin'
import type { DiscountCampaign, DiscountCampaignRequest, DiscountScheduleType } from '@/api/admin/discountCampaigns'
import { useAppStore } from '@/stores'
import { isStepUpCancelled, useStepUp } from '@/composables/useStepUp'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()
const stepUp = useStepUp()
const campaigns = ref<DiscountCampaign[]>([])
const loading = ref(false)
const saving = ref(false)
const formVisible = ref(false)
const editingID = ref<number | null>(null)
const pendingDelete = ref<DiscountCampaign | null>(null)
const scheduleTypes: DiscountScheduleType[] = ['one_time', 'weekly']
const weekdayOptions = [
  { value: 1, key: 'mon' }, { value: 2, key: 'tue' }, { value: 3, key: 'wed' },
  { value: 4, key: 'thu' }, { value: 5, key: 'fri' }, { value: 6, key: 'sat' }, { value: 0, key: 'sun' }
] as const

interface DiscountForm {
  name: string
  enabled: boolean
  schedule_type: DiscountScheduleType
  timezone: string
  starts_at: string
  ends_at: string
  weekdays: number[]
  start_time: string
  end_time: string
  all_day: boolean
  discount_percent: number
  min_effective_multiplier: string
  budget_cap: string
}

const form = reactive<DiscountForm>(emptyForm())

function emptyForm(): DiscountForm {
  const start = new Date(Date.now() + 60 * 60 * 1000)
  const end = new Date(start.getTime() + 24 * 60 * 60 * 1000)
  return {
    name: '', enabled: false, schedule_type: 'one_time', timezone: 'Asia/Shanghai',
    starts_at: toLocalInput(start), ends_at: toLocalInput(end), weekdays: [0],
    start_time: '00:00', end_time: '23:59', all_day: true, discount_percent: 90,
    min_effective_multiplier: '', budget_cap: ''
  }
}

const formValid = computed(() => {
  if (!form.name.trim() || !form.timezone.trim() || form.discount_percent <= 0 || form.discount_percent >= 100) return false
  if (form.schedule_type === 'one_time') {
    return Boolean(form.starts_at && form.ends_at && new Date(form.starts_at) < new Date(form.ends_at))
  }
  return form.weekdays.length > 0 && (form.all_day || Boolean(form.start_time && form.end_time && form.start_time !== form.end_time))
})

async function loadCampaigns() {
  loading.value = true
  try { campaigns.value = await adminAPI.discountCampaigns.list() }
  catch (error: any) { appStore.showError(error?.message || t('admin.discountCampaigns.errors.load')) }
  finally { loading.value = false }
}

function openCreate() {
  Object.assign(form, emptyForm())
  editingID.value = null
  formVisible.value = true
}

function openEdit(campaign: DiscountCampaign) {
  Object.assign(form, {
    name: campaign.name, enabled: campaign.enabled, schedule_type: campaign.schedule_type,
    timezone: campaign.timezone, starts_at: campaign.starts_at ? toLocalInput(new Date(campaign.starts_at)) : '',
    ends_at: campaign.ends_at ? toLocalInput(new Date(campaign.ends_at)) : '', weekdays: [...campaign.weekdays],
    start_time: campaign.start_time || '00:00', end_time: campaign.end_time || '23:59', all_day: campaign.all_day,
    discount_percent: Number(campaign.discount_factor) * 100,
    min_effective_multiplier: campaign.min_effective_multiplier || '', budget_cap: campaign.budget_cap || ''
  })
  editingID.value = campaign.id
  formVisible.value = true
}

function closeForm() { if (!saving.value) formVisible.value = false }

function buildPayload(): DiscountCampaignRequest {
  const weekly = form.schedule_type === 'weekly'
  return {
    name: form.name.trim(), enabled: form.enabled, schedule_type: form.schedule_type,
    timezone: form.timezone.trim(),
    starts_at: weekly ? undefined : new Date(form.starts_at).toISOString(),
    ends_at: weekly ? undefined : new Date(form.ends_at).toISOString(),
    weekdays: weekly ? [...form.weekdays].sort((a, b) => a - b) : [],
    start_time: weekly && !form.all_day ? form.start_time : undefined,
    end_time: weekly && !form.all_day ? form.end_time : undefined,
    all_day: weekly && form.all_day,
    discount_factor: (form.discount_percent / 100).toFixed(6),
    min_effective_multiplier: form.min_effective_multiplier.trim() || undefined,
    budget_cap: form.budget_cap.trim() || undefined
  }
}

async function saveCampaign() {
  if (!formValid.value) return
  saving.value = true
  try {
    const id = editingID.value
    await stepUp.run(() => id ? adminAPI.discountCampaigns.update(id, buildPayload()) : adminAPI.discountCampaigns.create(buildPayload()))
    appStore.showSuccess(t(id ? 'admin.discountCampaigns.updated' : 'admin.discountCampaigns.created'))
    formVisible.value = false
    await loadCampaigns()
  } catch (error: any) {
    if (!isStepUpCancelled(error)) appStore.showError(error?.message || t('admin.discountCampaigns.errors.save'))
  } finally { saving.value = false }
}

async function deleteCampaign() {
  const campaign = pendingDelete.value
  if (!campaign) return
  pendingDelete.value = null
  try {
    await stepUp.run(() => adminAPI.discountCampaigns.remove(campaign.id))
    appStore.showSuccess(t('admin.discountCampaigns.deleted'))
    await loadCampaigns()
  } catch (error: any) {
    if (!isStepUpCancelled(error)) appStore.showError(error?.message || t('admin.discountCampaigns.errors.delete'))
  }
}

function toLocalInput(value: Date): string {
  const local = new Date(value.getTime() - value.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}

function discountLabel(factor: string): string {
  const percent = Number(factor) * 100
  return `${Number(percent.toFixed(2))}% (${Number((percent / 10).toFixed(2))}折)`
}

function formatAmount(value: string): string {
  const number = Number(value)
  return Number.isFinite(number) ? number.toFixed(8).replace(/\.?0+$/, '') : value
}

function scheduleLabel(campaign: DiscountCampaign): string {
  if (campaign.schedule_type === 'one_time') return `${formatDateTime(campaign.starts_at)} - ${formatDateTime(campaign.ends_at)}`
  const labels = weekdayOptions.filter((day) => campaign.weekdays.includes(day.value)).map((day) => t(`admin.discountCampaigns.weekdays.${day.key}`))
  return labels.join('、')
}

function statusClass(status: DiscountCampaign['status']): string {
  if (status === 'active') return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/25 dark:text-emerald-300'
  if (status === 'upcoming') return 'bg-blue-50 text-blue-700 dark:bg-blue-900/25 dark:text-blue-300'
  if (status === 'budget_exhausted') return 'bg-amber-50 text-amber-700 dark:bg-amber-900/25 dark:text-amber-300'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
}

onMounted(loadCampaigns)
</script>
