import { describe, expect, it } from 'vitest'

import en from '@/i18n/locales/en'
import ja from '@/i18n/locales/ja'
import zh from '@/i18n/locales/zh'

const keys = [
  'consoleUiMode',
  'consoleUiModeHint',
  'consoleUiModeModern',
  'consoleUiModeLegacy',
] as const

describe('console interface locale catalogue', () => {
  it.each([
    ['English', en],
    ['Chinese', zh],
    ['Japanese', ja],
  ])('provides every console mode label in %s', (_language, catalogue) => {
    for (const key of keys) {
      expect(catalogue.admin.settings.site[key]).toEqual(expect.any(String))
      expect(catalogue.admin.settings.site[key].trim()).not.toBe('')
    }
  })

  it('keeps the Japanese catalogue complete through the English base catalogue', () => {
    expect(ja.admin.settings.site.hideCcsImportButton).toBe(
      en.admin.settings.site.hideCcsImportButton,
    )
    expect(ja.admin.settings.site.consoleUiModeModern).not.toBe(
      en.admin.settings.site.consoleUiModeModern,
    )
  })
})
