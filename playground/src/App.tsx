import { useEffect, useState } from 'react'
import { initStore } from './store'
import { useStore } from './store'
import { activateFirstImportedProfile, buildSettingsFromUrlParams, clearUrlSettingParams, hasUrlSettingParams } from './lib/urlSettings'
import { createDefaultOpenAIProfile, isDefaultConfigOnlyEnabled, mergeImportedSettings, normalizeSettings } from './lib/apiProfiles'
import { getCustomProviderConfigUrl, loadCustomProviderSettingsFromUrl } from './lib/customProviderConfigUrl'
import { useDockerApiUrlMigrationNotice } from './hooks/useDockerApiUrlMigrationNotice'
import { getHostedConfig, isHostedMode, subscribeHostedConfig, type HostedPlaygroundConfig } from './lib/hosted'
import type { AppSettings } from './types'
import Header from './components/Header'
import SearchBar from './components/SearchBar'
import TaskGrid from './components/TaskGrid'
import AgentWorkspace from './components/AgentWorkspace'
import InputBar from './components/InputBar'
import DetailModal from './components/DetailModal'
import Lightbox from './components/Lightbox'
import SettingsModal from './components/SettingsModal'
import ConfirmDialog from './components/ConfirmDialog'
import Toast from './components/Toast'
import MaskEditorModal from './components/MaskEditorModal'
import ImageContextMenu from './components/ImageContextMenu'
import SupportPromptModal from './components/SupportPromptModal'
import { FavoriteCollectionPickerModal, FavoriteCollectionsView, ManageCollectionsModal } from './components/FavoriteCollections'
import { useGlobalClickSuppression } from './lib/clickSuppression'

let customProviderConfigUrlImportStarted = false
let hostedStoreInitStarted = false

function createHostedSettings(settings: AppSettings, config: HostedPlaygroundConfig): AppSettings {
  const imageProfile = createDefaultOpenAIProfile({
    id: 'ipcai-hosted-image',
    name: '站内生图',
    baseUrl: config.baseUrl,
    apiKey: config.apiKey,
    model: config.imageModel,
    apiMode: 'images',
    apiProxy: false,
    // Prefer inline image data in hosted mode so external image URLs do not
    // depend on the upstream image host's CORS configuration.
    responseFormatB64Json: config.responseFormatB64Json !== false,
    streamImages: false,
  })
  const textProfile = createDefaultOpenAIProfile({
    id: 'ipcai-hosted-text',
    name: '站内对话',
    baseUrl: config.baseUrl,
    apiKey: config.apiKey,
    model: config.textModel,
    apiMode: 'responses',
    apiProxy: false,
    streamImages: true,
    streamPartialImages: 0,
  })

  return normalizeSettings({
    ...settings,
    baseUrl: imageProfile.baseUrl,
    apiKey: imageProfile.apiKey,
    model: imageProfile.model,
    apiMode: imageProfile.apiMode,
    codexCli: false,
    apiProxy: false,
    streamImages: false,
    customProviders: [],
    providerOrder: ['openai'],
    profiles: [imageProfile, textProfile],
    activeProfileId: imageProfile.id,
    agentApiConfigMode: 'hybrid',
    agentTextProfileId: textProfile.id,
    agentImageProfileId: imageProfile.id,
    agentWebSearch: false,
    agentMaxToolRounds: 6,
    reuseTaskApiProfileTemporarily: false,
    allowPromptRewrite: false,
    // 父页面刷新托管 API 配置（例如切换模型或主题）时保留本地偏好。
    zipDownloadRoutes: settings.zipDownloadRoutes,
  })
}

