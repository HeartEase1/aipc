export default {
  discountCampaigns: {
    title: '限时折扣活动',
    description: '为普通分组设置一次性或每周循环的 Token 倍率折扣。',
    create: '新建折扣',
    edit: '编辑折扣',
    empty: '暂无折扣活动',
    emptyHint: '创建一次性或每周循环活动后，系统会在有效时段展示并应用折后 Token 倍率。',
    safetyTitle: '计费范围',
    safetyHint: '折扣只作用于普通分组的 Token 计费；订阅分组以及独立定价的生图、视频和 Web Search 均不参与。',
    fields: {
      name: '活动名称', enabled: '启用活动', scheduleType: '活动周期', timezone: '活动时区',
      startsAt: '开始时间', endsAt: '结束时间', weekdays: '每周日期', allDay: '全天',
      startTime: '每日开始', endTime: '每日结束', discountPercent: '实付比例',
      minMultiplier: '折后最低倍率', budgetCap: '优惠预算上限'
    },
    scheduleTypes: { one_time: '一次性活动', weekly: '每周循环' },
    weekdays: { sun: '周日', mon: '周一', tue: '周二', wed: '周三', thu: '周四', fri: '周五', sat: '周六' },
    hints: {
      discountPercent: '例如填写 90%，原 2 倍率会变为 1.8 倍率，也就是九折。',
      crossMidnight: '结束时间早于开始时间时，活动会自动延续到次日。',
      minimum: '可选。应用折扣后的倍率不会低于此值。',
      budget: '可选。已记录的优惠金额达到上限后，新请求不再参加；已开始的请求仍保留开始时价格。',
      overlap: '多个活动重叠时，自动采用折后倍率最低的活动。'
    },
    columns: { campaign: '活动', schedule: '活动时间', discount: '折扣', budget: '已优惠 / 预算', status: '状态', actions: '操作' },
    statuses: { active: '生效中', upcoming: '未开始', ended: '已结束', disabled: '未启用', budget_exhausted: '预算已用完' },
    enabled: '已启用', disabled: '未启用', allDay: '全天', noLimit: '不限',
    save: '验证并保存', saving: '正在保存...', delete: '删除', deleteTitle: '删除折扣活动',
    deleteConfirm: '确认删除“{name}”吗？删除后新请求将不再享受该折扣。',
    created: '折扣活动已创建。', updated: '折扣活动已更新。', deleted: '折扣活动已删除。',
    errors: { load: '加载折扣活动失败', save: '保存折扣活动失败', delete: '删除折扣活动失败' }
  }
}
