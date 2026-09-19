export default {
  benefitGrants: {
    title: '福利与补偿发放',
    description: '通过金额预览和二次验证，安全地向用户增加可用余额。',
    tabs: { create: '新建发放', history: '发放记录' },
    sections: { rules: '发放规则', protection: '金额保护', notification: '到账通知' },
    fields: {
      includeSubscription: '包含订阅用户用量', subscriptionPercentage: '订阅用量补偿比例',
      type: '发放类型', mode: '发放方式', audience: '发放对象', fixedAmount: '每人固定额度',
      percentage: '消费补偿比例', percentagePeriod: '消费统计时间', platformIds: '按平台 ID 指定',
      customWindowStart: '开始时间', customWindowEnd: '结束时间', minAmount: '最低发放额', perUserCap: '单用户封顶',
      totalBudgetCap: '整批预算上限', reason: '发放原因', notificationTitle: '弹窗标题',
      notificationContent: '弹窗内容（支持 Markdown）', user: '用户'
    },
    types: { welfare: '福利', compensation: '补偿' },
    modes: { fixed: '固定额度', percentage_24h: '按消费比例' },
    periods: { '24h': '近 24 小时', '72h': '近 72 小时', '30d': '近 30 天', custom: '自定义' },
    audiences: { all: '全部符合条件的用户', selected: '指定用户' },
    audienceHints: {
      all: '发放给所有状态正常、未删除的普通用户。',
      selected: '可输入平台 ID，或搜索并选择用户；合计最多 500 名。'
    },
    platformIDPlaceholder: '例如：1024, 2048, 4096',
    platformIDHint: '支持逗号、空格或换行分隔，已识别 {count} 个平台 ID。',
    orSearchUsers: '或搜索选择用户',
    searchUsers: '按邮箱或用户名搜索用户',
    customWindowHint: '按本机时区输入起止时间，最长可统计 365 天；预览后将锁定为准确时间范围。',
    selectedCount: '已选择 {count} 人',
    summary: { title: '当前规则' },
    walletPercentageHint: '按所选时间段内余额计费的实际消费计算。',
    subscriptionPercentageHint: '可选。订阅计费的用量单独按此比例计算，最终仍统一发放到钱包余额。',
    safety: {
      title: '不会计入充值和返利',
      content: '本功能只增加用户可用余额，不增加累计充值、不触发邀请返利，也不修改计费、价格或订阅额度。'
    },
    preview: '预览发放',
    previewing: '正在生成预览...',
    previewNotification: '预览到账弹窗',
    confirmTitle: '确认发放',
    confirmAcknowledgement: '我已核对人数、计算规则与预计总额，并确认立即执行。',
    execute: '二次验证并发放',
    executing: '正在提交...',
    submitted: '批次已提交，将在后台持续处理。',
    retryFailed: '重试失败明细',
    retrySubmitted: '失败明细已重新提交。',
    detailTitle: '发放明细',
    emptyHistory: '暂无发放批次',
    allStatuses: '全部状态',
    overBudget: '预计总额超过整批预算上限，无法执行。',
    columns: { batch: '批次', progress: '成功/预计', amount: '金额', created: '创建时间' },
    metrics: {
      walletBaseCost: '余额消费基数', subscriptionBaseCost: '订阅用量基数',
      walletAmount: '余额消费补偿', subscriptionAmount: '订阅用量补偿',
      walletShort: '余额', subscriptionShort: '订阅',
      recipients: '预计发放人数', skipped: '跳过人数', baseCost: '所选时段消费基数',
      totalAmount: '预计总额', average: '平均金额', maximum: '最大单笔', amount: '发放金额',
      succeeded: '成功人数', failed: '失败人数', distributed: '已发放金额', window: '锁定消费窗口'
    },
    statuses: {
      draft: '待确认', pending: '等待处理', processing: '处理中', completed: '已完成',
      partially_failed: '部分失败', failed: '失败', expired: '已过期'
    },
    itemStatuses: {
      pending: '等待处理', succeeded: '已到账', failed: '失败', skipped_ineligible: '资格失效，已跳过'
    },
    defaults: {
      title: '您收到了一笔到账',
      previewReason: '示例发放原因',
      content: '您已收到 **{\'{{amount}}\'}** 额度。\n\n发放原因：{\'{{reason}}\'}\n\n当前余额：{\'{{balance}}\'}\n\n感谢您使用 {\'{{site_name}}\'}。'
    },
    errors: {
      preview: '生成发放预览失败', execute: '提交发放失败', load: '加载发放记录失败',
      retry: '重试失败明细失败', export: '导出发放明细失败', search: '搜索用户失败',
      selectedLimit: '手动指定一次最多支持 500 名用户',
      invalidPlatformIDs: '以下平台 ID 格式不正确：{values}'
    }
  }
}
