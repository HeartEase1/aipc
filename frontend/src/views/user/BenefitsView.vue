<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <div class="mb-2 flex items-center gap-3">
            <span class="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
              <Icon name="gift" size="lg" />
            </span>
            <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('benefits.title') }}</h1>
          </div>
          <p class="text-sm text-gray-600 dark:text-gray-400">{{ t('benefits.description') }}</p>
        </div>
        <button class="btn btn-secondary" :disabled="loading" @click="load">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          <span class="ml-2">{{ t('common.refresh') }}</span>
        </button>
      </header>

      <section class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div v-if="loading" class="flex min-h-64 items-center justify-center">
          <LoadingSpinner />
        </div>
        <div v-else-if="items.length === 0" class="px-6 py-16 text-center">
          <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-gray-100 text-gray-400 dark:bg-dark-700">
            <Icon name="inbox" size="lg" />
          </div>
          <p class="font-medium text-gray-800 dark:text-gray-200">{{ t('benefits.empty') }}</p>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('benefits.emptyHint') }}</p>
        </div>
        <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
          <article v-for="item in items" :key="item.id" class="grid gap-4 px-5 py-5 sm:grid-cols-[1fr_auto] sm:px-6">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <span
                  class="rounded-md px-2 py-1 text-xs font-medium"
                  :class="item.grant_type === 'welfare'
                    ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
                    : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'"
                >
                  {{ t(`benefits.types.${item.grant_type}`) }}
                </span>
                <span v-if="!item.read_at" class="h-2 w-2 rounded-full bg-primary-500" :title="t('benefits.unread')"></span>
                <time class="text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(item.created_at) }}</time>
              </div>
              <h2 class="mt-2 font-medium text-gray-900 dark:text-white">{{ item.title }}</h2>
              <p class="mt-1 break-words text-sm text-gray-600 dark:text-gray-400">{{ item.reason }}</p>
            </div>
            <div class="sm:text-right">
              <p class="text-xl font-semibold text-emerald-600 dark:text-emerald-400">+${{ formatAmount(item.amount) }}</p>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('benefits.balanceAfter', { amount: formatAmount(item.balance_after) }) }}
              </p>
            </div>
          </article>
        </div>
      </section>

      <Pagination
        v-if="total > pageSize"
        :total="total"
        :page="page"
        :page-size="pageSize"
        :show-page-size-selector="false"
        @update:page="changePage"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Pagination from '@/components/common/Pagination.vue'
import benefitGrantsAPI, { type UserBenefitGrant } from '@/api/benefitGrants'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const loading = ref(false)
const items = ref<UserBenefitGrant[]>([])
const page = ref(1)
const pageSize = 20
const total = ref(0)

function formatAmount(value: string) {
  const number = Number(value)
  return Number.isFinite(number) ? number.toFixed(8).replace(/\.?0+$/, '') : value
}

async function load() {
  loading.value = true
  try {
    const result = await benefitGrantsAPI.list(page.value, pageSize)
    items.value = result.items
    total.value = result.total
  } finally {
    loading.value = false
  }
}

function changePage(next: number) {
  page.value = next
  void load()
}

onMounted(load)
</script>
