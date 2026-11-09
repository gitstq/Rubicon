<div align="center">

# 🛡️ Rubicon

**AI Agent 决策卡与会话交接协议 · Decision Cards & Session Handoff for AI Agents**

> **让 Agent 自己跑，让人类只对关键的那一下拍板。**
> Let the agent run; ask a human only for the decisions that actually matter.

[![CI](https://img.shields.io/github/actions/workflow/status/gitstq/Rubicon/ci.yml?branch=main&label=CI)](https://github.com/gitstq/Rubicon/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go)](https://go.dev/)
[![Zero dependencies](https://img.shields.io/badge/dependencies-0-brightgreen)](#2--%E6%A0%B8%E5%BF%83%E7%89%B9%E6%80%A7)
[![Platforms](https://img.shields.io/badge/platforms-6-blue)](#6--%E6%89%93%E5%8C%85%E4%B8%8E%E9%83%A8%E7%BD%B2%E6%8C%87%E5%8D%97)
[![Release](https://img.shields.io/badge/release-v1.0.0-orange)](https://github.com/gitstq/Rubicon/releases/tag/v1.0.0)
[![License: MIT](https://img.shields.io/badge/license-MIT-yellow)](LICENSE)

### ⬇️ [下载 v1.0.0 全平台可执行文件](https://github.com/gitstq/Rubicon/releases/tag/v1.0.0)

**Windows · macOS · Linux｜amd64 + arm64｜单文件、免安装、零运行时依赖**

</div>

---

🌐 **选择语言**：[简体中文](#zh-cn)　·　[繁體中文](#zh-tw)　·　[English](#en)

---

<a id="zh-cn"></a>

## 简体中文

### 1. 🎉 项目介绍

**Rubicon 是一个离线优先、零依赖的决策卡与会话交接工具链**，专门解决 AI 编码 Agent 在长任务里最尴尬的两个瞬间：

- 🤖 **该不该动手？** Agent 跑到「删掉旧迁移脚本」「把发布标签打上去」这类不可逆操作时，不该自己猜，也不该把整段对话日志甩给人类。
- 🔄 **换个会话就失忆？** 上一个 Agent 会话做了什么、还有哪些问题悬而未决、下一个会话该从哪一步接上，往往只存在聊天记录里，一断就丢。

Rubicon 把这两个瞬间收敛成同一种结构化对象：**决策卡（Decision Card）**。Agent 遇到不确定事项时生成一张卡，人类用一条命令批准或否决；会话结束时生成一份**交接包（Handoff）**，只带走未决卡、已完成步骤和变更文件，不带走聊天记录。

**灵感来源**：近期 GitHub 热议的 `agent-autoloop`（后台自动接活、遇事才找人拍板）、`claude-mem`（跨会话持久上下文）与 `spec-kit`（规格驱动开发）共同指向同一个缺口——**Agent 流程里缺少一个可验证的人工决策点**。本项目仅参考其产品逻辑与所解决的痛点，**全部代码 100% 独立自研，未复制任何一行原项目代码**。

**自研差异化亮点**：

| 维度 | 常见做法 | Rubicon |
| --- | --- | --- |
| 决策记录 | 写在聊天记录 / Markdown 备注里 | **只追加的哈希链账本**，改一个字都能被发现 |
| 状态来源 | 直接改 JSON 文件 | **事件溯源**，当前状态永远由重放账本推导 |
| 时间行为 | 读系统时钟，CI 里结果飘忽 | **`--now` 可注入**，同样输入必然同样输出 |
| 会话交接 | 导出整段对话 | **结构化交接包 + 摘要校验**，只留必要信息 |
| 交付形态 | Python 脚本 + 一堆依赖 | **Go 单静态二进制**，六平台开箱即用 |

> 📌 **一句话定位**：Rubicon 不是又一个代码审计器，它是 Agent 工作流里的「决策闸门 + 交接协议」。

---

### 2. ✨ 核心特性

- 🧾 **结构化决策卡**：标题、明确的问题、至少两个选项、**安全默认动作**、影响面、可回滚标记、截止时间与标签，字段齐全才能落账。
- 🔗 **只追加哈希链账本**：每条事件 `sha256(前序哈希 | 事件类型 | 时间 | 载荷)`，任何篡改、删行、插行都会被 `verify` 精确定位到事件序号。
- ♻️ **事件溯源状态机**：`pending → approved / rejected / expired / superseded`，状态不落盘，全部由重放推导，**永不原地修改**。
- ⏰ **确定性超时**：`--now` 固定时钟，`expire` 幂等；截止时间一过，Agent 不能再偷偷自己做决定。
- 📦 **可验证交接包**：`handoff build` 生成带 SHA-256 摘要的 JSON，`handoff resume` 先验摘要再输出续接简报，**改一个字就拒绝**。
- 🚦 **CI 门禁就绪**：`verify` 链断返回 3，`lint` 有错误级规则返回 3，可直接串进流水线；支持 `--format table|json|md` 三种输出。
- 🧩 **16 条内置规则**：从「选项少于两个」到「未声明可回滚性」，逐条给出规则编号、严重级与命中卡片。
- 🪶 **零第三方依赖**：仅 Go 标准库，无网络调用、无环境变量、无隐式状态，`go.mod` 里干干净净。
- 📱 **六平台原生二进制**：Linux / macOS / Windows × amd64 / arm64，`CGO_ENABLED=0` 静态编译，拷过去就能跑。

---

### 3. 🚀 快速开始

#### 环境要求

| 项目 | 要求 |
| --- | --- |
| 直接运行预编译包 | **无要求**，任意 Windows / macOS / Linux（amd64 或 arm64） |
| 从源码构建 | Go **1.21+**、`make`、`bash`（仅打包脚本需要 `tar` / `zip`） |
| 网络 | **完全不需要**，工具运行时不发任何请求 |
| 环境变量 | **不需要配置任何环境变量**，所有行为由命令行参数决定，避免隐式状态 |

#### 方式一：下载 Release（推荐，最快）

前往 👉 **[v1.0.0 Release 下载页](https://github.com/gitstq/Rubicon/releases/tag/v1.0.0)**，按平台取一个压缩包，解压后把 `rubicon`（Windows 为 `rubicon.exe`）放进 `PATH` 即可。用 `SHA256SUMS` 校验完整性：

```bash
# Linux / macOS
sha256sum -c SHA256SUMS --ignore-missing
./rubicon version
```

```powershell
# Windows PowerShell
Get-FileHash .\rubicon_1.0.0_windows_amd64.zip -Algorithm SHA256
.\rubicon.exe version
```

#### 方式二：从源码构建

```bash
git clone https://github.com/gitstq/Rubicon.git
cd Rubicon
make all          # gofmt + go vet + go test -race + 构建
./dist/rubicon version
```

#### 方式三：go install

```bash
go install github.com/gitstq/Rubicon/cmd/rubicon@latest
rubicon version
```

#### 前三分钟：跑通一个完整闭环

```bash
# 1️⃣ 初始化工作区（会在当前目录生成 .rubicon/ledger.jsonl）
rubicon init

# 2️⃣ Agent 遇到不可逆操作时，生成一张决策卡
rubicon card new \
  --title "删除旧版迁移脚本？" \
  --question "我可以删除 db/migrate/001.sql 吗？" \
  --options "A:删除|high,B:保留|low" \
  --default B \
  --files "db/migrate/001.sql" \
  --reversible false

# 3️⃣ 人类拍板
rubicon decide dc_0001 --choice A --by alice --note "评审已确认"

# 4️⃣ CI 里守两道门
rubicon verify && rubicon lint

# 5️⃣ 会话结束，生成交接包
rubicon handoff build --summary "重构第一阶段完成" --out handoff.json
rubicon handoff resume --file handoff.json
```

> 💡 想要一次看完全流程？执行 `make demo` 或 `bash examples/demo.sh`，演示固定了 `--now`，**每次输出完全一致**。

---

### 4. 📖 详细使用指南

#### 命令全表

| 命令 | 作用 |
| --- | --- |
| `rubicon init` | 创建 `.rubicon` 工作区 |
| `rubicon card new [flags]` | 生成一张待决卡 |
| `rubicon card list [--status STATUS]` | 列出卡片，可按状态过滤 |
| `rubicon card show <id>` | 查看单张卡完整 JSON |
| `rubicon decide <id> --choice K` | 批准某个选项 |
| `rubicon reject <id>` | 否决该决策 |
| `rubicon supersede <id>` | 标记为已被后续决策取代 |
| `rubicon expire` | 关闭所有超时未决卡（幂等） |
| `rubicon verify` | 校验哈希链完整性（CI 门禁） |
| `rubicon lint` | 按 16 条规则体检所有卡片（CI 门禁） |
| `rubicon report` | 输出全量工作区报告 |
| `rubicon handoff build [flags]` | 生成带摘要的交接包 |
| `rubicon handoff resume --file F` | 校验并输出续接简报 |
| `rubicon version` / `help` | 版本 / 帮助 |

#### 全局参数（可放在命令前后任意位置）

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--dir DIR` | `.rubicon` | 工作区目录，内含 `ledger.jsonl` |
| `--format F` | `table` | `table` / `json` / `md` 三种输出格式 |
| `--now RFC3339` | 系统 UTC 时间 | **注入固定时钟**，让 CI 与测试完全可复现 |

#### `card new` 参数说明

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `--id` | 否 | 卡片 ID，省略时自动生成 `dc_0001`、`dc_0002`… |
| `--title` | ✅ | 一句话标题 |
| `--question` | ✅ | 人类需要回答的确切问题 |
| `--options` | ✅ | `KEY:标签[|影响],KEY:标签[|影响]`，至少两项 |
| `--default` | ✅ | 超时/失联时 Agent 应采取的安全选项键 |
| `--context` | 否 | 给评审者的背景信息 |
| `--deadline` | 否 | RFC3339 截止时间，如 `2026-11-10T10:00:00Z` |
| `--files` / `--scope` | 否 | 受影响文件（逗号分隔）/ 影响范围描述 |
| `--reversible` | 否 | `true` / `false`；不填会被 `lint` 提示 |
| `--tags` | 否 | 逗号分隔标签 |

#### 输出格式与退出码

```bash
rubicon --format json card list --status pending   # 机器可读
rubicon --format md   report                     # 贴进 PR / 文档
rubicon              report                     # 终端表格
```

| 退出码 | 含义 |
| --- | --- |
| `0` | 成功 |
| `1` | 运行时错误（文件不可写、卡片不存在等） |
| `2` | 用法错误（缺参数、格式非法） |
| `3` | **CI 门禁失败**：链被篡改，或 `lint` 出现错误级规则 |

```yaml
# .github/workflows/gate.yml 片段
- name: Decision ledger gate
  run: |
    rubicon --dir .rubicon verify
    rubicon --dir .rubicon lint
```

#### 账本与配置教程

工作区只有一个文件：`.rubicon/ledger.jsonl`。**每行一个不可变事件**，形如：

```json
{"seq":1,"prev_hash":"0000...0000","hash":"9f2c...","kind":"card.created","ts":"2026-11-09T10:00:00Z","payload":{"id":"dc_0001", "...": "..."}}
```

- `payload` 是该时刻卡片的**完整快照**，所以重放账本即可还原任意时刻的状态。
- 无需任何配置文件；策略通过命令行参数表达。
- 账本建议随代码一起提交，或由 CI 作为产物归档；**不要手工编辑**，编辑会被 `verify` 抓到。
- 参考示例卡：[`examples/decision-card.json`](examples/decision-card.json)。

#### 典型场景

**场景 A · Agent  emits 决策卡**：Agent 在执行前把不确定项写成卡，检测到 `pending` 卡就暂停并通知人类；人类 `decide` 后 Agent 重放账本继续。

**场景 B · 无人值守超时**：给卡设 `--deadline`，人类没回，CI 定时跑 `rubicon expire`，卡自动落到 `expired`，Agent 按 `--default` 走安全路径，**绝不擅自执行高影响动作**。

**场景 C · 跨会话交接**：会话 A 结束前 `handoff build`，会话 B 一上来 `handoff resume`，立刻知道还有几张卡没拍板、哪些文件动过、下一步该干什么，而不用读几千行聊天记录。

**场景 D · 合规审计**：`verify` + `report --format md` 直接输出「谁在什么时候批准了什么」的证据链，链头哈希可贴进审计记录。

> 📷 **演示截图 / 动图占位**：`docs/screenshots/cli-demo.gif`（计划补充终端演示动图）、`docs/screenshots/report-md.png`（Markdown 报告渲染效果）。当前版本请运行 `make demo` 查看等价的终端实录。

---

### 5. 💡 设计思路与迭代规划

#### 设计理念

1. **账本即真相**：不存「当前状态」，只存「发生了什么」。状态是重放的结果，因此无法被偷偷改掉。
2. **可验证优于可信任**：哈希链与交接摘要让「这份记录没被动过」成为可计算的结论，而不是一句承诺。
3. **确定性是一等公民**：时钟可注入、无网络、无随机数、切片排序后计算摘要，同样的输入在任何平台给出同样的字节。
4. **最小必要信息**：交接包只带未决卡、步骤与文件清单，不搬运整段对话，避免上下文膨胀。
5. **人类只做关键决策**：卡必须至少两个选项、必须有安全默认动作，逼着 Agent 把问题问清楚。

#### 为什么选 Go

| 备选 | 结论 |
| --- | --- |
| Python | 生态好，但「免依赖可执行文件」需要 PyInstaller，且**无法在同一台 Linux 上交叉产出 macOS / Windows 原生包** |
| Rust | 性能好，但标准库外的依赖会打破「零依赖」约束，编译链也更重 |
| **Go** ✅ | 标准库足够（JSON / SHA-256 / flag / tabwriter 全覆盖），`CGO_ENABLED=0` 一行交叉编译出六平台静态二进制，**真正开箱即用** |

#### 迭代规划

- 🗓️ **v1.1**：`rubicon diff` 对比两份交接包，只展示增量；卡片优先级与依赖（卡 A 未决时阻塞卡 B）。
- 🗓️ **v1.2**：可选的本地 Web 审核面板（仍是单二进制、仍离线）。
- 🗓️ **v1.3**：MCP Server 模式，让 Agent 直接通过工具调用读写决策卡。
- 🗓️ **v2.0**：多评审人签名与法定审批链（保持零第三方依赖）。

#### 社区贡献方向

最欢迎的贡献：**新的 lint 规则**、**更多输出格式**、**与各类 Agent 框架的集成示例**、**六平台实机验证反馈**。详见 [CONTRIBUTING.md](CONTRIBUTING.md)。

---

### 6. 📦 打包与部署指南

按本项目发布规范，Rubicon 属于**软件 / 可执行程序类项目**，因此**必须完成全平台打包并上传 Release 附件**。

#### 一键打包

```bash
make release           # 等价于 VERSION=1.0.0 bash scripts/build.sh
ls -1 dist/
```

脚本用纯 Go 工具链 + `tar` / `zip` 完成交叉编译，产物在 `dist/`：

| 平台 | 架构 | 产物 |
| --- | --- | --- |
| Linux | amd64 / arm64 | `rubicon_1.0.0_linux_amd64.tar.gz`、`..._arm64.tar.gz` |
| macOS | amd64 / arm64 | `rubicon_1.0.0_darwin_amd64.tar.gz`、`..._arm64.tar.gz` |
| Windows | amd64 / arm64 | `rubicon_1.0.0_windows_amd64.zip`、`..._arm64.zip` |

同时生成 `dist/SHA256SUMS`；每个压缩包内含可执行文件、`LICENSE` 与 `README.md`。

#### 校验和

```bash
cd dist && sha256sum -c SHA256SUMS
```

#### 部署方法

**没有部署**：不需要安装器、不需要运行时、不需要配置文件。把单个可执行文件拷到目标机器任意目录加入 `PATH` 即可；`rubicon init` 会在当前项目生成 `.rubicon/`。容器里同样只需一个 `COPY`：

```dockerfile
FROM scratch
COPY rubicon /rubicon
ENTRYPOINT ["/rubicon"]
```

#### 兼容环境

| 环境 | 支持情况 |
| --- | --- |
| Linux glibc / musl / Alpine | ✅ 静态二进制，`CGO_ENABLED=0` |
| macOS 10.15+（Intel / Apple Silicon） | ✅ |
| Windows 10+（x64 / ARM64） | ✅ |
| 需要 Go / Python / JVM 运行时 | ❌ **完全不需要** |
| 网络 / 数据库 / 后台服务 | ❌ **完全不需要** |

> ⬇️ **正式下载地址**：[https://github.com/gitstq/Rubicon/releases/tag/v1.0.0](https://github.com/gitstq/Rubicon/releases/tag/v1.0.0)（含全部平台附件与 `SHA256SUMS`）

---

### 7. 🤝 贡献指南

- 🍴 Fork 后从 `main` 建分支，分支名建议 `feat/xxx`、`fix/xxx`。
- ✅ 提交前跑 `make all`（含 `go vet` 与 `go test -race`），并确保 `rubicon verify`、`rubicon lint` 均返回 0。
- 📝 Commit 遵循 Angular / Conventional Commits：`feat(cli): ...`、`fix(store): ...`、`docs: ...`、`test(engine): ...`。
- 🚫 **不接受引入第三方依赖**的 PR；「零依赖」是本项目的硬约束。
- 🌏 用户可见变更需**同步更新本文档全部三个语言版本**。
- 🐛 提 Issue 请附：完整命令、`--now`、`--format`、预期与实际输出；**切勿粘贴任何凭证或密钥**。

完整规范见 [CONTRIBUTING.md](CONTRIBUTING.md)。

---

### 8. 📄 开源协议说明

本项目采用 **MIT License**，详见 [LICENSE](LICENSE)。

> 你可以自由地使用、复制、修改、合并、发布、分发、再授权及商业销售本软件，唯一条件是保留版权声明与许可声明。软件按「原样」提供，**不附带任何明示或默示担保**。

---
---

<a id="zh-tw"></a>

## 繁體中文

### 1. 🎉 專案介紹

**Rubicon 是一套離線優先、零依賴的決策卡與會話交接工具鏈**，專門處理 AI 編碼 Agent 在長任務中最棘手的兩個片刻：

- 🤖 **該不該動手？** 碰到「刪掉舊遷移腳本」「把發佈標籤打上去」這類不可逆動作時，Agent 不該自己猜，也不該把整段對話紀錄丟給人類。
- 🔄 **換個會話就失憶？** 上一個會話做了什麼、還有哪些問題懸著、下一個會話該從哪一步接上，往往只存在聊天紀錄裡，一斷就沒了。

Rubicon 把這兩個片刻收斂成同一種結構化物件：**決策卡（Decision Card）**。Agent 遇到不確定事項就開一張卡，人類用一條指令批准或否決；會話結束時產生一份**交接包（Handoff）**，只帶走未決卡、已完成步驟與異動檔案，不帶走聊天紀錄。

**靈感來源**：近期 GitHub 熱議的 `agent-autoloop`（背景自動接活、拿不定主意才找人拍板）、`claude-mem`（跨會話持久上下文）與 `spec-kit`（規格驅動開發），共同指向同一個缺口——**Agent 流程裡少了一個可驗證的人工決策點**。本專案僅參考其產品邏輯與欲解決的痛點，**所有程式碼 100% 獨立自研，未複製原專案任何一行程式**。

**自研差異化亮點**：

| 面向 | 常見做法 | Rubicon |
| --- | --- | --- |
| 決策紀錄 | 寫在對話紀錄 / Markdown 備註 | **只追加的雜湊鏈帳本**，改一個字都會被抓到 |
| 狀態來源 | 直接改 JSON 檔 | **事件溯源**，現況永遠由重放帳本推導 |
| 時間行為 | 讀系統時脈，CI 結果飄忽 | **`--now` 可注入**，同輸入必得同輸出 |
| 會話交接 | 匯出整段對話 | **結構化交接包 + 摘要校驗**，只留必要資訊 |
| 交付形態 | Python 腳本加一堆套件 | **Go 單一靜態執行檔**，六平台開箱即用 |

> 📌 **一句話定位**：Rubicon 不是又一個程式碼稽核器，而是 Agent 工作流裡的「決策閘門 + 交接協定」。

---

### 2. ✨ 核心特性

- 🧾 **結構化決策卡**：標題、明確提問、至少兩個選項、**安全預設動作**、影響範圍、可回滾標記、期限與標籤，欄位齊全才准落帳。
- 🔗 **只追加雜湊鏈帳本**：每筆事件 `sha256(前序雜湊 | 事件類型 | 時間 | 載荷)`，任何竄改、刪行、插行都會被 `verify` 精確定位到事件序號。
- ♻️ **事件溯源狀態機**：`pending → approved / rejected / expired / superseded`，狀態不落碟，全由重放推導，**永不原地修改**。
- ⏰ **確定性逾時**：`--now` 固定時脈，`expire` 冪等；期限一過，Agent 不能再偷偷自己決定。
- 📦 **可驗證交接包**：`handoff build` 產生含 SHA-256 摘要的 JSON，`handoff resume` 先驗摘要再輸出接續簡報，**改一個字就拒絕**。
- 🚦 **CI 門禁就緒**：`verify` 斷鏈回傳 3，`lint` 有錯誤級規則回傳 3，可直接串進流水線；支援 `--format table|json|md`。
- 🧩 **16 條內建規則**：從「選項少於兩個」到「未宣告可否回滾」，逐條給出規則編號、嚴重層級與命中卡片。
- 🪶 **零第三方依賴**：僅用 Go 標準函式庫，無網路呼叫、無環境變數、無隱含狀態，`go.mod` 乾乾淨淨。
- 📱 **六平台原生執行檔**：Linux / macOS / Windows × amd64 / arm64，`CGO_ENABLED=0` 靜態編譯，拷過去就能跑。

---

### 3. 🚀 快速開始

#### 環境需求

| 項目 | 需求 |
| --- | --- |
| 直接跑預編譯包 | **無需求**，任意 Windows / macOS / Linux（amd64 或 arm64） |
| 從原始碼建置 | Go **1.21+**、`make`、`bash`（打包腳本另需 `tar` / `zip`） |
| 網路 | **完全不需要**，工具執行時不發任何請求 |
| 環境變數 | **不需設定任何環境變數**，所有行為由命令列參數決定，避免隱含狀態 |

#### 方式一：下載 Release（最推薦）

前往 👉 **[v1.0.0 Release 下載頁](https://github.com/gitstq/Rubicon/releases/tag/v1.0.0)**，依平台取一個壓縮檔，解開後把 `rubicon`（Windows 為 `rubicon.exe`）放進 `PATH` 即可。用 `SHA256SUMS` 驗證完整性：

```bash
# Linux / macOS
sha256sum -c SHA256SUMS --ignore-missing
./rubicon version
```

```powershell
# Windows PowerShell
Get-FileHash .\rubicon_1.0.0_windows_amd64.zip -Algorithm SHA256
.\rubicon.exe version
```

#### 方式二：從原始碼建置

```bash
git clone https://github.com/gitstq/Rubicon.git
cd Rubicon
make all          # gofmt + go vet + go test -race + 建置
./dist/rubicon version
```

#### 方式三：go install

```bash
go install github.com/gitstq/Rubicon/cmd/rubicon@latest
rubicon version
```

#### 三分鐘跑完整個閉環

```bash
# 1️⃣ 初始化工作區（在目前目錄產生 .rubicon/ledger.jsonl）
rubicon init

# 2️⃣ Agent 碰到不可逆動作時，開一張決策卡
rubicon card new \
  --title "刪除舊版遷移腳本？" \
  --question "我可以刪除 db/migrate/001.sql 嗎？" \
  --options "A:刪除|high,B:保留|low" \
  --default B \
  --files "db/migrate/001.sql" \
  --reversible false

# 3️⃣ 人類拍板
rubicon decide dc_0001 --choice A --by alice --note "評審已確認"

# 4️⃣ CI 裡守兩道門
rubicon verify && rubicon lint

# 5️⃣ 會話結束，產生交接包
rubicon handoff build --summary "重構第一階段完成" --out handoff.json
rubicon handoff resume --file handoff.json
```

> 💡 想一次看完全流程？執行 `make demo` 或 `bash examples/demo.sh`，演示固定了 `--now`，**每次輸出完全一致**。

---

### 4. 📖 詳細使用指南

#### 指令一覽

| 指令 | 用途 |
| --- | --- |
| `rubicon init` | 建立 `.rubicon` 工作區 |
| `rubicon card new [flags]` | 新增一張待決卡 |
| `rubicon card list [--status STATUS]` | 列出卡片，可依狀態過濾 |
| `rubicon card show <id>` | 查看單張卡完整 JSON |
| `rubicon decide <id> --choice K` | 批准某個選項 |
| `rubicon reject <id>` | 否決該決策 |
| `rubicon supersede <id>` | 標記為已被後續決策取代 |
| `rubicon expire` | 關閉所有逾期未決卡（冪等） |
| `rubicon verify` | 校驗雜湊鏈完整性（CI 門禁） |
| `rubicon lint` | 依 16 條規則健檢所有卡片（CI 門禁） |
| `rubicon report` | 輸出完整工作區報告 |
| `rubicon handoff build [flags]` | 產生含摘要的交接包 |
| `rubicon handoff resume --file F` | 校驗並輸出接續簡報 |
| `rubicon version` / `help` | 版本 / 說明 |

#### 全域參數（可置於指令前後任一處）

| 參數 | 預設 | 說明 |
| --- | --- | --- |
| `--dir DIR` | `.rubicon` | 工作區目錄，內含 `ledger.jsonl` |
| `--format F` | `table` | `table` / `json` / `md` 三種輸出格式 |
| `--now RFC3339` | 系統 UTC 時間 | **注入固定時脈**，讓 CI 與測試完全可重現 |

#### `card new` 參數說明

| 參數 | 必填 | 說明 |
| --- | --- | --- |
| `--id` | 否 | 卡片 ID，省略時自動產生 `dc_0001`、`dc_0002`… |
| `--title` | ✅ | 一句話標題 |
| `--question` | ✅ | 人類需要回答的確切問題 |
| `--options` | ✅ | `KEY:標籤[|影響],KEY:標籤[|影響]`，至少兩項 |
| `--default` | ✅ | 逾時或失聯時 Agent 應採取的安全選項鍵 |
| `--context` | 否 | 給審閱者的背景資訊 |
| `--deadline` | 否 | RFC3339 期限，如 `2026-11-10T10:00:00Z` |
| `--files` / `--scope` | 否 | 受影響檔案（逗號分隔）/ 影響範圍描述 |
| `--reversible` | 否 | `true` / `false`；未填會被 `lint` 提示 |
| `--tags` | 否 | 逗號分隔標籤 |

#### 輸出格式與結束碼

```bash
rubicon --format json card list --status pending   # 機器可讀
rubicon --format md   report                     # 貼進 PR / 文件
rubicon              report                     # 終端表格
```

| 結束碼 | 意義 |
| --- | --- |
| `0` | 成功 |
| `1` | 執行期錯誤（檔案不可寫、卡片不存在等） |
| `2` | 用法錯誤（缺參數、格式非法） |
| `3` | **CI 門禁失敗**：鏈遭竄改，或 `lint` 出現錯誤級規則 |

```yaml
# .github/workflows/gate.yml 片段
- name: Decision ledger gate
  run: |
    rubicon --dir .rubicon verify
    rubicon --dir .rubicon lint
```

#### 帳本與設定教學

工作區只有一個檔案：`.rubicon/ledger.jsonl`。**每行一筆不可變事件**，形如：

```json
{"seq":1,"prev_hash":"0000...0000","hash":"9f2c...","kind":"card.created","ts":"2026-11-09T10:00:00Z","payload":{"id":"dc_0001", "...": "..."}}
```

- `payload` 是該時刻卡片的**完整快照**，因此重放帳本即可還原任意時刻狀態。
- 不需要任何設定檔；策略以命令列參數表達。
- 建議把帳本隨程式碼提交，或由 CI 歸檔為產物；**請勿手動編輯**，一改就會被 `verify` 抓到。
- 範例卡請見 [`examples/decision-card.json`](examples/decision-card.json)。

#### 典型情境

**情境 A · Agent 送出決策卡**：Agent 執行前把不確定項寫成卡，偵測到 `pending` 卡就暫停並通知人類；人類 `decide` 後，Agent 重放帳本繼續。

**情境 B · 無人值守逾時**：為卡設 `--deadline`，人類沒回，CI 定時跑 `rubicon expire`，卡自動落到 `expired`，Agent 依 `--default` 走安全路徑，**絕不擅自執行高影響動作**。

**情境 C · 跨會話交接**：會話 A 結束前 `handoff build`，會話 B 一上來 `handoff resume`，立刻知道還有幾張卡沒拍板、哪些檔案動過、下一步該做什麼，不必讀幾千行對話。

**情境 D · 合規稽核**：`verify` 加上 `report --format md` 直接產出「誰在何時批准了什麼」的證據鏈，鏈頭雜湊可貼進稽核紀錄。

> 📷 **演示截圖 / 動圖預留位**：`docs/screenshots/cli-demo.gif`（預計補上終端動圖）、`docs/screenshots/report-md.png`（Markdown 報告呈現效果）。現階段請執行 `make demo` 查看等價的終端實錄。

---

### 5. 💡 設計思路與迭代規劃

#### 設計理念

1. **帳本即真相**：不存「目前狀態」，只存「發生過什麼」。狀態是重放的結果，因此無法被偷偷改掉。
2. **可驗證優於可信任**：雜湊鏈與交接摘要，讓「這份紀錄沒被動過」成為可計算的結論，而非一句承諾。
3. **確定性是一等公民**：時脈可注入、無網路、無亂數、切片排序後才算摘要，同樣輸入在任何平台給出同樣位元組。
4. **最小必要資訊**：交接包只帶未決卡、步驟與檔案清單，不搬整段對話，避免上下文膨脹。
5. **人類只做關鍵決策**：卡必須至少兩個選項、必須有安全預設動作，逼著 Agent 把問題問清楚。

#### 為什麼選 Go

| 選項 | 結論 |
| --- | --- |
| Python | 生態佳，但「免依賴執行檔」需 PyInstaller，且**無法在同一台 Linux 上交叉產出 macOS / Windows 原生包** |
| Rust | 效能好，但標準函式庫以外的套件會打破「零依賴」，編譯鏈也更重 |
| **Go** ✅ | 標準函式庫已足夠（JSON / SHA-256 / flag / tabwriter 全包），`CGO_ENABLED=0` 一行交叉編譯出六平台靜態執行檔，**真正開箱即用** |

#### 迭代規劃

- 🗓️ **v1.1**：`rubicon diff` 比對兩份交接包只顯示增量；卡片優先順序與依賴（卡 A 未決時阻塞卡 B）。
- 🗓️ **v1.2**：可選的本機 Web 審核面板（仍是單一執行檔、仍然離線）。
- 🗓️ **v1.3**：MCP Server 模式，讓 Agent 直接透過工具呼叫讀寫決策卡。
- 🗓️ **v2.0**：多審閱者簽章與法定簽核鏈（維持零第三方依賴）。

#### 社群貢獻方向

最歡迎：**新的 lint 規則**、**更多輸出格式**、**各類 Agent 框架整合範例**、**六平台實機驗證回饋**。詳見 [CONTRIBUTING.md](CONTRIBUTING.md)。

---

### 6. 📦 打包與部署指南

依本專案發佈規範，Rubicon 屬**軟體 / 可執行程式類專案**，因此**必須完成全平台打包並上傳 Release 附件**。

#### 一鍵打包

```bash
make release           # 等同 VERSION=1.0.0 bash scripts/build.sh
ls -1 dist/
```

腳本只用 Go 工具鏈與 `tar` / `zip` 完成交叉編譯，產物置於 `dist/`：

| 平台 | 架構 | 產物 |
| --- | --- | --- |
| Linux | amd64 / arm64 | `rubicon_1.0.0_linux_amd64.tar.gz`、`..._arm64.tar.gz` |
| macOS | amd64 / arm64 | `rubicon_1.0.0_darwin_amd64.tar.gz`、`..._arm64.tar.gz` |
| Windows | amd64 / arm64 | `rubicon_1.0.0_windows_amd64.zip`、`..._arm64.zip` |

同時產生 `dist/SHA256SUMS`；每個壓縮檔內含執行檔、`LICENSE` 與 `README.md`。

#### 校驗和

```bash
cd dist && sha256sum -c SHA256SUMS
```

#### 部署方式

**沒有部署**：不需要安裝程式、不需要執行環境、不需要設定檔。把單一執行檔拷到目標機器任意目錄並加入 `PATH` 即可；`rubicon init` 會在該專案產生 `.rubicon/`。容器裡同樣只需一個 `COPY`：

```dockerfile
FROM scratch
COPY rubicon /rubicon
ENTRYPOINT ["/rubicon"]
```

#### 相容環境

| 環境 | 支援 |
| --- | --- |
| Linux glibc / musl / Alpine | ✅ 靜態執行檔，`CGO_ENABLED=0` |
| macOS 10.15+（Intel / Apple Silicon） | ✅ |
| Windows 10+（x64 / ARM64） | ✅ |
| 需要 Go / Python / JVM 執行環境 | ❌ **完全不需要** |
| 網路 / 資料庫 / 背景服務 | ❌ **完全不需要** |

> ⬇️ **正式下載位置**：[https://github.com/gitstq/Rubicon/releases/tag/v1.0.0](https://github.com/gitstq/Rubicon/releases/tag/v1.0.0)（含全部平台附件與 `SHA256SUMS`）

---

### 7. 🤝 貢獻指南

- 🍴 Fork 後從 `main` 開分支，分支名建議 `feat/xxx`、`fix/xxx`。
- ✅ 送 PR 前跑 `make all`（含 `go vet` 與 `go test -race`），並確認 `rubicon verify`、`rubicon lint` 都回傳 0。
- 📝 Commit 遵循 Angular / Conventional Commits：`feat(cli): ...`、`fix(store): ...`、`docs: ...`、`test(engine): ...`。
- 🚫 **不接受引入第三方依賴**的 PR；「零依賴」是本專案硬約束。
- 🌏 使用者可見變動需**同步更新本文件三個語言版本**。
- 🐛 提 Issue 請附：完整指令、`--now`、`--format`、預期與實際輸出；**切勿貼上任何憑證或金鑰**。

完整規範見 [CONTRIBUTING.md](CONTRIBUTING.md)。

---

### 8. 📄 開源授權說明

本專案採用 **MIT License**，詳見 [LICENSE](LICENSE)。

> 你可以自由使用、複製、修改、合併、發佈、散佈、再授權及商業銷售本軟體，唯一條件是保留版權聲明與授權聲明。軟體按「現況」提供，**不附帶任何明示或默示擔保**。

---
---

<a id="en"></a>

## English

### 1. 🎉 What is Rubicon?

**Rubicon is an offline-first, dependency-free toolkit for decision cards and session handoff in AI-agent workflows.** It targets the two moments where long-running coding agents are most awkward to work with:

- 🤖 **Should the agent act?** When it reaches an irreversible step — deleting a legacy migration, tagging a release — it should neither guess nor dump an entire chat log on a human.
- 🔄 **Amnesia between sessions.** What the previous session did, which questions are still open, and where the next session should resume usually live only in a transcript that disappears when the session ends.

Rubicon collapses both moments into one structured object: a **Decision Card**. The agent raises a card when it is unsure; a human approves or rejects it with a single command. At the end of a session, a **Handoff** package carries only the unresolved cards, completed steps and touched files — never the transcript.

**Inspiration.** Recent GitHub discussions around `agent-autoloop` (background agents that ask a human only when stuck), `claude-mem` (persistent cross-session context) and `spec-kit` (spec-driven development) all point at the same gap: **agent workflows lack a verifiable human decision point**. Rubicon borrows only the product logic and the problem statement. **Every line of code here was written independently; no source was copied from any of those projects.**

**What we deliberately did differently:**

| Dimension | Common approach | Rubicon |
| --- | --- | --- |
| Decision record | Chat log or a Markdown note | **Append-only hash-chained ledger** — a single edited character is detectable |
| Source of truth | Mutate a JSON file | **Event sourcing** — current state is always derived by replay |
| Time behaviour | Reads the wall clock, so CI output drifts | **Injectable `--now`** — identical input always yields identical output |
| Session handoff | Export the whole conversation | **Structured, digest-sealed package** with only the essentials |
| Delivery | Python script plus a pile of packages | **One static Go binary**, six platforms, no install step |

> 📌 **In one sentence**: Rubicon is not another code auditor. It is the *decision gate and handoff protocol* inside an agent workflow.

---

### 2. ✨ Key Features

- 🧾 **Structured decision cards** — title, an explicit question, at least two options, a **safe default action**, blast radius, rollback flag, deadline and tags. Incomplete cards are rejected before they reach the ledger.
- 🔗 **Append-only hash chain** — every event is `sha256(prev_hash | kind | timestamp | payload)`; `verify` pinpoints the exact event where a tamper, deletion or insertion occurred.
- ♻️ **Event-sourced state machine** — `pending → approved / rejected / expired / superseded`. State is never written directly; **nothing is ever mutated in place**.
- ⏰ **Deterministic expiry** — pin the clock with `--now`; `expire` is idempotent. Once a deadline passes, the agent can no longer quietly decide for itself.
- 📦 **Verifiable handoff** — `handoff build` emits SHA-256-sealed JSON; `handoff resume` verifies the digest before printing a briefing. **Edit one word and it refuses to load.**
- 🚦 **CI-gate ready** — `verify` exits 3 on a broken chain, `lint` exits 3 on any error-severity rule. Three output formats: `--format table|json|md`.
- 🧩 **16 built-in rules** — from "fewer than two options" to "rollback risk not declared", each with a rule ID, severity and the affected card.
- 🪶 **Zero third-party dependencies** — Go standard library only. No network, no environment variables, no hidden state; `go.mod` contains no `require` block.
- 📱 **Six native binaries** — Linux / macOS / Windows × amd64 / arm64, built with `CGO_ENABLED=0`. Copy one file and run it.

---

### 3. 🚀 Quick Start

#### Requirements

| Item | Requirement |
| --- | --- |
| Prebuilt binary | **None** — any Windows / macOS / Linux on amd64 or arm64 |
| Building from source | Go **1.21+**, `make`, `bash` (`tar` / `zip` only for the packaging script) |
| Network | **Not needed** — the tool makes zero requests at runtime |
| Environment variables | **None to configure** — all behaviour is explicit on the command line |

#### Option 1 — Download a release (fastest)

Grab the archive for your platform from the 👉 **[v1.0.0 release page](https://github.com/gitstq/Rubicon/releases/tag/v1.0.0)**, extract it, and put `rubicon` (`rubicon.exe` on Windows) on your `PATH`. Verify integrity first:

```bash
# Linux / macOS
sha256sum -c SHA256SUMS --ignore-missing
./rubicon version
```

```powershell
# Windows PowerShell
Get-FileHash .\rubicon_1.0.0_windows_amd64.zip -Algorithm SHA256
.\rubicon.exe version
```

#### Option 2 — Build from source

```bash
git clone https://github.com/gitstq/Rubicon.git
cd Rubicon
make all          # gofmt + go vet + go test -race + build
./dist/rubicon version
```

#### Option 3 — go install

```bash
go install github.com/gitstq/Rubicon/cmd/rubicon@latest
rubicon version
```

#### A complete loop in three minutes

```bash
# 1️⃣ Create the workspace (.rubicon/ledger.jsonl in the current directory)
rubicon init

# 2️⃣ The agent raises a card before an irreversible action
rubicon card new \
  --title "Delete the legacy migration?" \
  --question "May I remove db/migrate/001.sql?" \
  --options "A:Delete|high,B:Keep it|low" \
  --default B \
  --files "db/migrate/001.sql" \
  --reversible false

# 3️⃣ A human decides
rubicon decide dc_0001 --choice A --by alice --note "Confirmed in review"

# 4️⃣ Two gates in CI
rubicon verify && rubicon lint

# 5️⃣ End the session with a handoff
rubicon handoff build --summary "Refactor stage 1 complete" --out handoff.json
rubicon handoff resume --file handoff.json
```

> 💡 Want to see the whole flow at once? Run `make demo` or `bash examples/demo.sh`. The demo pins `--now`, so **its output is byte-for-byte reproducible**.

---

### 4. 📖 Detailed Usage

#### Command reference

| Command | Purpose |
| --- | --- |
| `rubicon init` | Create the `.rubicon` workspace |
| `rubicon card new [flags]` | Raise a new pending card |
| `rubicon card list [--status STATUS]` | List cards, optionally filtered by status |
| `rubicon card show <id>` | Print one card as full JSON |
| `rubicon decide <id> --choice K` | Approve a specific option |
| `rubicon reject <id>` | Reject the decision |
| `rubicon supersede <id>` | Mark as replaced by a later decision |
| `rubicon expire` | Close all overdue pending cards (idempotent) |
| `rubicon verify` | Verify the hash chain (CI gate) |
| `rubicon lint` | Run all 16 rules over every card (CI gate) |
| `rubicon report` | Full workspace report |
| `rubicon handoff build [flags]` | Build a sealed handoff package |
| `rubicon handoff resume --file F` | Verify and print the resume briefing |
| `rubicon version` / `help` | Version / help |

#### Global flags (may appear before or after the subcommand)

| Flag | Default | Meaning |
| --- | --- | --- |
| `--dir DIR` | `.rubicon` | Workspace directory containing `ledger.jsonl` |
| `--format F` | `table` | `table`, `json` or `md` output |
| `--now RFC3339` | system UTC clock | **Inject a fixed clock** so CI and tests are fully reproducible |

#### `card new` flags

| Flag | Required | Meaning |
| --- | --- | --- |
| `--id` | no | Card ID; auto-generated as `dc_0001`, `dc_0002`, … when omitted |
| `--title` | ✅ | One-line title |
| `--question` | ✅ | The exact question a human must answer |
| `--options` | ✅ | `KEY:Label[|impact],KEY:Label[|impact]` — at least two |
| `--default` | ✅ | Safe option key the agent falls back to on timeout or disconnect |
| `--context` | no | Background for the reviewer |
| `--deadline` | no | RFC3339 deadline, e.g. `2026-11-10T10:00:00Z` |
| `--files` / `--scope` | no | Comma-separated impacted files / free-text blast radius |
| `--reversible` | no | `true` or `false`; leaving it out produces a lint warning |
| `--tags` | no | Comma-separated tags |

#### Output formats and exit codes

```bash
rubicon --format json card list --status pending   # machine-readable
rubicon --format md   report                     # paste into a PR or docs
rubicon              report                     # terminal table
```

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | Runtime error (unwritable file, unknown card, …) |
| `2` | Usage error (missing argument, invalid format) |
| `3` | **CI gate failed** — chain tampered, or `lint` found an error-severity rule |

```yaml
# .github/workflows/gate.yml fragment
- name: Decision ledger gate
  run: |
    rubicon --dir .rubicon verify
    rubicon --dir .rubicon lint
```

#### The ledger, explained

A workspace is exactly one file: `.rubicon/ledger.jsonl`. **One immutable event per line**:

```json
{"seq":1,"prev_hash":"0000...0000","hash":"9f2c...","kind":"card.created","ts":"2026-11-09T10:00:00Z","payload":{"id":"dc_0001", "...": "..."}}
```

- `payload` is a **full snapshot** of the card at that moment, so replaying the ledger reconstructs any point in time.
- There is no config file; policy is expressed entirely through CLI flags.
- Commit the ledger with your code or archive it as a CI artifact. **Never edit it by hand** — `verify` will catch you.
- Reference card: [`examples/decision-card.json`](examples/decision-card.json).

#### Typical scenarios

**A · Agent emits cards.** Before an uncertain action the agent writes a card, sees a `pending` card and pauses to notify a human. After `decide`, it replays the ledger and continues.

**B · Unattended timeout.** Set `--deadline`; if nobody answers, CI periodically runs `rubicon expire`, the card lands in `expired`, and the agent follows `--default`. **It can never silently take the high-impact path.**

**C · Cross-session handoff.** Session A ends with `handoff build`; session B starts with `handoff resume` and immediately knows how many cards are unresolved, which files changed and what to do next — without reading thousands of lines of transcript.

**D · Compliance audit.** `verify` plus `report --format md` produces an evidence chain of *who approved what and when*, with a chain-head hash you can paste into an audit record.

> 📷 **Screencast / screenshot placeholders**: `docs/screenshots/cli-demo.gif` (terminal walkthrough, planned) and `docs/screenshots/report-md.png` (rendered Markdown report). Until then, `make demo` prints the equivalent terminal session.

---

### 5. 💡 Design Rationale & Roadmap

#### Principles

1. **The ledger is the truth.** Store *what happened*, not *what is true now*. State is a replay result, so it cannot be quietly edited.
2. **Verifiable beats trusted.** Hash chains and handoff digests turn "this record was not altered" into a computable fact rather than a promise.
3. **Determinism is a first-class feature.** Injectable clock, no network, no randomness, sorted slices before hashing — identical input yields identical bytes on every platform.
4. **Minimum necessary information.** A handoff carries unresolved cards, steps and file lists; it never ships a transcript, so context does not bloat.
5. **Humans decide only what matters.** A card must have two or more options and a safe default, which forces the agent to ask a precise question.

#### Why Go

| Candidate | Verdict |
| --- | --- |
| Python | Great ecosystem, but dependency-free executables need PyInstaller and **cannot be cross-built into native macOS / Windows binaries from a single Linux host** |
| Rust | Fast, but anything beyond the standard library breaks the zero-dependency constraint and adds a heavier toolchain |
| **Go** ✅ | The standard library already covers JSON, SHA-256, flags and tabular output; `CGO_ENABLED=0` cross-compiles six static binaries in one command — **genuinely double-click-and-run** |

#### Roadmap

- 🗓️ **v1.1** — `rubicon diff` between two handoffs (increments only); card priorities and dependencies (card A blocks card B while unresolved).
- 🗓️ **v1.2** — Optional local web review panel (still one binary, still offline).
- 🗓️ **v1.3** — MCP server mode so agents can read and write cards through tool calls.
- 🗓️ **v2.0** — Multi-reviewer signatures and formal approval chains, still with zero third-party dependencies.

#### Where contributions help most

**New lint rules**, **additional output formats**, **integration examples for agent frameworks**, and **real-machine validation on all six platforms**. See [CONTRIBUTING.md](CONTRIBUTING.md).

---

### 6. 📦 Packaging & Deployment

Under this project's release policy Rubicon is a **software / executable program**, so it **must be packaged for every supported platform and attached to a Release**.

#### One-command packaging

```bash
make release           # equivalent to VERSION=1.0.0 bash scripts/build.sh
ls -1 dist/
```

The script uses only the Go toolchain plus `tar` / `zip`. Outputs land in `dist/`:

| OS | Architectures | Artifacts |
| --- | --- | --- |
| Linux | amd64 / arm64 | `rubicon_1.0.0_linux_amd64.tar.gz`, `..._arm64.tar.gz` |
| macOS | amd64 / arm64 | `rubicon_1.0.0_darwin_amd64.tar.gz`, `..._arm64.tar.gz` |
| Windows | amd64 / arm64 | `rubicon_1.0.0_windows_amd64.zip`, `..._arm64.zip` |

A `dist/SHA256SUMS` manifest is generated too, and every archive contains the binary, `LICENSE` and `README.md`.

#### Checksums

```bash
cd dist && sha256sum -c SHA256SUMS
```

#### Deployment

**There is no deployment.** No installer, no runtime, no config file. Copy the single executable anywhere, add it to `PATH`, and run `rubicon init` inside your project. Even a container needs only one `COPY`:

```dockerfile
FROM scratch
COPY rubicon /rubicon
ENTRYPOINT ["/rubicon"]
```

#### Compatibility

| Environment | Supported |
| --- | --- |
| Linux glibc / musl / Alpine | ✅ static binary, `CGO_ENABLED=0` |
| macOS 10.15+ (Intel / Apple Silicon) | ✅ |
| Windows 10+ (x64 / ARM64) | ✅ |
| Requires Go / Python / JVM runtime | ❌ **not required** |
| Network / database / background service | ❌ **not required** |

> ⬇️ **Official downloads**: [https://github.com/gitstq/Rubicon/releases/tag/v1.0.0](https://github.com/gitstq/Rubicon/releases/tag/v1.0.0) — every platform archive plus `SHA256SUMS`.

---

### 7. 🤝 Contributing

- 🍴 Fork and branch from `main`; prefer `feat/xxx` or `fix/xxx` branch names.
- ✅ Run `make all` (includes `go vet` and `go test -race`) and confirm `rubicon verify` and `rubicon lint` both return 0 before opening a PR.
- 📝 Use Angular / Conventional Commits: `feat(cli): ...`, `fix(store): ...`, `docs: ...`, `test(engine): ...`.
- 🚫 PRs that **add a third-party dependency will not be accepted** — zero dependencies is a hard constraint.
- 🌏 Any user-visible change must **update all three language versions of this README**.
- 🐛 When filing an issue, include the full command, `--now`, `--format`, expected and actual output. **Never paste credentials or keys.**

Full guidelines: [CONTRIBUTING.md](CONTRIBUTING.md).

---

### 8. 📄 License

Released under the **MIT License** — see [LICENSE](LICENSE).

> You may use, copy, modify, merge, publish, distribute, sublicense and sell copies of this software, provided the copyright and permission notices are retained. The software is provided **"as is", without warranty of any kind, express or implied**.
