## 唐僧叨叨

<p align="center">
<img align="center" width="150px" src="./docs/logo.svg">
</p>

<p align="center">
<!-- 开源社区第二屌(🦅)的即时通讯软件 -->
</p>

<p align="center">
    几个老工匠，历时<a href="#">八年</a>时间打造的<a href="#">运营级别</a>的开源即时通讯聊天软件(<a href='https://github.com/WuKongIM/WuKongIM'>开源WuKongIM</a>提供通讯动力)
</p>
<div align=center>

<!-- [![Go](https://github.com/TangSengDaoDao/TangSengDaoDaoServer/workflows/Go/badge.svg?branch=main)](https://github.com/TangSengDaoDao/TangSengDaoDaoServer/actions) -->
![GitHub go.mod Go version (subdirectory of monorepo)](https://img.shields.io/github/go-mod/go-version/TangSengDaoDao/TangSengDaoDaoServer)
[![Go Report Card](https://goreportcard.com/badge/github.com/TangSengDaoDao/TangSengDaoDaoServer)](https://goreportcard.com/report/github.com/TangSengDaoDao/TangSengDaoDaoServer)
![License: Apache 2.0](https://img.shields.io/github/license/WuKongIM/WuKongIM) 

<!-- [![Release](https://img.shields.io/github/v/release/TangSengDaoDao/TangSengDaoDaoServer.svg?style=flat-square)](https://github.com/TangSengDaoDao/TangSengDaoDaoServer) -->

</div>

`开发环境需要go >=1.20`

愿景
------------

让企业轻松拥有自己的即时通讯软件。

架构图
------------

![架构图](./docs/architecture.png)

轻松上手
------------

安装脚本

```shell

curl -sSL https://gitee.com/TangSengDaoDao/TangSengDaoDaoCli/raw/main/install.sh | sudo bash

```

安装唐僧叨叨

```shell
tsdd install --ip xx.xx.xx.xx
```

`xx.xx.xx.xx为服务器IP地址（外网地址）`


开始唐僧叨叨

```shell
tsdd start
```

更多部署方式参考文档：https://tangsengdaodao.com/dev/backend/deploy-compose.html


客户端源码
------------

Web/PC端源码： https://github.com/TangSengDaoDao/TangSengDaoDaoWeb

Android端源码： https://github.com/TangSengDaoDao/TangSengDaoDaoAndroid

iOS端源码：     https://github.com/TangSengDaoDao/TangSengDaoDaoiOS

技术文档
------------

https://tangsengdaodao.com



功能特性
------------
- [x] 全局特性
    - [x] 消息永久存储
    - [x] 消息加密传输
    - [x] 消息多端同步(app,web,pc等)
    - [x] 群聊人数无限制
    - [x] 机器人
- [x] 消息列表
    - [x] 单聊
    - [x] 群聊
    - [x] 发起群聊
    - [x] 添加朋友
    - [x] 扫一扫
    - [x] 列表提醒项，比如消息@提醒，待办提醒，服务器可控
    - [x] 置顶
    - [x] 消息免打扰
    - [x] web登录状态显示
    - [x] 消息搜索
    - [x] 消息输入中
    - [x] 消息未读数
    - [x] 用户标识
    - [x] 无网提示
    - [x] 草稿提醒
- [x] 消息详情
    - [x] 文本消息
    - [x] 图片消息
    - [x] 语音消息
    - [x] Gif消息
    - [x] 合并转发消息
    - [x] 正在输入消息
    - [x] 自定义消息
    - [x] 撤回消息
    - [x] 群系统消息
    - [x] 群@消息
    - [x] 消息回复
    - [x] 消息转发
    - [x] 消息收藏
    - [x] 消息删除
- [x] 群功能
    - [x] 添加群成员/移除群成员
    - [x] 群成员列表
    - [x] 群名称
    - [x] 群二维码
    - [x] 群公告
    - [x] 保存到通讯录
    - [x] 我在本群昵称
    - [x] 群投诉    
    - [x] 清空群聊天记录    
- [x] 好友
    - [x] 备注
    - [x] 拉黑
    - [x] 投诉
    - [x] 添加/解除好友
- [x] 通讯录
    - [x] 新的朋友
    - [x] 保存的群
    - [x] 联系人列表
- [x] 我的
    - [x] 个人信息
    - [x] 新消息通知设置
    - [x] 安全与隐私
    - [x] 通用设置
    - [x] 聊天背景
    - [x] 多语言
    - [x] 黑暗模式
    - [x] 设备管理



动画演示
------------

||||
|:---:|:---:|:--:|
|![](./docs/screenshot/conversationlist.webp)|![](./docs/screenshot/messages.webp)|![](./docs/screenshot/robot.webp)|


|||          |
|:---:|:---:|:-------------------:|
|![](./docs/screenshot/weblogin.webp)|![](./docs/screenshot/apm.webp)| ![](./docs/screenshot/others.webp) |

![](docs/screenshot/pc2.png)

![](docs/screenshot/pc1.png)


演示地址
------------

| Android扫描体验 | iOS扫描体验(商店版本 apple store 搜“唐僧叨叨”) |
|:---:|:---:|
|![](docs/download/android.png)|![](docs/download/iOS.png)|

| Web端 | Windows端 | MAC端 | Ubuntun端 |
|:---:|:---:|:---:|:---:|
|[点击体验](https://web.botgate.cn)|[点击下载](https://github.com/TangSengDaoDao/TangSengDaoDaoWeb/releases/download/v1.0.0/tangsegndaodao_1.0.0_x64_zh-CN.msi)|[点击下载](https://github.com/TangSengDaoDao/TangSengDaoDaoWeb/releases/download/v1.0.0/tangsegndaodao_1.0.0_x64.dmg)|[点击下载](https://github.com/TangSengDaoDao/TangSengDaoDaoWeb/releases/download/v1.0.0/tangsegndaodao_1.0.0_amd64.deb)|


Star
------------

我们团队一直致力于即时通讯的研发，需要您的鼓励，如果您觉得本项目对您有帮助，欢迎点个star，您的支持是我们最大的动力。

加入群聊
------------

微信：加群请备注“唐僧叨叨”

<img src="docs/tsddwx.png" width="200px" height="200px">

许可证
------------

唐僧叨叨 使用 Apache 2.0 许可证。有关详情，请参阅 LICENSE 文件。


