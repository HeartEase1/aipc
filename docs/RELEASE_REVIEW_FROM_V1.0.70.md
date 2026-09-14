# AIPC 相对 v1.0.70 的发布审查

审查日期：2026-09-14。代码检查点：33321c9d0。未创建 Release。
旧 UPSTREAM_V0.2.4 文档包含隔离分支和中间实现的记录，不能作为当前主分支完成清单。

## 结论

暂不建议发布整个主分支。CI 通过并不排除未覆盖的计费缺陷。

确认阻断项：Image 2.5 渠道按图计费，多图请求可能只计一次。
calculateOpenAIRecordUsageCost 在没有分组图片覆盖价时选择 token 路径，
后续解析到渠道 BillingModeImage，却传入 RequestCount=1。
审查时用内存渠道价格 $0.20、ImageCount=3、有效 token usage 复现，
期待 $0.60，实际 $0.20。临时诊断测试已移除，生产实现未在本次审查中改动。

另一个需核对的端到端风险：Image 2.5 缺失 usage 返回 ErrModelPricingUnavailable，
RecordUsage 仍会将这类错误转成零成本告警记录。现有局部测试只断言计算器报错，
不能证明完整扣费链不会零扣。应恢复稳定按图默认行为，或完善明确启用的 token 计费链再发布。

## 已进入主分支的增量

| 类别 | 更新与作用 | 主要风险/边界 |
| --- | --- | --- |
| 会员 | 用户规则移除内部返利说明，展示首充折扣及范围/上限/时间 | 展示变更，不重算订单 |
| 控制台 | 有效订阅分组可见、用量筛选加载全部 Key、订阅跳转用户用量、失败刷新保留选择、停用分组可移除、账号到期快捷日期、关闭注册隐藏入口、安全 Markdown 充值帮助 | 权限和查询/前端回归 |
| Claude | CLI 版本覆盖与自愈下限；max_tokens=1 探测适用任意模型；消息缓存 JSON 转义 | 请求识别/上游兼容行为变化 |
| OpenAI 稳定性 | 429 未耗尽不按 reset-after 长停调、禁用回退冷却一致性、长流 HTTP/2 profile、结束时 cancel-before-close | 调度/连接生命周期变化，不保证所有断流有最终 usage |
| WebSocket | HTTP bridge 状态隔离、会话抢占实现 | 不等于后续轮次额度耗尽恢复 |
| 代理 | 部分更新保留省略字段、多主代理共享备用、重复到期回退 | 新数据库约束迁移、实际出口变化 |
| MiniMax | 平台与模型配置/网关/监控接线，remains 查询、5h/weekly 窗口与阈值 | 需要真实账号验收；不是整包等价认证 |
| Ollama | DeepSeek 输出上限、Messages URL 规范化、Anthropic Bearer、Chat reasoning 兼容 | 仅匹配适用端点/模型 |
| 缓存 | Redis 渠道失效广播；go-redis 9.22.0/context 适配 | 多实例失效传播与依赖升级 |
| 图片 | flare/sunburst 识别与测试器、本地目录/兜底价格、主控 gpt-5.6-luna、图输入用量 | 默认计费行为变化，上述多图问题阻断发布 |
| Gemini | 3.7/3.8 Flash thinking tier 优先精确匹配、再归一基础型号 | 价格匹配影响金额，非独立档位定价体系 |
| 目录 | 模型路由约束 Codex 元数据、Astra instructions/Ultra 元数据、广场区间倍率价格展示 | 仅部分官方适配，见缺口 |
| 成本 | OpenAI 周成本估算显示 | 估算不是上游真实账单 |
| 日志 | 普通 http.access 默认不复制到 DB；warn/error 与审计保留；系统日志独立保留期和设置热生效 | 普通访问历史减少；保留期届满会删除日志，不是余额/订单数据 |
| UI | Token 浮层视口边界；充值账户置顶、金额/支付双栏；支付方式逐行且长名称换行 | 共用组件影响经典/现代；局部组件和浏览器截图已验收 |
| 迁移 | 242 MiniMax 平台约束、243 共享备用代理、244 models_list_config 自愈 | 升级必须备份，旧程序回滚不等于数据库回滚 |

## 官方功能缺口（对照本地官方 v0.2.3/v0.2.4 及其前置提交）

| 项目 | 当前判断 |
| --- | --- |
| b8ed24508 WS 后续轮次额度耗尽恢复 | 未完成：relay 仍使用跨轮 wroteDownstream，缺逐轮复位与后续轮重连分支 |
| DeepSeek 峰谷用户扣费及账号成本统一 | 未完整合入；当前主分支无对应时段价格组件，不能宣称成本统计已经完整 |
| ff758f37d 模型广场 1h 缓存/区间回退 | 部分：前端区间倍率接入；后端字段及 1h 全链仍需补齐核对 |
| 0aaed397c Astra reasoning.mode | 未按官方此补丁适配；instructions/Ultra 元数据不能代表这项完成 |
| 2e31d8b70 allowed_tools 约束 | 官方工具别名/选择约束层尚未完整适配 |
| 95023e7d4 备份与迁移 advisory lock | 目标备份实现未带入该补丁，应单独核对 |
| 7a70de401 支付履约与兑换限制隔离 | 未带入此官方提交；AIPC 独立支付/营销链需按语义审查，不能直接宣布等价 |
| 76efc52fb 自定义页打开链接按钮拖动 | 未适配，可选 UI 功能 |
| 185951957 Apple 容器子网 | 未适配，仅对应部署方式有收益 |

以上是源码确认的重要缺口，不是官方全部提交的等价认证，也没有访问更新于本地 tag 的官方发布。
保留 AIPC 管理员手动获取/激活远端价格的机制，不存在本轮整体替换官方价格表的已完成事项。

## 验证范围

- 本轮局部 Go config/service 运维测试通过。
- UsageTable、PaymentMethodSelector、OpsSystemLogTable 共 37 项组件测试通过，已加入 GitHub critical suite。
- 本地前端构建被子进程 PATH 找不到 vue-tsc 阻断，远端标准环境构建成功与否以 Actions 为准。
- 完整 CI：https://github.com/HeartEase1/aipc/actions/runs/34850427857
- 安全扫描：https://github.com/HeartEase1/aipc/actions/runs/34850427875
- 上述 Image 2.5 多图诊断测试失败是独立审查证据，不因 CI 变绿而消失。
