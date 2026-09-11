# HarmonyOS 原生离线推送

`webhook` 的 `HARMONYOS` 通道对接客户端 `wkpush`：客户端通过原生 `pushService.getToken()` 取得 Token，使用现有 `POST /v1/user/device_token` 上报 `device_type: HARMONYOS`、`device_token` 和 `bundle_id`。`bundle_id` 必须与应用实际包名一致，下文使用通用包名作为示例。

服务端沿用 `msg.offline` 的通知开关、免打扰过滤、内容摘要和 Redis 未读数维护，通过 `HARMONYOS + bundle_id` 选择发送器。Android 华为设备继续使用 `HMS`。

## 华为官方文档

本实现按 HarmonyOS 原生 Push Kit 的服务端协议接入：

- [基于服务账号生成鉴权令牌](https://developer.huawei.com/consumer/cn/doc/doccenter-capabilities/push-jwt-token)：PS256 JWT，`kid = key_id`、`iss = sub_account`，固定 `aud` 为 `https://oauth-login.cloud.huawei.com/oauth2/v3/token`，有效期一小时。JWT 直接作为 Bearer 鉴权信息使用。
- [发送通知消息](https://developer.huawei.com/consumer/cn/doc/harmonyos-guides/push-send-alert)和[场景化消息请求参数](https://developer.huawei.com/consumer/cn/doc/harmonyos-references/push-scenariozed-api-request-param)：V3 接口、通知载荷和点击行为。
- [场景化消息响应参数](https://developer.huawei.com/consumer/cn/doc/harmonyos-references/push-scenariozed-api-response)：检查 HTTP 状态及业务响应码。
- [角标刷新消息](https://developer.huawei.com/consumer/en/doc/harmonyos-guides/push-send-badge)：`setNum` 接口范围为 0–99。本实现将角标随普通通知发送，没有接入独立的角标刷新消息。

服务端请求为 `POST https://push-api.cloud.huawei.com/v3/{project_id}/messages:send`，请求头包含 `push-type: 0`。载荷使用 `payload.notification`、`target.token` 和 `pushOptions`，不使用 Android 的 `message.android` 或 V1 appID 接口。

## 配置

1. 在华为平台为 HarmonyOS 应用开通 Push Kit，核对应用包名、签名和所属项目。
2. 在 API Console 下载推送服务的服务账号密钥 JSON，将它放在服务器可读的私密目录。需要包含 `project_id`、`key_id`、`private_key`、`sub_account`。`project_id` 必须对应客户端应用所属的项目。
3. 在**实际启动时 `-config` 指定的 YAML** 中合并以下配置，并重启服务：

```yaml
push:
  harmonyos:
    bundleID: "com.example.app"
    serviceAccountFile: "/run/secrets/harmonyos-push.json"
    category: ""      # 开通 IM 通知自分类权益后填写 IM
    testMessage: true # 调试应用；正式推送改为 false
```

`bundleID` 为空时不启用此通道；启用后，缺失或无效的服务账号文件会使 webhook 启动报错。私钥路径相对于服务进程工作目录解析，推荐使用绝对路径。更新密钥文件后需重启服务。

支持以下环境变量覆盖同名 YAML 配置：

```text
TS_PUSH_HARMONYOS_BUNDLEID
TS_PUSH_HARMONYOS_SERVICEACCOUNTFILE
TS_PUSH_HARMONYOS_CATEGORY
TS_PUSH_HARMONYOS_TESTMESSAGE
```

配置类型统一定义在基础库 `config/config.go` 的 `HarmonyOSPush`，通过 `Config.Push.HARMONYOS` 访问，由 `ConfigureWithViper()` 加载。webhook 直接读取同一个 Config，不再重新读取配置文件；YAML、环境变量和 Viper `Set()` 沿用基础库的统一优先级。包名 `BundleID` 默认留空，实际值仍由部署配置提供，不在 Go 源码中硬编码。

业务模块的 `go.mod` 已依赖包含 HarmonyOS 配置字段的基础库版本，支持独立构建。旗舰版通过本地 replace 引用这两个库时，也需要同步更新基础库代码。旗舰版运行时应更新旗舰版主工程实际使用的配置文件；仅修改本子模块的 `configs/tsdd.yaml` 模板不会改变旗舰版运行配置。不要提交服务账号 JSON。客户端仍需在 `entry` 的 `module.metadata` 配置真实应用 OAuth Client ID；客户端 Client ID 与服务端 `project_id` 是不同的标识，服务端不使用 Android 的 App Secret。

## 通知行为

- 标题、正文复用 `ParsePushInfo()`，保留项目原有内容详情配置。
- `clickAction.actionType = 0` 打开应用首页，沿用 `EntryAbility` 和登录/首页分流；没有新增直达聊天协议。
- `badge.setNum` 使用服务端维护的未读数，发送时限制为 0–99；Redis 及客户端上报的真实数量保持不变，因此超过 99 的离线通知角标按 99 发送。
- `category` 非空才下发。开通对应通知自分类权益后配置 `IM`；留空时由华为分类和限频，不能视为已获得 IM 通知权益。
- `testMessage` 独立于服务端 debug/release 模式，正式推送必须设为 `false`。测试推送受华为配额限制。
- JWT 在发送器实例内加锁缓存，提前一分钟刷新；HTTP 401 会使该缓存失效。HTTP 超时或错误不会自动重发当前通知，避免在送达结果不确定时重复提醒。
- 只有 HTTP 200 且业务码 `80000000` 才视为云端受理成功。业务失败、缺少 code、非 JSON 等均返回错误，不记录原始响应和 Token。
- 当前客户端未接入 VoIP/后台消息消费，本发送器拒绝带 `RTCPayload` 的通话控制载荷；此通道只提供普通离线通知。

## 验证

无需真实凭据、MySQL、Redis或华为网络的回归命令：

```bash
go test -race ./modules/webhook github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config -run '^TestHarmonyOS' -count=1
```

测试使用临时生成的 RSA 密钥和内存 HTTP Transport，覆盖签名校验、JWT 并发复用/刷新、V3 请求、角标边界、HTTP/业务失败、无效响应、配置覆盖及通道注册。此命令不会执行该目录原有的厂商联网测试。

这些测试只能验证本地实现和协议组装，不能证明真机送达。配置真实服务账号与客户端 Client ID 后，还需分别验证后台、锁屏、进程终止情况下的通知展示、角标、点击启动及免打扰过滤。

设备绑定沿用现有“每个 UID 一条设备 Token”的存储模型，没有新增多设备绑定或跨账号反向解绑。客户端 README 中要求的退出/失效及换号后的旧绑定清理，仍需结合现有账号生命周期单独验收；本次通道接入不能证明这些场景已完整解决。
