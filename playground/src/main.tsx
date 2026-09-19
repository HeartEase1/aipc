import 'core-js/actual/array/at'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import 'streamdown/styles.css'
import 'katex/dist/katex.min.css'
import './index.css'
import { installMobileViewportGuards } from './lib/viewport'
import {
  cleanupHostedServiceWorkers,
  clearHostedSensitiveUrlParams,
  isHostedMode,
  isHostedModeRequested,
  waitForHostedConfig,
} from './lib/hosted'

installMobileViewportGuards()

if (!__HOSTED_ONLY__ && !isHostedMode && 'serviceWorker' in navigator) {
  if (import.meta.env.PROD) {
    window.addEventListener('load', () => {
      navigator.serviceWorker.register(`${import.meta.env.BASE_URL}sw.js`).catch((error) => {
        console.error('Service worker registration failed:', error)
      })
    })
  } else {
    navigator.serviceWorker.getRegistrations().then((registrations) => {
      registrations.forEach((registration) => registration.unregister())
    })
  }
}

function renderBootstrapMessage(message: string, isError = false) {
  const root = document.getElementById('root')
  if (!root) return
  root.replaceChildren()
  const container = document.createElement('div')
  container.className = 'flex min-h-screen items-center justify-center px-6 text-center'
  const text = document.createElement('p')
  text.className = isError
    ? 'max-w-md text-sm leading-6 text-red-600 dark:text-red-400'
    : 'text-sm text-gray-500 dark:text-gray-400'
  text.textContent = message
  container.append(text)
  root.append(container)
}

async function bootstrap() {
  if (__HOSTED_ONLY__ || isHostedModeRequested) {
    try {
      clearHostedSensitiveUrlParams()
      if (await cleanupHostedServiceWorkers()) return
    } catch (error) {
      console.error('Failed to clear legacy playground state:', error)
      renderBootstrapMessage('工作台安全初始化失败，请清理本站浏览器数据后重试。', true)
      return
    }
  }

  if (__HOSTED_ONLY__ && !isHostedModeRequested) {
    renderBootstrapMessage('请从站内“在线使用”页面进入工作台。', true)
    return
  }

  if (isHostedModeRequested) {
    if (!isHostedMode) {
      renderBootstrapMessage('请从站内“在线使用”页面进入工作台。', true)
      return
    }
    renderBootstrapMessage('正在连接在线工作台...')
    try {
      await waitForHostedConfig()
    } catch (error) {
      renderBootstrapMessage(error instanceof Error ? error.message : '工作台连接失败，请刷新页面重试。', true)
      return
    }
  }

  const { default: App } = await import('./App')
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <App />
    </StrictMode>,
  )
}

void bootstrap()
