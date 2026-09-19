import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { describe, expect, it } from 'vitest'
import GroupModelCompatibilityFields from '@/components/admin/GroupModelCompatibilityFields.vue'
import messages from '@/i18n/locales/en/admin/groupModelCompatibility'

describe('GroupModelCompatibilityFields', () => {
  it('edits denied models without losing line separators or affecting listing mode', async () => {
    const wrapper = mount(GroupModelCompatibilityFields, {
      props: { blockedModels: ['gpt-image-*'], legacyListOnly: true },
      global: { plugins: [createI18n({ legacy: false, locale: 'en', messages: { en: { admin: { groupModelCompatibility: Object.fromEntries(Object.entries(messages).map(([key, value]) => [key, () => value])) } } } })] },
    })
    const input = wrapper.get('textarea')
    expect(wrapper.text()).toContain('Model listing only')
    expect(wrapper.text()).toContain('Denied models')
    await input.setValue('gpt-image-*\nclaude-*\n')
    expect(wrapper.emitted('update:blockedModels')?.at(-1)).toEqual([['gpt-image-*', 'claude-*']])
    await wrapper.setProps({ blockedModels: ['gpt-image-*', 'claude-*'] })
    expect((input.element as HTMLTextAreaElement).value).toBe('gpt-image-*\nclaude-*\n')
    await wrapper.get('[role="switch"]').trigger('click')
    expect(wrapper.emitted('update:legacyListOnly')?.at(-1)).toEqual([false])
    await input.setValue('')
    expect(wrapper.emitted('update:blockedModels')?.at(-1)).toEqual([[]])
  })
})
