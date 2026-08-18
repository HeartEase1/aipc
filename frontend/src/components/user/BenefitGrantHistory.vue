<template>
  <section data-testid="benefit-grant-history" class="card overflow-hidden">
    <header
      class="flex items-start justify-between gap-4 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:px-6"
    >
      <div>
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('benefits.historyTitle') }}
        </h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('benefits.historyDescription') }}
        </p>
      </div>
      <button
        type="button"
        class="btn btn-ghost btn-sm shrink-0"
        :disabled="isLoading"
        :title="t('common.refresh')"
        @click="refreshAll"
      >
        <Icon name="refresh" size="sm" :class="isLoading ? 'animate-spin' : ''" />
        <span class="sr-only">{{ t('common.refresh') }}</span>
      </button>
    </header>

    <div v-if="isLoading" class="flex min-h-48 items-center justify-center">
      <LoadingSpinner />
    </div>

    <template v-else>
      <div
        v-if="loadFailed"
        class="flex items-center justify-between gap-4 border-b border-red-100 bg-red-50 px-5 py-3 dark:border-red-900/40 dark:bg-red-900/10 sm:px-6"
      >
        <p class="text-sm text-red-700 dark:text-red-300">
          {{ t('benefits.loadFailed') }}
        </p>
        <button type="button" class="btn btn-secondary btn-sm shrink-0" @click="load">
          {{ t('common.refresh') }}
        </button>
      </div>

      <div v-if="activities.length > 0" class="divide-y divide-gray-100 dark:divide-dark-700">
        <article
          v-for="activity in activities"
          :key="activity.key"
          data-testid="account-activity-item"
          :data-activity-kind="activity.benefit ? activity.benefit.grant_type : 'standard'"
          class="px-5 py-5 transition-colors sm:px-6"
          :class="
            activity.benefit
              ? benefitActivityClass(activity.benefit.grant_type)
              : 'bg-white hover:bg-gray-50/70 dark:bg-dark-800 dark:hover:bg-dark-700/30'
          "
        >
          <template v-if="activity.benefit">
            <div class="grid gap-4 sm:grid-cols-[minmax(0,1fr)_auto]">
              <div class="flex min-w-0 items-start gap-4">
                <div
                  class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl ring-1 ring-inset"
                  :class="
                    activity.benefit.grant_type === 'welfare'
                      ? 'bg-emerald-100 text-emerald-700 ring-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-300 dark:ring-emerald-800'
                      : 'bg-amber-100 text-amber-700 ring-amber-200 dark:bg-amber-900/30 dark:text-amber-300 dark:ring-amber-800'
                  "
                >
                  <Icon name="gift" size="md" />
                </div>
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <span
                      class="rounded-md px-2 py-1 text-xs font-medium"
                      :class="
                        activity.benefit.grant_type === 'welfare'
                          ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
                          : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
                      "
                    >
                      {{ t(`benefits.types.${activity.benefit.grant_type}`) }}
                    </span>
                    <span
                      v-if="!activity.benefit.read_at"
                      class="h-2 w-2 rounded-full bg-primary-500"
                      :title="t('benefits.unread')"
                    ></span>
                    <time class="text-xs text-gray-500 dark:text-gray-400">
                      {{ formatDateTime(activity.benefit.created_at) }}
                    </time>
                  </div>
                  <h3 class="mt-2 font-medium text-gray-900 dark:text-white">
                    {{ activity.benefit.title }}
                  </h3>
                  <p class="mt-1 break-words text-sm text-gray-600 dark:text-gray-400">
                    {{ activity.benefit.reason }}
                  </p>
                </div>
              </div>
              <div class="pl-14 sm:pl-0 sm:text-right">
                <p class="text-xl font-semibold text-emerald-600 dark:text-emerald-400">
                  +${{ formatGrantAmount(activity.benefit.amount) }}
                </p>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{
                    t('benefits.balanceAfter', {
                      amount: formatGrantAmount(activity.benefit.balance_after),
                    })
                  }}
                </p>
              </div>
            </div>

            <BenefitGrantCalculationDetails :grant="activity.benefit" class="mt-4 sm:ml-14" />
          </template>

          <template v-else-if="activity.redeem">
            <div class="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
              <div class="flex min-w-0 items-center gap-4">
                <div
                  :class="[
                    'flex h-10 w-10 shrink-0 items-center justify-center rounded-xl',
                    isBalanceType(activity.redeem.type)
                      ? activity.redeem.value >= 0
                        ? 'bg-emerald-100 dark:bg-emerald-900/30'
                        : 'bg-red-100 dark:bg-red-900/30'
                      : isSubscriptionType(activity.redeem.type)
                        ? 'bg-purple-100 dark:bg-purple-900/30'
                        : activity.redeem.value >= 0
                          ? 'bg-blue-100 dark:bg-blue-900/30'
                          : 'bg-orange-100 dark:bg-orange-900/30',
                  ]"
                >
                  <Icon
                    v-if="isBalanceType(activity.redeem.type)"
                    name="dollar"
                    size="md"
                    :class="
                      activity.redeem.value >= 0
                        ? 'text-emerald-600 dark:text-emerald-400'
                        : 'text-red-600 dark:text-red-400'
                    "
                  />
                  <Icon
                    v-else-if="isSubscriptionType(activity.redeem.type)"
                    name="badge"
                    size="md"
                    class="text-purple-600 dark:text-purple-400"
                  />
                  <Icon
                    v-else
                    name="bolt"
                    size="md"
                    :class="
                      activity.redeem.value >= 0
                        ? 'text-blue-600 dark:text-blue-400'
                        : 'text-orange-600 dark:text-orange-400'
                    "
                  />
                </div>
                <div class="min-w-0">
                  <p class="text-sm font-medium text-gray-900 dark:text-white">
                    {{ getHistoryItemTitle(activity.redeem) }}
                  </p>
                  <p class="text-xs text-gray-500 dark:text-dark-400">
                    {{ formatDateTime(activity.redeem.used_at) }}
                  </p>
                </div>
              </div>

              <div class="pl-14 sm:pl-0 sm:text-right">
                <p :class="historyValueClass(activity.redeem)">
                  {{ formatHistoryValue(activity.redeem) }}
                </p>
                <p
                  v-if="!isAdminAdjustment(activity.redeem.type)"
                  class="font-mono text-xs text-gray-400 dark:text-dark-500"
                >
                  {{ activity.redeem.code.slice(0, 8) }}...
                </p>
                <p v-else class="text-xs text-gray-400 dark:text-dark-500">
                  {{ t('redeem.adminAdjustment') }}
                </p>
                <p
                  v-if="activity.redeem.notes"
                  class="mt-1 break-words text-xs italic text-gray-500 dark:text-dark-400 sm:max-w-[240px]"
                >
                  {{ activity.redeem.notes }}
                </p>
              </div>
            </div>
          </template>
        </article>
      </div>

      <div v-else-if="!loadFailed" class="px-6 py-12 text-center">
        <div
          class="mx-auto mb-3 flex h-11 w-11 items-center justify-center rounded-lg bg-gray-100 text-gray-400 dark:bg-dark-700"
        >
          <Icon name="clock" size="lg" />
        </div>
        <p class="font-medium text-gray-800 dark:text-gray-200">{{ t('benefits.empty') }}</p>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('benefits.emptyHint') }}
        </p>
      </div>
    </template>

    <Pagination
      v-if="!isLoading && !loadFailed && total > pageSize"
      :total="total"
      :page="page"
      :page-size="pageSize"
      :show-page-size-selector="false"
      @update:page="changePage"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import benefitGrantsAPI, { type UserBenefitGrant } from '@/api/benefitGrants'
