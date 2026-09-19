import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import TotpStepUpDialog from '../TotpStepUpDialog.vue'

const { verify } = vi.hoisted(() => ({ verify: vi.fn().mockResolvedValue({}) }))
vi.mock('@/api', () => ({ totpAPI: { stepUp: verify } }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showError: vi.fn() }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('step-up over a console dialog', () => {
  it('teleports outside the clipping parent and verifies before resuming', async () => {
    const parent = document.createElement('div')
    parent.style.overflow = 'hidden'
    document.body.appendChild(parent)
    const controller = { visible: ref(false), onVerified: vi.fn(), onCancel: vi.fn(), run: vi.fn() }
    const wrapper = mount(TotpStepUpDialog, { props: { controller }, attachTo: parent })
    try {
      controller.visible.value = true
      await flushPromises()
      const dialog = document.querySelector('[data-testid="totp-step-up-dialog"]')!
      expect(dialog.parentElement).toBe(document.body)
      const inputs = dialog.querySelectorAll<HTMLInputElement>('.totp-step-up-code-input')
      expect(inputs).toHaveLength(6)
      expect(document.activeElement).toBe(inputs[0])
      for (const [i, input] of Array.from(inputs).entries()) {
        input.value = String(i + 1)
        input.dispatchEvent(new Event('input', { bubbles: true }))
      }
      await flushPromises()
      expect(verify).toHaveBeenCalledWith('123456')
      expect(controller.onVerified).toHaveBeenCalledOnce()
    } finally {
      wrapper.unmount()
      parent.remove()
    }
  })
})
