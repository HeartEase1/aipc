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
    participation: 'ランキングに参加',
    notParticipating: '現在、ランキングに参加していません。上のスイッチを有効にすると、ランキングと集計に含まれます。',
    notRanked: 'この期間のランキングデータはありません。',
    period24h: '24時間',
    period72h: '72時間',
    period7d: '7日間',
    period30d: '30日間',
    usageTab: '利用ランキング',
    consumptionTab: '消費ランキング',
    rebateTab: '還元ランキング',
    rank: '順位',
    currentRank: '現在の順位: #{rank}',
    user: 'ユーザー',
    requests: 'リクエスト数',
    tokens: 'トークン',
    cost: '消費額',
    invitedUsers: '新規被紹介ユーザー',
    rebateCount: '還元件数',
    rebateAmount: '還元額',
    empty: 'この期間のランキングデータはありません。',
    loadFailed: 'ランキングを読み込めませんでした。',
    saveFailed: 'ランキング参加状態を更新できませんでした。'
  }
}
