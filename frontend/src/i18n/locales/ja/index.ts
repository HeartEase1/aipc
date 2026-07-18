import en from '../en'

// Japanese currently reuses the complete English catalogue for unchanged
// screens, while the leaderboard copy is translated locally.
export default {
  ...en,
  nav: {
    ...en.nav,
    leaderboard: 'ランキング'
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
  }
}
