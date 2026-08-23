import en from '../en'
import discountCampaigns from './admin/discountCampaigns'

// Japanese currently reuses the complete English catalogue for unchanged
// screens, while the leaderboard copy is translated locally.
export default {
  ...en,
  nav: {
    ...en.nav,
    leaderboard: 'ランキング',
    onlinePlayground: 'オンライン利用',
    usageGuide: '利用ガイド',
    benefits: '入金履歴',
    benefitGrants: '特典付与',
    discountCampaigns: '期間限定割引'
  },
  usageGuide: {
    title: '利用ガイド',
    description: 'APIキーの作成から対応クライアントの設定までを説明します。'
  },
  userSubscriptions: {
    ...en.userSubscriptions,
    usedPercentage: '使用済み {percentage}%'
  },
  subscriptionProgress: {
    ...en.subscriptionProgress,
    used: '使用済み'
  },
  onlinePlayground: {
    ...en.onlinePlayground,
    title: 'オンライン利用',
    description: 'サイト内のAPIキーを使って、複数ターンのチャット、画像生成、画像編集を利用できます。',
    configuration: 'ワークスペース設定',
    textChat: 'マルチウィンドウチャット',
    imageGeneration: 'オンライン画像生成',
    selectKey: 'APIキー',
    selectKeyPlaceholder: '有効なキーを選択',
    unnamedKey: '名称未設定のキー',
    selectTextModel: 'チャットモデル',
    selectImageModel: '画像モデル',
    selectModelPlaceholder: 'モデルを選択',
    base64Images: 'Base64画像データを返す',
    base64ImagesHint: '画像URLのクロスオリジン制限を避けるため、まず埋め込みデータを要求します。対応しないサービスではコピー用に元のURLを保持します。',
    base64ImagesOn: 'Base64画像データを有効にしています',
    base64ImagesOff: '無効、サービスが返した画像URLを使用します',
    keyPrivacyNotice: '設定を保存しても、このブラウザに記録されるのはキーID、モデル、画像設定だけで、キー本体は保存されません。キー本体はこのページのメモリ内だけで使用され、URL、チャット履歴、ローカル履歴には保存されません。すべてのリクエストはサイトのゲートウェイを経由して従来のルールで課金されます。Agent は1つの操作で複数の課金対象リクエストを行う場合があり、対話は最大6ラウンド、画像生成は最大6件です。',
    loadingKeys: '利用可能なキーを読み込んでいます…',
    loadingModels: 'モデル一覧を取得しています…',
    keysLoadFailed: 'APIキーを読み込めませんでした。ネットワークを確認して再試行してください。',
    modelsLoadFailed: '選択したキーでモデル一覧を取得できませんでした。キー、グループ、IP制限を確認してください。',
    httpStatus: 'リクエスト状態：{status}',
    noActiveKey: '利用可能なAPIキーがありません',
    createKeyHint: '先にAPIキーを作成して有効にしてください。',
    goToKeys: 'APIキーを作成',
    saveConfiguration: '設定を保存',
    configurationSaved: '設定を保存しました',
    saveConfigurationFailed: 'ブラウザに設定を保存できませんでした。サイトストレージが無効になっていないか確認してください。',
    refreshModels: 'モデルを更新',
    retry: '再試行',
    workspace: 'オンライン制作ワークスペース',
    iframeTitle: 'オンラインチャット・画像生成ワークスペース',
    connecting: 'ワークスペースに接続しています…',
    connected: 'ワークスペースに接続済み',
    connectionFailed: 'ワークスペースへの接続に失敗しました',
    connectionFailedHint: '静的ファイルがまだビルドされていないか、接続がタイムアウトしました。再接続してください。',
    reconnect: '再接続',
    enterFullscreen: 'ワークスペースを全画面表示',
    exitFullscreen: '全画面表示を終了',
    localHistoryNotice: '会話、タスク、画像はアカウント別のブラウザストレージにのみ保存され、クラウドには同期されません。同じブラウザプロファイルを使う他の利用者がローカルデータを閲覧できる可能性があるため、共有端末では利用後にこのサイトのブラウザデータを消去してください。'
  },
  keys: {
    ...en.keys,
    fastMode: 'Fastモード',
    fastModeHint: '有効にすると、このキーのOpenAIリクエストはPriorityサービス層を使用します。通常は標準料金の2倍で課金されますが、実際のチャネル設定が優先されます。OpenAI以外のグループには影響しません。',
    discountCampaign: {
      currentDiscount: '現在の割引',
      remaining: '残り {time}',
      remainingDays: '残り {days}日 {time}',
      endingSoon: 'まもなく終了',
      discountValue: '{percent}% OFF',
      balanceOnly: '残高払いのリクエストのみ対象',
      subscriptionExcluded: 'サブスクリプション利用は割引対象外'
    }
  },
  payment: {
    ...en.payment,
    restartNow: '今すぐリセット',
    restart: {
      ...en.payment.restart,
      selectedTitle: '即時リセットを選択中',
      selectedDescription: '支払い完了後、現在の残り期間と未使用枠は失効し、直ちに新しい完全な契約期間が始まります。',
      reviewAndPay: '即時リセットを確認 {amount}',
      confirmTitle: 'サブスクリプションの即時リセットを確認',
      confirmDescription: '失効する現在の特典をご確認ください。サブスクリプションは支払い成功後にのみ変更されます。',
      forfeitTitle: '現在の未使用特典は繰り越されません',
      forfeitDescription: '残り期間と各上限の未使用枠は支払い成功時に失効し、復元できません。',
      currentTerm: '現在のサブスクリプション',
      remainingTime: '残り期間',
      daysRemaining: '約 {days} 日',
      used: '使用済み',
      remainingForfeit: '残り / 失効',
      newTerm: 'リセット後の新しい契約',
      newValidity: '完全な期間',
      estimatedExpiry: '予定有効期限',
      paymentAmount: '支払金額',
      paymentSafety: '支払いの失敗、キャンセル、またはタイムアウト時は、現在の期間と利用枠は変更されません。新しい期間は実際の支払い完了時から始まります。',
      confirmPayment: '{amount} を支払い今すぐリセット',
      unavailable: '現在、このサブスクリプションは即時リセットの対象外です。ページを更新して再度お試しください。',
    },
  },
  leaderboard: {
    ...en.leaderboard,
    title: 'ランキング',
    description: '参加ユーザーの利用、消費、紹介還元を確認できます。',
    privacyNote: 'ユーザー名が未設定の場合、メールアドレスはマスクされて表示されます。',
    periodLabel: '集計期間',
    participation: 'ランキングに参加',
    participationHint: 'オフにするとランキングと集計から除外されます',
    notParticipating: '現在、ランキングに参加していません。上のスイッチを有効にすると、ランキングと集計に含まれます。',
    notRanked: 'この期間のランキングデータはありません。',
    period24h: '24時間',
    period72h: '72時間',
    period7d: '7日間',
    period30d: '30日間',
    usageTab: '利用ランキング',
    consumptionTab: '消費ランキング',
    rebateTab: '還元ランキング',
    summaryScope: '選択期間内に参加している全ユーザーの合計です。',
    usageRule: '消費トークンの降順です。各項目は選択期間内の合計です。',
    consumptionRule: '実際の消費額の降順です。各項目は選択期間内の合計です。',
    rebateRule: '還元額の降順です。招待ユーザーは選択期間内に新規紐付けされた人数です。',
    shareOverview: '合計に占める割合',
    usageShareDescription: '消費トークンを基準に、上位20名とその他の参加ユーザーが総トークンに占める割合を表示します。',
    consumptionShareDescription: '消費額を基準に、上位20名とその他の参加ユーザーが総消費額に占める割合を表示します。',
    rebateShareDescription: '還元額を基準に、上位20名とその他の参加ユーザーが総還元額に占める割合を表示します。',
    otherUsers: 'その他',
    top20: '上位20名',
    myData: '自分のデータ',
    totalRequests: '総リクエスト数',
    totalTokens: '総トークン',
    totalCost: '総消費額',
    totalInvitedUsers: '総招待ユーザー数',
    totalRebateCount: '総還元件数',
    totalRebateAmount: '総還元額',
    rank: '順位',
    currentRank: '現在の順位: #{rank}',
    user: 'ユーザー',
    platformId: 'プラットフォームID',
    requests: 'リクエスト数',
    tokens: 'トークン',
    cost: '消費額',
    invitedUsers: '新規被紹介ユーザー',
    rebateCount: '還元件数',
    rebateAmount: '還元額',
    share: '占有率',
    empty: 'この期間のランキングデータはありません。',
    loadFailed: 'ランキングを読み込めませんでした。',
    saveFailed: 'ランキング参加状態を更新できませんでした。'
  },
  benefits: {
    ...en.benefits,
    title: '入金履歴',
    description: '管理者から付与された特典と補償の履歴を確認できます。',
    historyTitle: 'アカウント履歴',
    historyDescription: 'コード交換、プラン、管理者調整、特典、補償を時系列でまとめて表示します。',
    empty: 'アカウント履歴はありません',
    emptyHint: 'コード交換、プラン、特典、補償の履歴がここに保存されます。',
    loadFailed: '入金履歴を読み込めませんでした。もう一度お試しください。',
    unread: '未読',
    popupBadge: '残高入金',
    grantReason: '付与理由',
    balanceAfter: '付与後の残高：${amount}',
    acknowledge: '確認しました',
    calculation: {
      title: '今回の補償明細',
      balanceSpending: '残高利用分',
      subscriptionSpending: 'プラン利用分',
      calculatedAmount: '割合による計算額',
      ruleAdjustment: '付与ルールによる調整',
      actualAmount: '実際の入金額',
      window: '利用集計期間',
      windowStart: '開始',
      windowEnd: '終了'
    },
    types: { welfare: '特典', compensation: '補償' }
  },
  admin: {
    ...en.admin,
    ...discountCampaigns,
    usage: {
      ...en.admin.usage,
      collapseUserDetails: 'ユーザー情報を折りたたむ',
      expandUserDetails: 'ユーザー情報を展開',
      collapseResponseModelDetails: '応答モデルを折りたたむ',
      expandResponseModelDetails: '応答モデルを展開'
    },
    accounts: {
      ...en.admin.accounts,
      cnProviders: {
        ...en.admin.accounts.cnProviders,
        balanceAutoPause: '残高不足時の自動停止',
        balanceAutoPauseHint: '残高を正しく取得できない非公式アップストリームでは無効にできます。認証失敗、通常のレート制限、手動停止は引き続き有効です。'
      }
    },
    settings: {
      ...en.admin.settings,
      site: {
        ...en.admin.settings.site,
        consoleUiMode: 'コンソール画面',
        consoleUiModeHint: 'すべてのログインユーザーに表示する画面を選択します。ランディングページは変更されません。',
        consoleUiModeModern: 'モダンコンソール',
        consoleUiModeLegacy: '従来のコンソール'
      },
      features: {
        ...en.admin.settings.features,
        onlinePlayground: {
          ...en.admin.settings.features.onlinePlayground,
          title: 'オンライン利用',
          description: 'ブラウザ内のチャット、画像生成、画像編集ワークスペースをユーザーに公開するか設定します。',
          enabled: 'オンライン利用を有効化',
          enabledHint: 'オフにするとユーザー側の入口を隠し、/playground への直接アクセスもブロックします。API と既存の課金には影響しません。'
        }
      }
    },
    users: {
      ...en.admin.users,
      benefitGrant: '特典・補償を付与',
      typeWelfare: '特典付与',
      typeCompensation: '補償付与',
      balanceAddedWelfare: '管理者からの特典',
      balanceAddedCompensation: '管理者からの補償'
    },
    benefitGrants: {
      ...en.admin.benefitGrants,
      title: '特典・補償の付与',
      description: '金額のプレビューと追加認証を使用して、ユーザー残高を安全に追加します。',
      tabs: { create: '新規付与', history: '付与履歴' },
      sections: { rules: '付与ルール', protection: '金額保護', notification: '入金通知' },
      fields: {
        includeSubscription: 'サブスクリプション利用分を含める', subscriptionPercentage: 'サブスクリプション利用分の補償率',
        type: '付与タイプ', mode: '付与方法', audience: '対象ユーザー', fixedAmount: '1人あたりの固定額',
        percentage: '消費額に対する補償率', percentagePeriod: '消費集計期間', platformIds: 'プラットフォームIDで指定',
        customWindowStart: '開始日時', customWindowEnd: '終了日時', minAmount: '最低付与額', perUserCap: 'ユーザーごとの上限',
        totalBudgetCap: 'バッチ予算上限', reason: '付与理由', notificationTitle: 'ポップアップタイトル',
        notificationContent: 'ポップアップ内容（Markdown対応）', user: 'ユーザー'
      },
      types: { welfare: '特典', compensation: '補償' },
      modes: { fixed: '固定額', percentage_24h: '消費額の割合' },
      periods: { '24h': '直近24時間', '72h': '直近72時間', '30d': '直近30日', custom: 'カスタム' },
      audiences: { all: '対象となる全ユーザー', selected: '選択したユーザー' },
      audienceHints: {
        all: '有効で削除されていない一般ユーザー全員に付与します。',
        selected: 'プラットフォームIDの入力または検索で、合計500人まで指定できます。'
      },
      platformIDPlaceholder: '例：1024, 2048, 4096',
      platformIDHint: 'カンマ、空白、改行で区切ります。{count}件のプラットフォームIDを認識しました。',
      orSearchUsers: 'またはユーザーを検索',
      searchUsers: 'メールアドレスまたはユーザー名で検索',
      customWindowHint: '端末のタイムゾーンで入力してください。最長365日で、プレビュー時に期間が固定されます。',
      selectedCount: '{count}人を選択済み',
      summary: { title: '現在のルール' },
      walletPercentageHint: '選択期間内の残高課金による実消費額から計算します。',
      subscriptionPercentageHint: '任意。サブスクリプション課金の利用分を別の割合で計算し、最終的に残高へ付与します。',
      safety: {
        title: 'チャージ額・紹介還元には算入されません',
        content: '利用可能残高のみを追加します。累計チャージ額、紹介還元、課金、価格、サブスクリプション枠には影響しません。'
      },
      preview: '付与内容をプレビュー', previewing: 'プレビューを作成中...', previewNotification: '入金ポップアップをプレビュー', confirmTitle: '付与の確認',
      confirmAcknowledgement: '対象人数、計算ルール、予定総額を確認し、即時実行に同意します。',
      execute: '認証して付与', executing: '送信中...', submitted: 'バッチを送信しました。バックグラウンドで処理されます。',
      retryFailed: '失敗項目を再試行', retrySubmitted: '失敗項目を再送信しました。', detailTitle: '付与明細',
      emptyHistory: '付与バッチはありません', allStatuses: 'すべての状態',
      overBudget: '予定総額がバッチ予算上限を超えているため実行できません。',
      columns: { batch: 'バッチ', progress: '成功 / 予定', amount: '金額', created: '作成日時' },
      metrics: {
        walletBaseCost: '残高消費の基準額', subscriptionBaseCost: 'サブスクリプション利用の基準額',
        walletAmount: '残高消費分の補償', subscriptionAmount: 'サブスクリプション利用分の補償',
        walletShort: '残高', subscriptionShort: 'サブスクリプション',
        recipients: '予定対象人数', skipped: 'スキップ', baseCost: '選択期間の消費基準額', totalAmount: '予定総額',
        average: '平均額', maximum: '最大付与額', amount: '付与額', succeeded: '成功', failed: '失敗',
        distributed: '付与済み金額', window: '固定された消費期間'
      },
      statuses: {
        draft: '確認待ち', pending: '待機中', processing: '処理中', completed: '完了',
        partially_failed: '一部失敗', failed: '失敗', expired: '期限切れ'
      },
      itemStatuses: { pending: '待機中', succeeded: '入金済み', failed: '失敗', skipped_ineligible: '対象外のためスキップ' },
      defaults: {
        title: '残高が付与されました',
        previewReason: '付与理由の例',
        content: '**{\'{{amount}}\'}** の残高が付与されました。\n\n理由：{\'{{reason}}\'}\n\n現在の残高：{\'{{balance}}\'}\n\n{\'{{site_name}}\'} をご利用いただきありがとうございます。'
      },
      errors: {
        preview: '付与プレビューを作成できませんでした', execute: '付与を送信できませんでした',
        load: '付与履歴を読み込めませんでした', retry: '失敗項目を再試行できませんでした',
        export: '付与明細をエクスポートできませんでした', search: 'ユーザーを検索できませんでした',
        selectedLimit: '手動で指定できるユーザーは最大500人です',
        invalidPlatformIDs: '次のプラットフォームIDは無効です：{values}'
      }
    }
  }
}
