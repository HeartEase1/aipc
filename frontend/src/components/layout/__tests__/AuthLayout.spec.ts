import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AuthLayout from '../AuthLayout.vue'

const fetchPublicSettings = vi.fn()

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    siteName: 'Example API',
    siteLogo: '/brand.svg',
    cachedPublicSettings: {
      site_subtitle: 'Reliable access to supported models',
    },
    publicSettingsLoaded: true,
    fetchPublicSettings,
  }),
}))

describe('AuthLayout', () => {
  beforeEach(() => {
    fetchPublicSettings.mockClear()
  })

  it('keeps auth content and footer slots inside the responsive two-surface layout', () => {
    const wrapper = mount(AuthLayout, {
      slots: {
        default: '<form data-testid="auth-form">Form</form>',
        footer: '<a data-testid="auth-footer">Create account</a>',
      },
    })

    expect(wrapper.find('.auth-brand-panel').exists()).toBe(true)
    expect(wrapper.find('.auth-form-panel').exists()).toBe(true)
    expect(wrapper.find('.auth-mobile-brand').exists()).toBe(true)
    expect(wrapper.find('.auth-card [data-testid="auth-form"]').exists()).toBe(true)
    expect(wrapper.find('.auth-footer [data-testid="auth-footer"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Example API')
    expect(wrapper.text()).toContain('Reliable access to supported models')
    expect(fetchPublicSettings).toHaveBeenCalledOnce()
  })
})