export default function App() {
  const setSettings = useStore((s) => s.setSettings)
  const setHostedSettings = useStore((s) => s.setHostedSettings)
  const appMode = useStore((s) => s.appMode)
  const filterFavorite = useStore((s) => s.filterFavorite)
  const activeFavoriteCollectionId = useStore((s) => s.activeFavoriteCollectionId)
  const [hostedConnected, setHostedConnected] = useState(!isHostedMode || Boolean(getHostedConfig()))
  const [hostedError, setHostedError] = useState('')
  useDockerApiUrlMigrationNotice()
  useGlobalClickSuppression()

  useEffect(() => {
    if (isHostedMode) {
      const applyConfig = (config: HostedPlaygroundConfig | null) => {
        if (!config) {
          const state = useStore.getState()
          state.setHostedSettings(normalizeSettings({
            ...state.settings,
            apiKey: '',
            profiles: state.settings.profiles.map((profile) => ({ ...profile, apiKey: '' })),
          }))
          setHostedConnected(false)
          return
        }

        const state = useStore.getState()
        state.setHostedSettings(createHostedSettings(state.settings, config))
        setHostedConnected(true)
        setHostedError('')
        if (!hostedStoreInitStarted) {
          hostedStoreInitStarted = true
          void initStore().catch((error) => {
            console.error('Failed to initialize hosted playground data:', error)
            setHostedError('本地会话数据加载失败，请刷新页面重试。')
          })
        }
      }

      applyConfig(getHostedConfig())
      return subscribeHostedConfig(applyConfig)
    }

    const searchParams = new URLSearchParams(window.location.search)
    const customProviderConfigUrl = getCustomProviderConfigUrl()
    const defaultConfigOnly = isDefaultConfigOnlyEnabled()

    const applyUrlSettings = (baseSettings: Partial<AppSettings>) => {
      const nextSettings = buildSettingsFromUrlParams(baseSettings, searchParams)
      return Object.keys(nextSettings).length ? nextSettings : baseSettings
    }

    const clearAppliedUrlSettings = () => {
      if (!hasUrlSettingParams(searchParams)) return

      clearUrlSettingParams(searchParams)

      const nextSearch = searchParams.toString()
      const nextUrl = `${window.location.pathname}${nextSearch ? `?${nextSearch}` : ''}${window.location.hash}`
      window.history.replaceState(null, '', nextUrl)
    }

    if (customProviderConfigUrl && defaultConfigOnly && !customProviderConfigUrlImportStarted) {
      customProviderConfigUrlImportStarted = true
      void loadCustomProviderSettingsFromUrl(customProviderConfigUrl)
        .then((importedSettings) => {
          const state = useStore.getState()
          const baseSettings = importedSettings
            ? activateFirstImportedProfile(mergeImportedSettings(state.settings, importedSettings), importedSettings)
            : state.settings
          state.setSettings(applyUrlSettings(baseSettings))
          clearAppliedUrlSettings()
        })
        .catch((error) => {
          console.warn('Failed to import custom provider config URL:', error)
          const state = useStore.getState()
          state.setSettings(applyUrlSettings(state.settings))
          clearAppliedUrlSettings()
        })

      initStore()
      return
    }

    const nextSettings = buildSettingsFromUrlParams(useStore.getState().settings, searchParams)

    setSettings(nextSettings)

    clearAppliedUrlSettings()

    if (customProviderConfigUrl && !customProviderConfigUrlImportStarted) {
      customProviderConfigUrlImportStarted = true
      void loadCustomProviderSettingsFromUrl(customProviderConfigUrl)
        .then((importedSettings) => {
          if (!importedSettings) return
          const state = useStore.getState()
          state.setSettings(mergeImportedSettings(state.settings, importedSettings))
        })
        .catch((error) => {
          console.warn('Failed to import custom provider config URL:', error)
        })
    }

    initStore()
  }, [setHostedSettings, setSettings])

  useEffect(() => {
    const preventPageImageDrag = (e: DragEvent) => {
      if ((e.target as HTMLElement | null)?.closest('img')) {
        e.preventDefault()
      }
    }

    document.addEventListener('dragstart', preventPageImageDrag)
    return () => document.removeEventListener('dragstart', preventPageImageDrag)
  }, [])

  if (isHostedMode && (!hostedConnected || hostedError)) {
    return (
      <main className="flex min-h-screen items-center justify-center px-6 text-center">
        <p className={hostedError ? 'max-w-md text-sm leading-6 text-red-600 dark:text-red-400' : 'text-sm text-gray-500 dark:text-gray-400'}>
          {hostedError || '工作台连接已断开，请刷新页面重试。'}
        </p>
      </main>
    )
  }

  return (
    <>
      <Header />
      {appMode === 'agent' ? (
        <AgentWorkspace />
      ) : (
        <main data-home-main data-drag-select-surface className="pb-48">
          <div className="safe-area-x max-w-7xl mx-auto">
            <SearchBar />
            {filterFavorite && !activeFavoriteCollectionId ? <FavoriteCollectionsView /> : <TaskGrid />}
          </div>
        </main>
      )}
      <InputBar />
      <DetailModal />
      <Lightbox />
      <SettingsModal />
      <ConfirmDialog />
      {!isHostedMode && <SupportPromptModal />}
      <FavoriteCollectionPickerModal />
      <ManageCollectionsModal />
      <Toast />
      <MaskEditorModal />
      <ImageContextMenu />
    </>
  )
}
