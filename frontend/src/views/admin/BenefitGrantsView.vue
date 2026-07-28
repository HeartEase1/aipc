<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <header class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <div class="mb-2 flex items-center gap-3">
            <span class="flex h-11 w-11 items-center justify-center rounded-lg bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
              <Icon name="gift" size="lg" />
            </span>
            <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('admin.benefitGrants.title') }}</h1>
          </div>
          <p class="text-sm text-gray-600 dark:text-gray-400">{{ t('admin.benefitGrants.description') }}</p>
        </div>
        <div class="inline-flex w-full rounded-lg border border-gray-200 bg-white p-1 shadow-sm dark:border-dark-700 dark:bg-dark-800 lg:w-auto">
          <button
            class="flex-1 rounded-md px-4 py-2 text-sm font-medium transition-colors lg:flex-none"
            :class="activeTab === 'create' ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
            @click="activeTab = 'create'"
          >
            {{ t('admin.benefitGrants.tabs.create') }}
          </button>
          <button
            class="flex-1 rounded-md px-4 py-2 text-sm font-medium transition-colors lg:flex-none"
            :class="activeTab === 'history' ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
            @click="activeTab = 'history'; loadBatches()"
          >
            {{ t('admin.benefitGrants.tabs.history') }}
          </button>
        </div>
      </header>

      <div v-if="activeTab === 'create'" class="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <form class="space-y-6" @submit.prevent="createPreview">
          <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800 sm:p-6">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.benefitGrants.sections.rules') }}</h2>
            <div class="mt-5 grid gap-6 md:grid-cols-2">
              <fieldset>
                <legend class="input-label">{{ t('admin.benefitGrants.fields.type') }}</legend>
                <div class="mt-2 grid grid-cols-2 rounded-lg bg-gray-100 p-1 dark:bg-dark-900">
                  <button v-for="type in grantTypes" :key="type" type="button" class="rounded-md px-3 py-2 text-sm font-medium" :class="form.grant_type === type ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 dark:text-gray-400'" @click="form.grant_type = type">
                    {{ t(`admin.benefitGrants.types.${type}`) }}
                  </button>
                </div>
              </fieldset>
              <fieldset>
                <legend class="input-label">{{ t('admin.benefitGrants.fields.mode') }}</legend>
                <div class="mt-2 grid grid-cols-2 rounded-lg bg-gray-100 p-1 dark:bg-dark-900">
                  <button v-for="mode in grantModes" :key="mode" type="button" class="rounded-md px-3 py-2 text-sm font-medium" :class="form.grant_mode === mode ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 dark:text-gray-400'" @click="setGrantMode(mode)">
                    {{ t(`admin.benefitGrants.modes.${mode}`) }}
                  </button>
                </div>
              </fieldset>
            </div>

            <div class="mt-6">
              <label class="input-label">{{ t('admin.benefitGrants.fields.audience') }}</label>
              <div class="mt-2 grid gap-3 sm:grid-cols-2">
                <label v-for="audience in audiences" :key="audience" class="flex cursor-pointer items-start gap-3 rounded-lg border p-4" :class="form.audience_type === audience ? 'border-primary-500 bg-primary-50/50 dark:bg-primary-900/10' : 'border-gray-200 dark:border-dark-600'">
                  <input v-model="form.audience_type" type="radio" :value="audience" class="mt-0.5 h-4 w-4 text-primary-600" />
                  <span>
                    <span class="block text-sm font-medium text-gray-900 dark:text-white">{{ t(`admin.benefitGrants.audiences.${audience}`) }}</span>
                    <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">{{ t(`admin.benefitGrants.audienceHints.${audience}`) }}</span>
                  </span>
                </label>
              </div>
            </div>

            <div v-if="form.audience_type === 'selected'" class="mt-5 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-900/40">
              <div class="relative">
                <Icon name="search" size="sm" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                <input v-model="userSearch" class="input pl-9" :placeholder="t('admin.benefitGrants.searchUsers')" @input="searchUsers" />
              </div>
              <div v-if="userResults.length" class="mt-2 max-h-52 overflow-y-auto rounded-lg border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800">
                <button v-for="user in userResults" :key="user.id" type="button" class="flex w-full items-center justify-between gap-3 border-b border-gray-100 px-4 py-3 text-left last:border-0 hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-700" @click="toggleUser(user)">
                  <span class="min-w-0">
                    <span class="block truncate text-sm font-medium text-gray-900 dark:text-white">{{ user.username || user.email }}</span>
                    <span class="block truncate text-xs text-gray-500">ID {{ user.id }} · {{ user.email }}</span>
                  </span>
                  <Icon :name="selectedUsers.has(user.id) ? 'checkCircle' : 'plus'" size="sm" :class="selectedUsers.has(user.id) ? 'text-emerald-600' : 'text-gray-400'" />
                </button>
              </div>
              <div v-if="selectedUserList.length" class="mt-3 flex flex-wrap gap-2">
                <button v-for="user in selectedUserList" :key="user.id" type="button" class="inline-flex max-w-full items-center gap-2 rounded-md bg-white px-2.5 py-1.5 text-xs text-gray-700 shadow-sm ring-1 ring-gray-200 dark:bg-dark-800 dark:text-gray-200 dark:ring-dark-600" @click="toggleUser(user)">
                  <span class="truncate">{{ user.username || user.email }}</span>
                  <Icon name="x" size="xs" />
                </button>
              </div>
            </div>

            <div class="mt-6 grid gap-4 sm:grid-cols-2">
              <div v-if="form.grant_mode === 'fixed'">
                <label class="input-label">{{ t('admin.benefitGrants.fields.fixedAmount') }}</label>
                <div class="relative mt-1">
                  <span class="absolute left-3 top-1/2 -translate-y-1/2 text-sm text-gray-500">$</span>
                  <input v-model.trim="form.fixed_amount" required inputmode="decimal" class="input pl-7" placeholder="0.00000000" />
                </div>
              </div>
              <div v-else>
                <label class="input-label">{{ t('admin.benefitGrants.fields.percentage') }}</label>
                <div class="relative mt-1">
                  <input v-model.trim="form.percentage" required inputmode="decimal" class="input pr-8" placeholder="10" />
                  <span class="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-gray-500">%</span>
                </div>
              </div>
            </div>
          </section>

          <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800 sm:p-6">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.benefitGrants.sections.protection') }}</h2>
            <div class="mt-5 grid gap-4 md:grid-cols-3">
              <GuardInput v-model:enabled="guards.min" v-model:value="form.min_amount" :label="t('admin.benefitGrants.fields.minAmount')" :disabled="form.grant_mode === 'fixed'" />
              <GuardInput v-model:enabled="guards.cap" v-model:value="form.per_user_cap" :label="t('admin.benefitGrants.fields.perUserCap')" />
              <GuardInput v-model:enabled="guards.budget" v-model:value="form.total_budget_cap" :label="t('admin.benefitGrants.fields.totalBudgetCap')" />
            </div>
          </section>

          <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800 sm:p-6">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.benefitGrants.sections.notification') }}</h2>
            <div class="mt-5 space-y-4">
              <div>
                <label class="input-label">{{ t('admin.benefitGrants.fields.reason') }}</label>
                <textarea v-model.trim="form.reason" required maxlength="500" rows="2" class="input mt-1 resize-y" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.benefitGrants.fields.notificationTitle') }}</label>
                <input v-model.trim="form.notification_title" required maxlength="200" class="input mt-1" />
              </div>
              <div>
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <label class="input-label">{{ t('admin.benefitGrants.fields.notificationContent') }}</label>
                  <div class="flex flex-wrap gap-1.5">
                    <button v-for="variable in templateVariables" :key="variable" type="button" class="rounded-md bg-gray-100 px-2 py-1 font-mono text-xs text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600" @click="appendVariable(variable)">{{ variable }}</button>
                  </div>
                </div>
                <textarea ref="contentInput" v-model="form.notification_content" required maxlength="10000" rows="7" class="input mt-1 resize-y font-mono text-sm" />
                <button type="button" class="btn btn-secondary mt-3" @click="showNotificationPreview = true">
                  <Icon name="eye" size="sm" />
                  <span class="ml-2">{{ t('admin.benefitGrants.previewNotification') }}</span>
                </button>
              </div>
            </div>
          </section>

          <button type="submit" class="btn btn-primary w-full py-3" :disabled="previewing || !canPreview">
            <Icon name="eye" size="sm" />
            <span class="ml-2">{{ previewing ? t('admin.benefitGrants.previewing') : t('admin.benefitGrants.preview') }}</span>
          </button>
        </form>

        <aside class="space-y-4 xl:sticky xl:top-24">
          <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.benefitGrants.summary.title') }}</h2>
            <dl class="mt-4 space-y-3 text-sm">
              <SummaryRow :label="t('admin.benefitGrants.fields.type')" :value="t(`admin.benefitGrants.types.${form.grant_type}`)" />
              <SummaryRow :label="t('admin.benefitGrants.fields.mode')" :value="t(`admin.benefitGrants.modes.${form.grant_mode}`)" />
              <SummaryRow :label="t('admin.benefitGrants.fields.audience')" :value="form.audience_type === 'all' ? t('admin.benefitGrants.audiences.all') : t('admin.benefitGrants.selectedCount', { count: selectedUsers.size })" />
            </dl>
          </section>
          <section class="rounded-lg border border-amber-200 bg-amber-50 p-5 text-sm text-amber-900 shadow-sm dark:border-amber-900/50 dark:bg-amber-900/10 dark:text-amber-200">
            <div class="flex items-start gap-3">
              <Icon name="shield" size="md" class="mt-0.5 shrink-0" />
              <div>
                <p class="font-medium">{{ t('admin.benefitGrants.safety.title') }}</p>
                <p class="mt-1 leading-6 opacity-80">{{ t('admin.benefitGrants.safety.content') }}</p>
              </div>
            </div>
          </section>
        </aside>
      </div>

      <section v-else class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <Select v-model="historyStatus" :options="statusOptions" class="w-full sm:w-48" @change="historyPage = 1; loadBatches()" />
          <button class="btn btn-secondary" :disabled="historyLoading" @click="loadBatches">
            <Icon name="refresh" size="sm" :class="historyLoading ? 'animate-spin' : ''" />
            <span class="ml-2">{{ t('common.refresh') }}</span>
          </button>
        </div>
        <div v-if="historyLoading && !batches.length" class="flex min-h-64 items-center justify-center"><LoadingSpinner /></div>
        <div v-else-if="!batches.length" class="px-6 py-16 text-center text-sm text-gray-500">{{ t('admin.benefitGrants.emptyHistory') }}</div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-900/50">
              <tr>
                <th v-for="heading in historyHeadings" :key="heading" class="whitespace-nowrap px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">{{ heading }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="batch in batches" :key="batch.id" class="hover:bg-gray-50 dark:hover:bg-dark-700/40">
                <td class="px-5 py-4 text-sm font-medium text-gray-900 dark:text-white">#{{ batch.id }}</td>
                <td class="px-5 py-4 text-sm"><span class="rounded-md bg-gray-100 px-2 py-1 dark:bg-dark-700">{{ t(`admin.benefitGrants.types.${batch.grant_type}`) }}</span></td>
                <td class="px-5 py-4 text-sm text-gray-600 dark:text-gray-300">{{ t(`admin.benefitGrants.modes.${batch.grant_mode}`) }}</td>
                <td class="px-5 py-4 text-sm text-gray-600 dark:text-gray-300">{{ batch.success_count }}/{{ batch.eligible_count }}</td>
                <td class="px-5 py-4 text-sm font-medium text-gray-900 dark:text-white">${{ formatAmount(batch.distributed_amount || batch.total_amount) }}</td>
                <td class="px-5 py-4"><StatusBadge :status="batch.status" :label="t(`admin.benefitGrants.statuses.${batch.status}`)" /></td>
                <td class="whitespace-nowrap px-5 py-4 text-sm text-gray-500">{{ formatDateTime(batch.created_at) }}</td>
                <td class="px-5 py-4 text-right"><button class="btn btn-ghost btn-sm" @click="openDetail(batch.id)">{{ t('common.view') }}</button></td>
              </tr>
            </tbody>
          </table>
        </div>
        <Pagination v-if="historyTotal > historyPageSize" :total="historyTotal" :page="historyPage" :page-size="historyPageSize" :show-page-size-selector="false" @update:page="changeHistoryPage" />
      </section>

      <BaseDialog :show="!!previewBatch" :title="t('admin.benefitGrants.confirmTitle')" width="wide" :close-on-escape="!executing" @close="closePreview">
        <div v-if="previewBatch" class="space-y-5">
          <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <MetricBox :label="t('admin.benefitGrants.metrics.recipients')" :value="String(previewBatch.eligible_count)" />
            <MetricBox :label="t('admin.benefitGrants.metrics.skipped')" :value="String(previewBatch.skipped_count)" />
            <MetricBox :label="t('admin.benefitGrants.metrics.baseCost')" :value="`$${formatAmount(previewBatch.total_base_cost)}`" />
            <MetricBox :label="t('admin.benefitGrants.metrics.totalAmount')" :value="`$${formatAmount(previewBatch.total_amount)}`" emphasis />
            <MetricBox :label="t('admin.benefitGrants.metrics.average')" :value="`$${formatAmount(previewBatch.average_amount)}`" />
            <MetricBox :label="t('admin.benefitGrants.metrics.maximum')" :value="`$${formatAmount(previewBatch.max_amount)}`" />
          </div>
          <div v-if="previewBatch.window_start" class="rounded-lg bg-gray-50 px-4 py-3 text-sm text-gray-600 dark:bg-dark-900/60 dark:text-gray-300">
            {{ t('admin.benefitGrants.metrics.window') }}: {{ formatDateTime(previewBatch.window_start) }} - {{ formatDateTime(previewBatch.window_end) }}
          </div>
          <div v-if="previewBatch.over_budget" class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm font-medium text-red-700 dark:border-red-900/50 dark:bg-red-900/10 dark:text-red-300">
            {{ t('admin.benefitGrants.overBudget') }}
          </div>
          <label class="flex items-start gap-3 rounded-lg border border-gray-200 p-4 dark:border-dark-600">
            <input v-model="confirmed" type="checkbox" class="mt-0.5 h-4 w-4 rounded text-primary-600" />
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.benefitGrants.confirmAcknowledgement') }}</span>
          </label>
        </div>
        <template #footer>
          <button class="btn btn-secondary" :disabled="executing" @click="closePreview">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" :disabled="!confirmed || previewBatch?.over_budget || executing" @click="executeGrant">
            {{ executing ? t('admin.benefitGrants.executing') : t('admin.benefitGrants.execute') }}
          </button>
        </template>
      </BaseDialog>

      <BaseDialog :show="!!detail" :title="detail ? `#${detail.batch.id} ${t('admin.benefitGrants.detailTitle')}` : ''" width="extra-wide" @close="closeDetail">
        <div v-if="detail" class="space-y-5">
          <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <MetricBox :label="t('admin.benefitGrants.metrics.recipients')" :value="String(detail.batch.eligible_count)" />
            <MetricBox :label="t('admin.benefitGrants.metrics.succeeded')" :value="String(detail.batch.success_count)" />
            <MetricBox :label="t('admin.benefitGrants.metrics.failed')" :value="String(detail.batch.failed_count)" />
            <MetricBox :label="t('admin.benefitGrants.metrics.distributed')" :value="`$${formatAmount(detail.batch.distributed_amount)}`" emphasis />
          </div>
          <div class="max-h-[50vh] overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
            <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
              <thead class="sticky top-0 bg-gray-50 dark:bg-dark-800"><tr><th class="px-4 py-3 text-left">ID</th><th class="px-4 py-3 text-left">{{ t('admin.benefitGrants.fields.user') }}</th><th class="px-4 py-3 text-right">{{ t('admin.benefitGrants.metrics.baseCost') }}</th><th class="px-4 py-3 text-right">{{ t('admin.benefitGrants.metrics.amount') }}</th><th class="px-4 py-3 text-left">{{ t('common.status') }}</th></tr></thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700"><tr v-for="item in detail.items" :key="item.id"><td class="px-4 py-3">{{ item.user_id }}</td><td class="px-4 py-3"><span class="block font-medium">{{ item.username || item.email }}</span><span class="block text-xs text-gray-500">{{ item.email }}</span></td><td class="px-4 py-3 text-right">${{ formatAmount(item.base_cost) }}</td><td class="px-4 py-3 text-right font-medium">${{ formatAmount(item.amount) }}</td><td class="px-4 py-3"><StatusBadge :status="item.status" :label="t(`admin.benefitGrants.itemStatuses.${item.status}`)" /></td></tr></tbody>
            </table>
          </div>
          <Pagination v-if="detail.total > detail.page_size" :total="detail.total" :page="detail.page" :page-size="detail.page_size" :show-page-size-selector="false" @update:page="changeDetailPage" />
        </div>
        <template #footer>
          <button class="btn btn-secondary" @click="exportDetail"><Icon name="download" size="sm" /><span class="ml-2">{{ t('common.export') }}</span></button>
          <button v-if="detail && detail.batch.failed_count > 0" class="btn btn-primary" @click="retryDetail">{{ t('admin.benefitGrants.retryFailed') }}</button>
        </template>
      </BaseDialog>

      <AnnouncementPopup v-if="showNotificationPreview" :announcement="notificationPreview" preview @close="showNotificationPreview = false" />
      <TotpStepUpDialog :controller="stepUp" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { saveAs } from 'file-saver'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'
import Icon from '@/components/icons/Icon.vue'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import { adminAPI } from '@/api/admin'
import type { BenefitGrantBatch, BenefitGrantBatchDetail, BenefitGrantMode, BenefitGrantPreviewRequest, BenefitGrantType } from '@/api/admin/benefitGrants'
import type { AdminUser } from '@/types'
import { useAppStore } from '@/stores'
import { isStepUpCancelled, useStepUp } from '@/composables/useStepUp'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const stepUp = useStepUp()
const activeTab = ref<'create' | 'history'>('create')
const grantTypes: BenefitGrantType[] = ['welfare', 'compensation']
const grantModes: BenefitGrantMode[] = ['fixed', 'percentage_24h']
const audiences = ['all', 'selected'] as const
const templateVariables = ['{{amount}}', '{{reason}}', '{{balance}}', '{{site_name}}']
const form = reactive<BenefitGrantPreviewRequest>({
  grant_type: 'welfare', grant_mode: 'fixed', audience_type: 'all', user_ids: [],
  fixed_amount: '', percentage: '10', min_amount: '', per_user_cap: '', total_budget_cap: '',
  reason: '', notification_title: t('admin.benefitGrants.defaults.title'),
  notification_content: t('admin.benefitGrants.defaults.content')
})
const guards = reactive({ min: false, cap: false, budget: false })
const selectedUsers = reactive(new Map<number, AdminUser>())
const selectedUserList = computed(() => [...selectedUsers.values()])
const userSearch = ref('')
const userResults = ref<AdminUser[]>([])
let searchTimer: ReturnType<typeof setTimeout> | undefined
const contentInput = ref<HTMLTextAreaElement | null>(null)
const previewing = ref(false)
const previewBatch = ref<BenefitGrantBatch | null>(null)
const confirmed = ref(false)
const executing = ref(false)
const showNotificationPreview = ref(false)

const batches = ref<BenefitGrantBatch[]>([])
const historyLoading = ref(false)
const historyPage = ref(1)
const historyPageSize = 20
const historyTotal = ref(0)
const historyStatus = ref('')
const detail = ref<BenefitGrantBatchDetail | null>(null)
const detailPage = ref(1)
let pollTimer: ReturnType<typeof setInterval> | undefined

const historyHeadings = computed(() => [t('admin.benefitGrants.columns.batch'), t('admin.benefitGrants.fields.type'), t('admin.benefitGrants.fields.mode'), t('admin.benefitGrants.columns.progress'), t('admin.benefitGrants.columns.amount'), t('common.status'), t('admin.benefitGrants.columns.created'), ''])
const statusOptions = computed(() => [{ value: '', label: t('admin.benefitGrants.allStatuses') }, ...['draft', 'pending', 'processing', 'completed', 'partially_failed', 'failed', 'expired'].map((status) => ({ value: status, label: t(`admin.benefitGrants.statuses.${status}`) }))])
const canPreview = computed(() => form.reason && form.notification_title && form.notification_content && (form.audience_type === 'all' || selectedUsers.size > 0) && (form.grant_mode === 'fixed' ? form.fixed_amount : form.percentage))
const notificationPreview = computed(() => {
  const amount = form.grant_mode === 'fixed' && form.fixed_amount ? form.fixed_amount : '10.00000000'
  const values: Record<string, string> = {
    '{{amount}}': amount,
    '{{reason}}': form.reason || t('admin.benefitGrants.defaults.previewReason'),
    '{{balance}}': '100.00000000',
    '{{site_name}}': appStore.siteName
  }
  const render = (template: string) => Object.entries(values).reduce(
    (result, [key, value]) => result.split(key).join(value),
    template
  )
  return {
    title: render(form.notification_title),
    content: render(form.notification_content),
    created_at: new Date().toISOString()
  }
})

const GuardInput = defineComponent({
  props: { enabled: Boolean, value: { type: String, default: '' }, label: { type: String, required: true }, disabled: Boolean },
  emits: ['update:enabled', 'update:value'],
  setup(props, { emit }) {
    return () => h('label', { class: ['block rounded-lg border p-3', props.disabled ? 'cursor-not-allowed opacity-50' : 'border-gray-200 dark:border-dark-600'] }, [
      h('span', { class: 'flex items-center gap-2 text-sm font-medium text-gray-800 dark:text-gray-200' }, [
        h('input', { type: 'checkbox', checked: props.enabled, disabled: props.disabled, class: 'h-4 w-4 rounded text-primary-600', onChange: (event: Event) => emit('update:enabled', (event.target as HTMLInputElement).checked) }), props.label
      ]),
      props.enabled && !props.disabled ? h('div', { class: 'relative mt-3' }, [h('span', { class: 'absolute left-3 top-1/2 -translate-y-1/2 text-sm text-gray-500' }, '$'), h('input', { value: props.value, inputmode: 'decimal', class: 'input pl-7', placeholder: '0.00000000', onInput: (event: Event) => emit('update:value', (event.target as HTMLInputElement).value) })]) : null
    ])
  }
})
const SummaryRow = defineComponent({ props: { label: String, value: String }, setup: (props) => () => h('div', { class: 'flex items-start justify-between gap-4' }, [h('dt', { class: 'text-gray-500 dark:text-gray-400' }, props.label), h('dd', { class: 'text-right font-medium text-gray-900 dark:text-white' }, props.value)]) })
const MetricBox = defineComponent({ props: { label: String, value: String, emphasis: Boolean }, setup: (props) => () => h('div', { class: ['rounded-lg border p-4', props.emphasis ? 'border-emerald-200 bg-emerald-50 dark:border-emerald-900/50 dark:bg-emerald-900/10' : 'border-gray-200 bg-gray-50 dark:border-dark-600 dark:bg-dark-900/40'] }, [h('p', { class: 'text-xs text-gray-500 dark:text-gray-400' }, props.label), h('p', { class: ['mt-1 text-lg font-semibold', props.emphasis ? 'text-emerald-700 dark:text-emerald-300' : 'text-gray-900 dark:text-white'] }, props.value)]) })

function setGrantMode(mode: BenefitGrantMode) { form.grant_mode = mode; if (mode === 'percentage_24h') form.grant_type = 'compensation' }
function toggleUser(user: AdminUser) {
  if (selectedUsers.has(user.id)) {
    selectedUsers.delete(user.id)
    return
  }
  if (selectedUsers.size >= 500) {
    appStore.showError(t('admin.benefitGrants.errors.selectedLimit'))
    return
  }
  selectedUsers.set(user.id, user)
}
function searchUsers() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(async () => {
    try {
      const result = await adminAPI.users.list(1, 20, { status: 'active', role: 'user', search: userSearch.value.trim() })
      userResults.value = result.items
    } catch (error: any) {
      appStore.showError(error?.message || t('admin.benefitGrants.errors.search'))
    }
  }, 250)
}
function appendVariable(variable: string) { form.notification_content += variable; contentInput.value?.focus() }
function formatAmount(value?: string) { if (!value) return '0'; const number = Number(value); return Number.isFinite(number) ? number.toFixed(8).replace(/\.?0+$/, '') : value }
function buildPayload(): BenefitGrantPreviewRequest { return { ...form, user_ids: form.audience_type === 'selected' ? [...selectedUsers.keys()] : undefined, min_amount: guards.min && form.grant_mode !== 'fixed' ? form.min_amount : undefined, per_user_cap: guards.cap ? form.per_user_cap : undefined, total_budget_cap: guards.budget ? form.total_budget_cap : undefined } }

