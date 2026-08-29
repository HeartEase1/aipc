<template>
  <section class="border-t pt-4">
    <div class="mb-3 flex flex-wrap items-start justify-between gap-3">
      <div class="min-w-0">
        <div class="flex flex-wrap items-center gap-2">
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t("admin.groups.blockedModels.title") }}
          </label>
          <span
            class="rounded-full bg-red-50 px-2 py-0.5 text-xs font-medium text-red-600 dark:bg-red-950/40 dark:text-red-300"
          >
            {{ t("admin.groups.blockedModels.count", { count: blockedModels.length }) }}
          </span>
        </div>
        <p class="mt-1 max-w-2xl text-xs leading-5 text-gray-500 dark:text-gray-400">
          {{ t("admin.groups.blockedModels.hint") }}
        </p>
      </div>
      <button
        v-if="blockedModels.length > 0"
        type="button"
        class="rounded-md px-2.5 py-1.5 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-white"
        @click="emit('clear')"
      >
        {{ t("admin.groups.blockedModels.clear") }}
      </button>
    </div>

    <div class="overflow-hidden rounded-lg border border-gray-200 bg-gray-50/60 dark:border-dark-600 dark:bg-dark-800/40">
      <div class="border-b border-gray-200 bg-white p-2.5 dark:border-dark-600 dark:bg-dark-800">
        <input
          v-model="search"
          type="search"
          class="input h-9"
          :placeholder="t('admin.groups.blockedModels.searchPlaceholder')"
        />
      </div>
      <div class="max-h-64 overflow-y-auto p-2">
        <p v-if="loading" class="px-1 py-2 text-xs text-gray-500 dark:text-gray-400">
          {{ t("admin.groups.modelsList.loading") }}
        </p>
        <p
          v-else-if="filteredModels.length === 0"
          class="px-1 py-2 text-xs text-gray-500 dark:text-gray-400"
        >
          {{ search ? t("admin.groups.blockedModels.noResults") : t("admin.groups.blockedModels.empty") }}
        </p>
        <label
          v-for="model in filteredModels"
          :key="model"
          class="flex cursor-pointer items-center gap-3 rounded-md px-2.5 py-2 transition-colors hover:bg-white dark:hover:bg-dark-700"
          :class="isBlocked(model) ? 'bg-red-50/70 dark:bg-red-950/20' : ''"
        >
          <input
            type="checkbox"
            class="h-4 w-4 rounded border-gray-300 text-red-600 focus:ring-red-500"
            :checked="isBlocked(model)"
            @change="emit('toggle-model', model)"
          />
          <span class="min-w-0 flex-1 break-all text-sm text-gray-700 dark:text-gray-300">
            {{ model }}
          </span>
          <span
            v-if="isBlocked(model)"
            class="text-xs font-medium text-red-600 dark:text-red-300"
          >
            {{ t("admin.groups.blockedModels.blocked") }}
          </span>
        </label>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from "vue"
import { useI18n } from "vue-i18n"

const props = defineProps<{
  models: string[]
  blockedModels: string[]
  loading?: boolean
}>()

const emit = defineEmits<{
  (event: "toggle-model", model: string): void
  (event: "clear"): void
}>()

const { t } = useI18n()
const search = ref("")
const blockedSet = computed(() => new Set(props.blockedModels))
const filteredModels = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  if (!query) {
    return props.models
  }
  return props.models.filter(model => model.toLocaleLowerCase().includes(query))
})

const isBlocked = (model: string) => blockedSet.value.has(model)
</script>
