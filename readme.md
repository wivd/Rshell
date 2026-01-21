# Rshell - 跨平台多协议C2框架

Rshell是一款开源的golang编写的支持多平台的C2框架，旨在帮助安服人员渗透测试、红蓝对抗。功能介绍如下：

## 基础使用

下载Rshell二进制文件并运行。

通过-p参数指定端口（默认端口8089）并运行：

```
./Rshell -p 8089
```

**默认账号密码：admin/admin123**

![image-20251215170249521](./assets/image-20251215170249521.png)

![image-20251215170310085](./assets/image-20251215170310085.png)

## 添加listener

目前支持websocket、tcp、kcp、http、oss协议监听：

![image-20251215170333265](./assets/image-20251215170333265.png)

![image-20251215170345935](./assets/image-20251215170345935.png)

## 生成客户端

支持windows、linux、darwin

**注：客户端可选配置反沙箱密码上线  如： r.exe tNROopcR45q4Z8I1**

![image-20251215170444480](./assets/image-20251215170444480.png)

## Webdelivery

![image-20251215170535367](./assets/image-20251215170535367.png)

![image-20251215170616267](./assets/image-20251215170616267.png)

## 客户端管理

### 支持Note、颜色标记

![image-20251215170636840](./assets/image-20251215170636840.png)

### 命令执行

![image-20251215170735495](./assets/image-20251215170735495.png)

### 交互式终端

建议在websocket、tcp、kcp的长连接协议下使用，http协议下可以将sleep时间设置为0，以获得低延迟体验。

![image-20260113163653945](./assets/image-20260113163653945.png)

![image-20260113163726928](./assets/image-20260113163726928.png)

### 文件管理

双击进入文件夹：

![image-20251215170813937](./assets/image-20251215170813937.png)

### PID查看

![image-20251215171013713](./assets/image-20251215171013713.png)

杀毒软件识别

![image-20251215171028106](./assets/image-20251215171028106.png)

### 文件下载

![image-20251215171050625](./assets/image-20251215171050625.png)

### 笔记

![image-20251215171126935](./assets/image-20251215171126935.png)

## Windows相关

### shellcode生成：

生成步骤：

（1）新建监听；
（2）建立对应监听的windows版的webdelivery；
（3）在对应的webdelivery的右侧，有shellcode生成的选项。

新增windows的webdelivery后，可以生成stage分阶段的shellcode（体积较小，方便上线）：

![image-20251215171205269](./assets/image-20251215171205269.png)

![image-20251215171222896](./assets/image-20251215171222896.png)

### 内存执行

windows内存执行 支持Execute Assembly(.net程序内存执行)、Inline Bin(其他exe程序内存执行)、shellcode执行(执行shellcode,方便上线其他C2等)、Inline Execute(执行BOF)：

![image-20251215171403903](./assets/image-20251215171403903.png)

#### execute-assembly

执行badpotato提权：

![image-20251215171526330](./assets/image-20251215171526330.png)

![image-20251216154821723](./assets/image-20251216154821723.png)

#### inline-bin

内存执行fscan：

![image-20251215172334870](./assets/image-20251215172334870.png)

#### shellcode-inject

上线msf：

![image-20251215172443880](./assets/image-20251215172443880.png)

![image-20251215172633049](./assets/image-20251215172633049.png)

#### inline-execute

执行bof：

![image-20251215172730716](./assets/image-20251215172730716.png)

![image-20251215172848925](./assets/image-20251215172848925.png)

# ToDoList

**目前有一些待实现的改进想法，将不定期更新。如果你有好的建议或想参与开发，欢迎提交 PR 或开 issue 讨论。**

- [ ] 一键退出所有终端
- [ ] 文件下载增加中断功能  #21
- [ ] 截图功能  #9
- [ ] linux内存执行
- [ ] web托管bash、bat脚本  #24
- [ ] 笔记功能增加图床
- [ ] 笔记md格式所见即所得
- [ ] 一键信息收集，收集信息内容直接保存到笔记中
- [ ] 丰富上线提醒的方式
- [ ] 增加插件模块  #9
- [ ] 增加开机启动等插件是否合适  #9

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