import type { RedeemHistoryItem } from '@/api'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Pagination from '@/components/common/Pagination.vue'
import BenefitGrantCalculationDetails from '@/components/user/BenefitGrantCalculationDetails.vue'
import { formatDateTime } from '@/utils/format'

const props = withDefaults(
  defineProps<{
    redeemHistory?: RedeemHistoryItem[]
    redeemLoading?: boolean
  }>(),
  {
    redeemHistory: () => [],
    redeemLoading: false,
  },
)

const emit = defineEmits<{
  refreshRedeemHistory: []
}>()

const { t } = useI18n()
const loading = ref(false)
const loadFailed = ref(false)
const items = ref<UserBenefitGrant[]>([])
const page = ref(1)
const pageSize = 20
const total = ref(0)

interface ActivityEntry {
  key: string
  timestamp: number
  benefit?: UserBenefitGrant
  redeem?: RedeemHistoryItem
}

const isLoading = computed(() => loading.value || props.redeemLoading)
const activities = computed<ActivityEntry[]>(() => {
  const entries: ActivityEntry[] = items.value.map((benefit) => ({
    key: `benefit-${benefit.id}`,
    timestamp: Date.parse(benefit.created_at),
    benefit,
  }))

  if (page.value === 1) {
    entries.push(
      ...props.redeemHistory.map((redeem) => ({
        key: `redeem-${redeem.id}`,
        timestamp: Date.parse(redeem.used_at),
        redeem,
      })),
    )
  }

  return entries.sort((left, right) => right.timestamp - left.timestamp)
})

