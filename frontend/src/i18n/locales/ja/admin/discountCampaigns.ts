export default {
  discountCampaigns: {
    title: '期間限定割引',
    description: '通常グループのToken倍率に、一回限りまたは毎週繰り返す割引を設定します。',
    create: '割引を作成', edit: '割引を編集', empty: '割引キャンペーンはありません',
    emptyHint: 'キャンペーンを作成すると、有効期間中に割引後のToken倍率が表示・適用されます。',
    safetyTitle: '課金対象',
    safetyHint: '通常グループのToken課金だけが対象です。サブスクリプション、画像・動画・Web Searchの独立価格は対象外です。',
    fields: {
      name: 'キャンペーン名', description: 'ユーザー向け説明', descriptionPlaceholder: '割引のルールや説明を入力...', enabled: 'キャンペーンを有効化', scheduleType: 'スケジュール', timezone: 'タイムゾーン',
      startsAt: '開始日時', endsAt: '終了日時', weekdays: '曜日', allDay: '終日', startTime: '開始時刻', endTime: '終了時刻',
      discountPercent: '支払割合', minMultiplier: '割引後の最低倍率', budgetCap: '割引予算上限'
    },
    scheduleTypes: { one_time: '一回限り', weekly: '毎週繰り返し' },
    weekdays: { sun: '日', mon: '月', tue: '火', wed: '水', thu: '木', fri: '金', sat: '土' },
    hints: {
      discountPercent: '例：90%の場合、2xは1.8xになります。', crossMidnight: '終了時刻が開始時刻より前の場合は翌日まで継続します。',
      minimum: '任意。割引後の倍率はこの値を下回りません。',
      budget: '任意。記録済みの割引額が上限に達すると新規リクエストは対象外になり、処理中のリクエストは開始時の価格を維持します。',
      overlap: '複数のキャンペーンが重なる場合、最も低い実効倍率を適用します。',
      description: 'キャンペーン中、API キーページにユーザー向けに表示されます。'
    },
    columns: { campaign: 'キャンペーン', schedule: '期間', discount: '割引', budget: '割引済み / 予算', status: '状態', actions: '操作' },
    statuses: { active: '適用中', upcoming: '開始前', ended: '終了', disabled: '無効', budget_exhausted: '予算上限到達' },
    enabled: '有効', disabled: '無効', allDay: '終日', noLimit: '上限なし',
    save: '認証して保存', saving: '保存中...', delete: '削除', deleteTitle: '割引キャンペーンを削除',
    deleteConfirm: '「{name}」を削除しますか？新規リクエストには割引が適用されなくなります。',
    created: '割引キャンペーンを作成しました。', updated: '割引キャンペーンを更新しました。', deleted: '割引キャンペーンを削除しました。',
    errors: { load: '割引キャンペーンを読み込めませんでした', save: '割引キャンペーンを保存できませんでした', delete: '割引キャンペーンを削除できませんでした' }
  }
}
