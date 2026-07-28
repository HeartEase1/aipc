import en from '../en'

// Japanese currently reuses the complete English catalogue for unchanged
// screens, while the leaderboard copy is translated locally.
export default {
  ...en,
  nav: {
    ...en.nav,
    leaderboard: 'ランキング',
    benefits: '入金履歴',
    benefitGrants: '特典付与'
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
    empty: '入金履歴はありません',
    emptyHint: '特典または補償が付与されると、ここに履歴が保存されます。',
    unread: '未読',
    popupBadge: '残高入金',
    balanceAfter: '付与後の残高：${amount}',
    acknowledge: '確認しました',
    types: { welfare: '特典', compensation: '補償' }
  },
  admin: {
    ...en.admin,
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
        type: '付与タイプ', mode: '付与方法', audience: '対象ユーザー', fixedAmount: '1人あたりの固定額',
        percentage: '直近24時間の消費額に対する補償率', minAmount: '最低付与額', perUserCap: 'ユーザーごとの上限',
        totalBudgetCap: 'バッチ予算上限', reason: '付与理由', notificationTitle: 'ポップアップタイトル',
        notificationContent: 'ポップアップ内容（Markdown対応）', user: 'ユーザー'
      },
      types: { welfare: '特典', compensation: '補償' },
      modes: { fixed: '固定額', percentage_24h: '24時間消費率' },
      audiences: { all: '対象となる全ユーザー', selected: '選択したユーザー' },
      audienceHints: {
        all: '有効で削除されていない一般ユーザー全員に付与します。',
        selected: '対象となる一般ユーザーを最大500人まで選択します。'
      },
      searchUsers: 'メールアドレスまたはユーザー名で検索',
      selectedCount: '{count}人を選択済み',
      summary: { title: '現在のルール' },
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
        recipients: '予定対象人数', skipped: 'スキップ', baseCost: '24時間消費基準額', totalAmount: '予定総額',
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
        selectedLimit: '手動で選択できるユーザーは最大500人です'
      }
    }
  }
}
