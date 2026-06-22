# Rshell - 跨平台多协议C2框架

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

Rshell是一款开源的golang编写的支持多平台的C2框架，旨在帮助安服人员渗透测试、红蓝对抗。功能介绍如下：

## 核心特性

- **跨平台支持**：支持 Windows、Linux、Darwin 等系统的客户端。
- **多协议支持**：支持 Websocket、TCP、KCP、HTTP、OSS 等多种协议监听。
- **免杀与隐蔽**：客户端可选配置反沙箱密码上线，支持内存执行（Execute Assembly, Inline Bin, Shellcode Inject, BOF）。
- **交互式管理**：提供交互式终端、文件管理、PID查看及杀软识别、命令执行等功能。
- **模块化插件**：内置丰富的插件管理功能，支持动态加载执行。
- **数据解耦**：通讯流量实现自定义配置或随机密钥，增加逆向解密难度。

## 基础使用

下载Rshell二进制文件并运行。

通过-p参数指定端口（默认端口8089）并运行：

```bash
./Rshell -p 8089
```

![image-20260429142230230](./assets/image-20260429142230230.png)

## 详细配置与使用文档

**具体的功能介绍（包含大量操作截图和细节），请查阅：**

👉 **[Rshell 详细使用文档](./docs/USAGE.md)**

了解了如下内容：
- 账号密码修改及主题自定义
- 添加多种协议监听 (Listener) 与客户端生成
- Webdelivery 设置
- 客户端管理（终端、文件、进程管理等操作）
- Windows专项高级操作（shellcode免杀生成与各类内存执行）
- 插件的使用与管理

## ToDoList

**目前有一些待实现的改进想法，将不定期更新。如果你有好的建议或想参与开发，欢迎提交 PR 或开 issue 讨论。**

- [x] 一键退出所有终端
- [ ] 文件下载增加中断功能  [#21](https://github.com/Rubby2001/Rshell---A-Cross-Platform-C2/issues/21)
- [x] 截图功能  [#9](https://github.com/Rubby2001/Rshell---A-Cross-Platform-C2/issues/9)
- [x] linux内存执行
- [x] web托管bash、bat脚本  [#24](https://github.com/Rubby2001/Rshell---A-Cross-Platform-C2/issues/24)  (合并到内存执行)
- [ ] 笔记功能增加图床
- [ ] 笔记md格式所见即所得
- [ ] 一键信息收集，收集信息内容直接保存到笔记中
- [x] 丰富上线提醒的方式（邮箱、钉钉、telegram等）
- [x] 增加正向上线方式 [#26](https://github.com/Rubby2001/Rshell---A-Cross-Platform-C2/issues/26)
- [x] 增加插件模块  [#9](https://github.com/Rubby2001/Rshell---A-Cross-Platform-C2/issues/9)
- [x] 增加开机启动等插件是否合适  [#9](https://github.com/Rubby2001/Rshell---A-Cross-Platform-C2/issues/9) (可借助插件模块实现)
- [x] 通讯流量实现自定义配置或随机密钥，增加逆向解密难度

## 相关项目

客户端开源地址：https://github.com/Rubby2001/Rshell-client

前端开源地址：https://github.com/Rubby2001/Rshell-web

## Star Trending

[![Star History Chart](https://api.star-history.com/svg?repos=Rubby2001/Rshell---A-Cross-Platform-C2&type=Date)](https://star-history.com/#Rubby2001/Rshell---A-Cross-Platform-C2&Date)

## 免责声明

1. 本项目仅为网络安全研究、合法授权测试及教育目的而设计开发，旨在帮助安全专业人员提升防御能力、测试系统安全性。
2. **禁止将本项目用于任何非法用途**，包括但不限于：
   - 未经授权的系统入侵
   - 网络攻击活动
   - 任何违反《中华人民共和国网络安全法》《刑法》等法律法规的行为
3. 使用者应确保在**完全合法授权**的前提下使用本工具，开发者不对任何滥用行为负责。
4. 本工具提供的功能可能对目标系统造成影响，使用者需自行承担所有风险，确保：
   - 已获得目标系统的明确授权
   - 遵守当地法律法规
   - 不会危害关键信息基础设施
5. 开发者不承诺工具的隐蔽性、稳定性或适用性，不承担因使用本工具导致的任何直接或间接责任。
6. 下载、使用本项目即表示您已充分阅读并同意本声明所有条款。
