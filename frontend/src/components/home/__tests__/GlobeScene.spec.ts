import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../GlobeScene.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('GlobeScene interaction controls', () => {
  it('uses bounded OrbitControls without allowing pan', () => {
    expect(componentSource).toContain("OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'")
    expect(componentSource).toContain('controls.enablePan=false')
    expect(componentSource).toContain('controls.minDistance=MIN_CAMERA_DISTANCE')
    expect(componentSource).toContain('controls.maxDistance=MAX_CAMERA_DISTANCE')
    expect(componentSource).toContain("controls.addEventListener('change',onControlsChange)")
  })

  it('pauses automatic rotation during interaction and resumes after a delay', () => {
    expect(componentSource).toContain("controls.addEventListener('start',onControlsStart)")
    expect(componentSource).toContain("controls.addEventListener('end',onControlsEnd)")
    expect(componentSource).toContain('AUTO_ROTATE_RESUME_DELAY_MS = 3000')
  })

  it('provides an accessible reset control', () => {
    expect(componentSource).toContain('data-testid="globe-reset"')
    expect(componentSource).toContain("t('home.resetGlobeView')")
    expect(componentSource).toContain('function resetView()')
  })

  it('renders every provider on two crossed, counter-rotating orbits', () => {
    expect(componentSource).toContain("id: 'primary'")
    expect(componentSource).toContain("id: 'cross'")
    expect(componentSource).toContain('order: [0, 1, 2, 3]')
    expect(componentSource).toContain('order: [3, 2, 1, 0]')
    expect(componentSource).toContain('roll: 0.64')
    expect(componentSource).toContain('roll: -0.64')
    expect(componentSource).toContain('const LOGO_ORBIT_PITCH = 0.34')
    expect(componentSource).toContain('speed: -0.19')
    expect(componentSource).toContain('phase: Math.PI * 0.35')
    expect(componentSource).toContain('LOGO_ORBITS.flatMap')
    expect(componentSource).toContain(':data-orbit="logo.orbitId"')
  })
})
