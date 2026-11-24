# Sub2API v0.2.25 同步适配

同步基线：AIPC `v1.0.94` / Sub2API `v0.2.21`，合并官方 `v0.2.22..v0.2.25` 累计更新。本次只同步代码，不提升 AIPC 版本、不创建发布标签。

## 主要上游更新

- API Key 上游请求头模板支持网关身份变量；账号可以同步上游分组名称和描述。
- OpenAI 透传错误处理、compact 流式错误转换、请求模型重复字段校验等安全与兼容修复。
- GPT-6.2、Ultra Fast 服务档位与渠道倍率、Vertex 多区域等模型和上游能力更新。
- 缓存用量与 TTL 处理、异步用量结算、分组用量汇总和连接复用等稳定性及性能改进。
- Gemini、Anthropic、Grok 等协议桥接及模型能力同步更新。

## AIPC 适配

- 保留充值赠送、折扣、会员、邀请返利、长上下文豁免及 token 定价快照；获赠订单的账务路径仍按 AIPC 规则处理。
- 保留 OpenAI OAuth 与 API Key 账号的“仅允许 Codex 官方客户端”能力、API Key 透传模型筛选及 Fast Mode。
- API Key 身份模板作用域同时记录 AIPC Fast Mode 和上游用户、Key、分组标识。
- 保留 AIPC 支付宝退款请求 ID 幂等逻辑；没有请求 ID 时使用稳定的订单及金额标识，保证重试不会生成新退款号。
- Bedrock 连通性测试沿用上游兼容 payload，同时保留 AIPC Pelican 专用测试请求。
- 保留 AIPC 手动价格目录策略；上游新增的模型价格数据可在 AIPC 自有价格配置上继续调整。
- Ent 与 Wire 依据合并后的源定义重新生成；保留 AIPC 管理页面、在线工作台和发布流程。

## 校验

- `go generate ./ent`：通过。
- `go generate ./cmd/server`：通过。
- GitHub CI 与安全扫描：待推送后验证。
