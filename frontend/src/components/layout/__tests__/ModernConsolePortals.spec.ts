import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const stylesPath = resolve(
  dirname(fileURLToPath(import.meta.url)),
  '../../../styles/modern-console.css',
)
const styles = readFileSync(stylesPath, 'utf8')

describe('modern console floating surface scope', () => {
  it('styles in-shell header menus without changing the global dropdown contract', () => {
    expect(styles).toContain('.modern-console-shell .dropdown {')
    expect(styles).toContain('.modern-console-shell .dropdown-item:not(.text-red-600)')
    expect(styles).not.toMatch(/(?:^|\n)\.dropdown\s*\{/)
  })

  it('covers the shared teleported dialog and select contracts only in modern mode', () => {
    expect(styles).toContain(
      "html[data-console-ui-mode='modern'] :is(.modal-content, .dialog-container, .select-dropdown-portal)",
    )
    expect(styles).toContain(
      "html[data-console-ui-mode='modern'] .select-dropdown-portal .select-search",
    )
    expect(styles).not.toMatch(/(?:^|\n)\.select-dropdown-portal\s*\{/)
  })

  it('covers toast, announcement, and tooltip portals through stable DOM contracts', () => {
    expect(styles).toContain(
      "html[data-console-ui-mode='modern'] [aria-live='polite'][aria-atomic='true'] > div",
    )
    expect(styles).toContain(
      "html[data-console-ui-mode='modern'] .announcement-popup-panel",
    )
    expect(styles).toContain(
      "html[data-console-ui-mode='modern'] [role='tooltip']",
    )
  })

  it('keeps dialog controls on the modern token layer after Teleport', () => {
    expect(styles).toContain(
      ":is(.modal-content, .dialog-container)\n  :is(.btn-primary, .btn-success)",
    )
    expect(styles).toContain(
      ":is(.modal-content, .dialog-container)\n  :is(.input, .select-trigger)",
    )
    expect(styles).toContain("[role='switch'][aria-checked='true']")
  })

  it('preserves toast status borders while adding a neutral surface outline', () => {
    const toastRule = styles.match(
      /html\[data-console-ui-mode='modern'\] \[aria-live='polite'\]\[aria-atomic='true'\] > div \{([^}]*)\}/,
    )

    expect(toastRule?.[1]).toContain('outline: 1px solid var(--console-border);')
    expect(toastRule?.[1]).not.toContain('border-color:')
  })
})
