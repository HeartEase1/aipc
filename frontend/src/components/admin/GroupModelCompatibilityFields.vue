<template>
  <div class="mb-4 space-y-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
    <div class="flex items-center justify-between gap-3">
      <div class="min-w-0">
        <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.groupModelCompatibility.listingOnly') }}</p>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.groupModelCompatibility.listingHint') }}</p>
      </div>
      <Toggle :model-value="legacyListOnly" :aria-label="t('admin.groupModelCompatibility.listingOnly')" @update:model-value="emit('update:legacyListOnly', $event)" />
    </div>
    <label class="block text-sm text-gray-700 dark:text-gray-300">
      {{ t('admin.groupModelCompatibility.blocked') }}
      <textarea
        :value="draft"
        rows="3"
        class="input mt-2 w-full resize-y"
        :placeholder="t('admin.groupModelCompatibility.placeholder')"
        @input="updateBlocked(($event.target as HTMLTextAreaElement).value)"
      />
    </label>
    <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.groupModelCompatibility.blockedHint') }}</p>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'

const props = defineProps<{ blockedModels: string[]; legacyListOnly: boolean }>()
const emit = defineEmits<{
  (event: 'update:blockedModels', value: string[]): void
  (event: 'update:legacyListOnly', value: boolean): void
}>()
const draft = ref(props.blockedModels.join('\n'))
const parse = (value: string) => [...new Set(value.split(/[\s,，]+/).map(item => item.trim()).filter(Boolean))]
watch(() => props.blockedModels, value => {
  if (JSON.stringify(parse(draft.value)) !== JSON.stringify(value)) draft.value = value.join('\n')
})
function updateBlocked(value: string) {
  draft.value = value
  emit('update:blockedModels', parse(value))
}
const { t } = useI18n()
</script>
