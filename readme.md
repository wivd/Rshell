# Rshell - 跨平台多协议C2框架

Rshell是一款开源的golang编写的支持多平台的C2框架，旨在帮助安服人员渗透测试、红蓝对抗。功能介绍如下：

## 基础使用

下载Rshell二进制文件并运行。

通过-p参数指定端口（默认端口8089）并运行：

```bash
./Rshell -p 8089
```

## 账号密码

**账号:admin**

**密码:首次运行后随机生成**

![image-20260420125544426](./assets/image-20260420125544426.png)


![image-20260205112117018](./assets/image-20260205112117018.png)

![image-20260205112151336](./assets/image-20260205112151336.png)

## 主题修改

### 修改主题颜色

![image-20260205112303486](./assets/image-20260205112303486.png)

### 添加背景图片

![image-20260205112328685](./assets/image-20260205112328685.png)

![image-20260205112423810](./assets/image-20260205112423810.png)



## 添加listener

目前支持websocket、tcp、kcp、http、oss协议监听：

![image-20260205112214388](./assets/image-20260205112214388.png)

![image-20260205112507887](./assets/image-20260205112507887.png)

## 生成客户端

支持windows、linux、darwin

**注：客户端可选配置反沙箱密码上线  如： r.exe tNROopcR45q4Z8I1**

![image-20260206163631937](./assets/image-20260206163631937.png)

## Webdelivery

![image-20260206163702513](./assets/image-20260206163702513.png)

![image-20260206163718260](./assets/image-20260206163718260.png)

## 客户端管理

### 支持Note、颜色标记

![image-20260206163822729](./assets/image-20260206163822729.png)

### 命令执行

shell + [cmd]

![image-20260206163858795](./assets/image-20260206163858795.png)

### 交互式终端

建议在websocket、tcp、kcp的长连接协议下使用，http协议下可以将sleep时间设置为0，以获得低延迟体验。

![image-20260206163932532](./assets/image-20260206163932532.png)

![image-20260206164007016](./assets/image-20260206164007016.png)

### 文件管理

双击进入文件夹：

![image-20260206164037005](./assets/image-20260206164037005.png)

### PID查看

![image-20260206164855120](./assets/image-20260206164855120.png)

杀毒软件识别

![image-20260206170905006](./assets/image-20260206170905006.png)

### 文件下载

![image-20260206165840634](./assets/image-20260206165840634.png)

### 笔记

![image-20260206170443496](./assets/image-20260206170443496.png)

## Windows相关

### shellcode生成：

生成步骤：

（1）新建监听；
（2）建立对应监听的windows版的webdelivery；
（3）在对应的webdelivery的右侧，有shellcode生成的选项。

新增windows的webdelivery后，可以生成stage分阶段的shellcode（体积较小，方便上线）：

![image-20260206171033082](./assets/image-20260206171033082.png)

![image-20260206171516567](./assets/image-20260206171516567.png)

### 内存执行

windows内存执行 支持Execute Assembly(.net程序内存执行)、Inline Bin(其他exe程序内存执行)、shellcode执行(执行shellcode,方便上线其他C2等)、Inline Execute(执行BOF)：

![image-20260206171640831](./assets/image-20260206171640831.png)

#### execute-assembly

执行badpotato提权：

![image-20260206171836364](./assets/image-20260206171836364.png)

![image-20260206171903611](./assets/image-20260206171903611.png)

#### inline-bin

内存执行fscan：

![image-20260206172005126](./assets/image-20260206172005126.png)

#### shellcode-inject

上线msf：

![image-20260206172115518](./assets/image-20260206172115518.png)

![image-20251215172633049](./assets/image-20251215172633049.png)

#### inline-execute

执行bof：

![image-20260206172303469](./assets/image-20260206172303469.png)

![image-20260206173424252](./assets/image-20260206173424252.png)

## 插件管理

新增插件：

![image-20260423102325331](./assets/image-20260423102325331.png)

调用插件：

![image-20260423102406659](./assets/image-20260423102406659.png)

# ToDoList

**目前有一些待实现的改进想法，将不定期更新。如果你有好的建议或想参与开发，欢迎提交 PR 或开 issue 讨论。**

- [ ] 一键退出所有终端
- [ ] 文件下载增加中断功能  [#21](https://github.com/Rubby2001/Rshell---A-Cross-Platform-C2/issues/21)
- [ ] 截图功能  [#9](https://github.com/Rubby2001/Rshell---A-Cross-Platform-C2/issues/9)
- [x] linux内存执行
- [x] web托管bash、bat脚本  [#24](https://github.com/Rubby2001/Rshell---A-Cross-Platform-C2/issues/24)  (合并到内存执行)
- [x] 增加MCP功能
- [ ] 笔记功能增加图床
- [ ] 笔记md格式所见即所得
- [ ] 一键信息收集，收集信息内容直接保存到笔记中
- [ ] 丰富上线提醒的方式（邮箱、钉钉、telegram等）
- [x] 增加正向上线方式 [#26](https://github.com/Rubby2001/Rshell---A-Cross-Platform-C2/issues/26)
- [x] 增加插件模块  [#9](https://github.com/Rubby2001/Rshell---A-Cross-Platform-C2/issues/9)
- [x] 增加开机启动等插件是否合适  [#9](https://github.com/Rubby2001/Rshell---A-Cross-Platform-C2/issues/9) (可借助插件模块实现)
- [x] 通讯流量实现自定义配置或随机密钥，增加逆向解密难度

# 相关项目

客户端开源地址：https://github.com/Rubby2001/Rshell-client

前端开源地址：https://github.com/Rubby2001/Rshell-web

# Star Trending

[![Star History Chart](https://api.star-history.com/svg?repos=Rubby2001/Rshell---A-Cross-Platform-C2&type=Date)](https://star-history.com/#Rubby2001/Rshell---A-Cross-Platform-C2&Date)

# 免责声明

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

