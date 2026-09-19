import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const { getCommunityGroups, copyToClipboard } = vi.hoisted(() => ({
  getCommunityGroups: vi.fn(),
  copyToClipboard: vi.fn().mockResolvedValue(true)
}))

vi.mock('@/api', () => ({
  communityGroupsAPI: { getCommunityGroups }
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard })
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

import CommunityGroupsButton from '../CommunityGroupsButton.vue'

describe('CommunityGroupsButton', () => {
  beforeEach(() => {
    getCommunityGroups.mockReset()
    copyToClipboard.mockClear()
  })

  afterEach(() => {
    document.body.innerHTML = ''
    document.body.classList.remove('modal-open')
  })

  it('loads and renders multiple communities only after the button is opened', async () => {
    getCommunityGroups.mockResolvedValue([
      {
        name: 'QQ Group',
        group_number: '123456',
        qr_code_image: 'data:image/png;base64,iVBORw0KGgo=',
        join_url: 'https://example.com/join'
      },
      {
        name: 'Backup Group',
        group_number: '654321',
        qr_code_image: '',
        join_url: ''
      }
    ])
    const wrapper = mount(CommunityGroupsButton)

    expect(getCommunityGroups).not.toHaveBeenCalled()
    expect(wrapper.get('[data-testid="community-groups-button"]').attributes('aria-expanded')).toBe('false')
    await wrapper.get('[data-testid="community-groups-button"]').trigger('click')
    await flushPromises()

    expect(getCommunityGroups).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="community-groups-button"]').attributes('aria-expanded')).toBe('true')
    expect(document.body.querySelectorAll('[data-testid="community-group-card"]')).toHaveLength(2)
    expect(document.body.textContent).toContain('QQ Group')
    expect(document.body.textContent).toContain('654321')
    const link = document.body.querySelector<HTMLAnchorElement>('[data-testid="community-group-join"]')
    expect(link?.href).toBe('https://example.com/join')

    document.body.querySelector<HTMLButtonElement>('[data-testid="community-group-copy"]')?.click()
    await flushPromises()
    expect(copyToClipboard).toHaveBeenCalledWith('123456', 'communityGroups.copied')
    wrapper.unmount()
  })

  it('does not render an unsafe join link returned by a compromised response', async () => {
    getCommunityGroups.mockResolvedValue([{
      name: 'Unsafe Group',
      group_number: '123',
      qr_code_image: '',
      join_url: 'javascript:alert(1)'
    }])
    const wrapper = mount(CommunityGroupsButton)

    await wrapper.get('[data-testid="community-groups-button"]').trigger('click')
    await flushPromises()

    expect(document.body.querySelector('[data-testid="community-group-join"]')).toBeNull()
    wrapper.unmount()
  })
})