function formatGrantAmount(value: string): string {
  return value.replace(/(\.\d*?[1-9])0+$|\.0+$/, '$1')
}

function benefitActivityClass(grantType: UserBenefitGrant['grant_type']): string[] {
  return grantType === 'welfare'
    ? [
        'border-l-4 border-l-emerald-400 bg-emerald-50/45 hover:bg-emerald-50/70',
        'dark:border-l-emerald-500 dark:bg-emerald-950/10 dark:hover:bg-emerald-950/20',
      ]
    : [
        'border-l-4 border-l-amber-400 bg-amber-50/45 hover:bg-amber-50/70',
        'dark:border-l-amber-500 dark:bg-amber-950/10 dark:hover:bg-amber-950/20',
      ]
}

function isBalanceType(type: string): boolean {
  return type === 'balance' || type === 'admin_balance'
}

function isSubscriptionType(type: string): boolean {
  return type === 'subscription'
}

function isAdminAdjustment(type: string): boolean {
  return type === 'admin_balance' || type === 'admin_concurrency'
}

function getHistoryItemTitle(item: RedeemHistoryItem): string {
  if (item.type === 'balance') return t('redeem.balanceAddedRedeem')
  if (item.type === 'admin_balance') {
    return item.value >= 0 ? t('redeem.balanceAddedAdmin') : t('redeem.balanceDeductedAdmin')
  }
  if (item.type === 'concurrency') return t('redeem.concurrencyAddedRedeem')
  if (item.type === 'admin_concurrency') {
    return item.value >= 0
      ? t('redeem.concurrencyAddedAdmin')
      : t('redeem.concurrencyReducedAdmin')
  }
  if (item.type === 'subscription') return t('redeem.subscriptionAssigned')
  return t('common.unknown')
}

function formatHistoryValue(item: RedeemHistoryItem): string {
  if (isBalanceType(item.type)) {
    return `${item.value >= 0 ? '+' : ''}$${item.value.toFixed(2)}`
  }
  if (isSubscriptionType(item.type)) {
    const days = item.validity_days || Math.round(item.value)
    const groupName = item.group?.name || ''
    return groupName ? `${days}${t('redeem.days')} - ${groupName}` : `${days}${t('redeem.days')}`
  }
  return `${item.value >= 0 ? '+' : ''}${item.value} ${t('redeem.requests')}`
}

function historyValueClass(item: RedeemHistoryItem): string[] {
  const color = isBalanceType(item.type)
    ? item.value >= 0
      ? 'text-emerald-600 dark:text-emerald-400'
      : 'text-red-600 dark:text-red-400'
    : isSubscriptionType(item.type)
      ? 'text-purple-600 dark:text-purple-400'
      : item.value >= 0
        ? 'text-blue-600 dark:text-blue-400'
        : 'text-orange-600 dark:text-orange-400'
  return ['text-sm font-semibold', color]
}

async function load() {
  loading.value = true
  loadFailed.value = false
  try {
    const result = await benefitGrantsAPI.list(page.value, pageSize)
    items.value = result.items
    total.value = result.total
  } catch (error) {
    loadFailed.value = true
    console.error('Failed to load benefit grant history:', error)
  } finally {
    loading.value = false
  }
}

function refreshAll() {
  emit('refreshRedeemHistory')
  void load()
}

function changePage(next: number) {
  page.value = next
  void load()
}

onMounted(() => {
  void load()
})
</script>
