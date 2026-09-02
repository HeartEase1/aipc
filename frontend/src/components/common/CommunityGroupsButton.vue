<template>
  <div>
    <button
      type="button"
      class="group inline-flex h-9 items-center justify-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-gray-600 transition-all duration-200 hover:scale-105 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/35 focus-visible:ring-offset-2 dark:text-gray-400 dark:hover:bg-dark-800 dark:hover:text-white dark:focus-visible:ring-offset-dark-900"
      :class="dialogOpen && 'bg-gray-100 text-gray-900 dark:bg-dark-800 dark:text-white'"
      :aria-label="t('communityGroups.buttonLabel')"
      :aria-expanded="dialogOpen"
      :title="t('communityGroups.buttonLabel')"
      data-testid="community-groups-button"
      @click="openDialog"
    >
      <Icon name="chatBubble" size="sm" class="shrink-0 transition-transform duration-200 group-hover:scale-110" />
      <span class="hidden whitespace-nowrap xl:inline">{{ t('communityGroups.button') }}</span>
    </button>

    <BaseDialog
      :show="dialogOpen"
      :title="t('communityGroups.title')"
      width="medium-wide"
      :close-on-click-outside="true"
      :z-index="120"
      @close="dialogOpen = false"
    >
      <div class="community-dialog">
        <div class="community-intro">
          <div class="community-intro__icon">
            <Icon name="chatBubble" size="lg" />
          </div>
          <div class="min-w-0">
            <p class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('communityGroups.title') }}
            </p>
            <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">
              {{ t('communityGroups.subtitle') }}
            </p>
          </div>
        </div>

        <div v-if="loading" class="flex min-h-56 flex-col items-center justify-center gap-3 text-sm text-gray-500 dark:text-gray-400">
          <span class="h-8 w-8 animate-spin rounded-full border-2 border-primary-200 border-t-primary-600 dark:border-primary-900 dark:border-t-primary-400"></span>
          <span>{{ t('communityGroups.loading') }}</span>
        </div>

        <div v-else-if="loadError" class="flex min-h-56 flex-col items-center justify-center gap-3 text-center">
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-red-50 text-red-500 dark:bg-red-900/20 dark:text-red-400">
            <Icon name="exclamationCircle" size="lg" />
          </div>
          <p class="text-sm font-medium text-gray-800 dark:text-gray-200">{{ t('communityGroups.loadFailed') }}</p>
          <button type="button" class="btn btn-secondary btn-sm" @click="loadGroups">
            <Icon name="refresh" size="sm" />
            {{ t('communityGroups.retry') }}
          </button>
        </div>

        <div v-else-if="displayGroups.length === 0" class="flex min-h-56 flex-col items-center justify-center gap-3 text-center">
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-gray-100 text-gray-400 dark:bg-dark-700 dark:text-gray-500">
            <Icon name="chatBubble" size="lg" />
          </div>
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('communityGroups.empty') }}</p>
        </div>

        <div v-else class="grid grid-cols-1 gap-4 py-1 lg:grid-cols-2">
          <article
            v-for="group in displayGroups"
            :key="`${group.name}:${group.group_number}`"
            class="community-card"
            data-testid="community-group-card"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <h4 class="truncate text-base font-semibold text-gray-900 dark:text-white">{{ group.name }}</h4>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('communityGroups.groupNumber') }}</p>
              </div>
              <span class="inline-flex shrink-0 items-center gap-1.5 rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-900/25 dark:text-emerald-300">
                <span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
                {{ t('communityGroups.available') }}
              </span>
            </div>

            <button
              type="button"
              class="mt-3 flex w-full items-center justify-between gap-3 rounded-xl border border-gray-200 bg-white/80 px-3.5 py-3 text-left transition-colors hover:border-primary-300 hover:bg-primary-50/60 dark:border-dark-600 dark:bg-dark-800/70 dark:hover:border-primary-700 dark:hover:bg-primary-900/15"
              :title="t('communityGroups.copyNumber')"
              data-testid="community-group-copy"
              @click="copyGroupNumber(group.group_number)"
            >
              <span class="min-w-0 truncate font-mono text-sm font-semibold text-gray-800 dark:text-gray-200">{{ group.group_number }}</span>
              <Icon name="copy" size="sm" class="shrink-0 text-gray-400" />
            </button>

            <div v-if="group.safeQRCode" class="mt-4 flex items-center gap-4 rounded-xl bg-white/75 p-3 ring-1 ring-gray-200/80 dark:bg-dark-800/65 dark:ring-dark-600">
              <img
                :src="group.safeQRCode"
                :alt="`${group.name} ${t('communityGroups.scanTitle')}`"
                class="h-24 w-24 shrink-0 rounded-lg bg-white object-contain p-1 shadow-sm sm:h-28 sm:w-28"
              />
              <div class="min-w-0">
                <p class="text-sm font-semibold text-gray-800 dark:text-gray-200">{{ t('communityGroups.scanTitle') }}</p>
                <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('communityGroups.scanHint') }}</p>
              </div>
            </div>

            <a
              v-if="group.safeJoinURL"
              :href="group.safeJoinURL"
              target="_blank"
              rel="noopener noreferrer"
              class="mt-4 inline-flex h-10 w-full items-center justify-center gap-2 rounded-xl bg-primary-600 px-4 text-sm font-semibold text-white shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:bg-primary-700 hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 focus-visible:ring-offset-2 dark:focus-visible:ring-offset-dark-900"
              data-testid="community-group-join"
            >
              <Icon name="externalLink" size="sm" />
              {{ t('communityGroups.join') }}
            </a>
          </article>
        </div>

        <p v-if="!loading && !loadError && displayGroups.length > 0" class="community-footer">
          {{ t('communityGroups.footer') }}
        </p>
      </div>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { communityGroupsAPI } from '@/api'
