import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import TotpStepUpDialog from '../TotpStepUpDialog.vue'
import { useStepUp } from '@/composables/useStepUp'

const { stepUp, showError } = vi.hoisted(() => ({
  stepUp: vi.fn(),
  showError: vi.fn()
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('@/api', () => ({
  totpAPI: { stepUp }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ showError })
}))

describe('TotpStepUpDialog', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    stepUp.mockReset()
    showError.mockReset()
  })

  it('teleports above nested dialogs and keeps the code inputs interactive', async () => {
    const host = document.createElement('div')
    host.style.position = 'relative'
    host.style.zIndex = '1'
    document.body.appendChild(host)
    const controller = useStepUp()
    const wrapper = mount(TotpStepUpDialog, {
      attachTo: host,
      props: { controller }
    })

    controller.visible.value = true
    await nextTick()
    await flushPromises()

    const overlay = document.body.querySelector<HTMLElement>('[data-testid="totp-step-up-dialog"]')
    expect(overlay).not.toBeNull()
    expect(overlay?.parentElement).toBe(document.body)
    expect(overlay?.classList.contains('z-[160]')).toBe(true)

    const inputs = Array.from(document.body.querySelectorAll<HTMLInputElement>('.totp-step-up-code-input'))
    expect(inputs).toHaveLength(6)
    expect(document.activeElement).toBe(inputs[0])

    inputs[0].value = '1'
    inputs[0].dispatchEvent(new Event('input', { bubbles: true }))
    await nextTick()
    expect(document.activeElement).toBe(inputs[1])

    wrapper.unmount()
  })
})