async function loadSelectedUsers(ids: number[]) {
  form.audience_type = 'selected'
  for (let offset = 0; offset < ids.length; offset += 10) {
    const users = await Promise.all(
      ids.slice(offset, offset + 10).map((id) => adminAPI.users.getById(id).catch(() => null))
    )
    users.forEach((user) => {
      if (user?.status === 'active' && user.role === 'user') selectedUsers.set(user.id, user)
    })
  }
}

async function createPreview() {
  previewing.value = true
  try { previewBatch.value = await adminAPI.benefitGrants.preview(buildPayload()); confirmed.value = false }
  catch (error: any) { appStore.showError(error?.message || t('admin.benefitGrants.errors.preview')) }
  finally { previewing.value = false }
}
function closePreview() { if (!executing.value) { previewBatch.value = null; confirmed.value = false } }
async function executeGrant() {
  if (!previewBatch.value) return
  executing.value = true
  try {
    await stepUp.run(() => adminAPI.benefitGrants.execute(previewBatch.value!.id))
    appStore.showSuccess(t('admin.benefitGrants.submitted'))
    previewBatch.value = null; activeTab.value = 'history'; await loadBatches()
  } catch (error: any) { if (!isStepUpCancelled(error)) appStore.showError(error?.message || t('admin.benefitGrants.errors.execute')) }
  finally { executing.value = false }
}
async function loadBatches() {
  historyLoading.value = true
  try { const result = await adminAPI.benefitGrants.list(historyPage.value, historyPageSize, historyStatus.value); batches.value = result.items; historyTotal.value = result.total }
  catch (error: any) { appStore.showError(error?.message || t('admin.benefitGrants.errors.load')) }
  finally { historyLoading.value = false }
}
function changeHistoryPage(page: number) { historyPage.value = page; void loadBatches() }
async function openDetail(id: number, page = 1) { try { detail.value = await adminAPI.benefitGrants.get(id, page); detailPage.value = page } catch (error: any) { appStore.showError(error?.message || t('admin.benefitGrants.errors.load')) } }
function changeDetailPage(page: number) { if (detail.value) void openDetail(detail.value.batch.id, page) }
function closeDetail() { detail.value = null; detailPage.value = 1 }
async function retryDetail() { if (!detail.value) return; try { await stepUp.run(() => adminAPI.benefitGrants.retryFailed(detail.value!.batch.id)); appStore.showSuccess(t('admin.benefitGrants.retrySubmitted')); await openDetail(detail.value.batch.id, detailPage.value); await loadBatches() } catch (error: any) { if (!isStepUpCancelled(error)) appStore.showError(error?.message || t('admin.benefitGrants.errors.retry')) } }
async function exportDetail() { if (!detail.value) return; try { const blob = await adminAPI.benefitGrants.exportItems(detail.value.batch.id); saveAs(blob, `benefit_grant_${detail.value.batch.id}.csv`) } catch (error: any) { appStore.showError(error?.message || t('admin.benefitGrants.errors.export')) } }

onMounted(async () => {
  const ids = String(route.query.users || '').split(',').map(Number).filter((id) => id > 0).slice(0, 500)
  if (ids.length) await loadSelectedUsers(ids)
  pollTimer = setInterval(() => {
    if (activeTab.value !== 'history') return
    if (batches.value.some((batch) => ['pending', 'processing'].includes(batch.status))) void loadBatches()
    if (detail.value && ['pending', 'processing'].includes(detail.value.batch.status)) void openDetail(detail.value.batch.id, detailPage.value)
  }, 5000)
})
onBeforeUnmount(() => { clearTimeout(searchTimer); clearInterval(pollTimer) })
</script>