import type { CommunityGroup } from '@/types'
import { useClipboard } from '@/composables/useClipboard'
import { sanitizeUrl } from '@/utils/url'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

type DisplayCommunityGroup = CommunityGroup & {
  safeJoinURL: string
  safeQRCode: string
}

const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const dialogOpen = ref(false)
const loading = ref(false)
const loadError = ref(false)
const groups = ref<CommunityGroup[]>([])

const displayGroups = computed<DisplayCommunityGroup[]>(() =>
  groups.value.map((group) => ({
    ...group,
    safeJoinURL: sanitizeUrl(group.join_url || ''),
    safeQRCode: sanitizeUrl(group.qr_code_image || '', { allowDataUrl: true })
  }))
)

async function openDialog() {
  dialogOpen.value = true
  await loadGroups()
}

async function loadGroups() {
  loading.value = true
  loadError.value = false
  try {
    groups.value = await communityGroupsAPI.getCommunityGroups()
  } catch {
    groups.value = []
    loadError.value = true
  } finally {
    loading.value = false
  }
}

async function copyGroupNumber(groupNumber: string) {
  await copyToClipboard(groupNumber, t('communityGroups.copied'))
}
</script>

<style scoped>
.community-dialog {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.community-intro {
  display: flex;
  align-items: center;
  gap: 0.875rem;
  padding: 1rem;
  border: 1px solid rgb(var(--color-primary-200) / 0.7);
  border-radius: 0.875rem;
  background: linear-gradient(135deg, rgb(var(--color-primary-50) / 0.92), rgb(255 255 255 / 0.76));
}

.community-intro__icon {
  display: flex;
  width: 3rem;
  height: 3rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 0.875rem;
  color: rgb(var(--color-primary-600));
  background: rgb(var(--color-primary-100) / 0.88);
  box-shadow: 0 8px 20px rgb(var(--color-primary-500) / 0.12);
}

.community-card {
  min-width: 0;
  padding: 1rem;
  border: 1px solid rgb(226 232 240);
  border-radius: 1rem;
  background: linear-gradient(145deg, rgb(248 250 252 / 0.98), rgb(255 255 255 / 0.98));
  box-shadow: 0 10px 25px rgb(15 23 42 / 0.06);
}

.community-footer {
  padding-top: 0.875rem;
  border-top: 1px solid rgb(226 232 240);
  color: rgb(100 116 139);
  font-size: 0.8125rem;
  line-height: 1.5rem;
}

:global(.dark .community-intro) {
  border-color: rgb(var(--color-primary-800) / 0.7);
  background: linear-gradient(135deg, rgb(var(--color-primary-950) / 0.42), rgb(17 24 39 / 0.78));
}

:global(.dark .community-intro__icon) {
  color: rgb(var(--color-primary-300));
  background: rgb(var(--color-primary-900) / 0.52);
}

:global(.dark .community-card) {
  border-color: rgb(55 65 81);
  background: linear-gradient(145deg, rgb(31 41 55 / 0.94), rgb(17 24 39 / 0.96));
  box-shadow: 0 12px 28px rgb(0 0 0 / 0.2);
}

:global(.dark .community-footer) {
  border-color: rgb(55 65 81);
  color: rgb(156 163 175);
}

@media (prefers-reduced-motion: reduce) {
  .community-dialog *,
  .community-dialog *::before,
  .community-dialog *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
