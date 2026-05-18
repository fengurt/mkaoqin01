# 班次网格 JSON — Agent 数据准备指南

> 面向 Cursor / 自动化 Agent：为 Intervoice「团队考勤 → 班次导入/导出」准备可一次导入成功的 JSON。  
> 参考样例：仓库根目录 `20260517.json`（Chronoscape 导出）。

## 1. 推荐工作流

1. **先导出**：管理员登录 → 团队考勤 → 设置日期区间 → **导出 JSON**。  
   导出文件已含 `users`、`employees`、`roster`、`activeTags`、`assignments`，可直接作为模板。
2. **再编辑**：只改 `assignments`（或 Chronoscape 的 `cells`），不要改 `users.userId` 与系统账号不一致。
3. **后导入**：同一页面 **导入 JSON**，查看 toast：`写入 N 条，跳过 M，未匹配 K`。

## 2. 支持的 Schema

### A. `intervoice.scheduleGrid.v1`（推荐）

```json
{
  "schema": "intervoice.scheduleGrid.v1",
  "schemaVersion": 2,
  "dateRange": { "start": "2026-05-11", "end": "2026-05-24" },
  "users": [
    { "userId": 132369, "account": "132369", "displayName": "Albee Liu", "role": "employee", "normalizedName": "albeeliu" }
  ],
  "employees": [ "…非 admin 账号子集…" ],
  "roster": [
    { "objectId": 132369, "objectName": "Albee Liu", "account": "132369", "simObjectId": 70132369, "sortOrder": 1, "isActive": true }
  ],
  "activeTags": [
    { "tagItemName": "RDO", "mode": "leave", "code": "RDO" },
    { "tagItemName": "1330-2306", "mode": "work", "code": "1330-2306" }
  ],
  "assignments": [
    {
      "userId": 132369,
      "account": "132369",
      "displayName": "Albee Liu",
      "objectName": "Albee Liu",
      "date": "2026-05-12",
      "mode": "work",
      "code": "1330-2306",
      "tagItemName": "1330-2306"
    }
  ]
}
```

### B. `chronoscape.simulationProjectData`（兼容 `20260517.json`）

- `roster[].objectName`：员工显示名（与 `users.display_name` 一致）
- `cells[]`：每条 `{ "objectName", "date", "tagItemName" }`
- `tagItemName` 必须与 `activeTags` / 样例中的名称一致（见下表）

## 3. 用户匹配规则（导入时）

按优先级解析员工（任一命中即可）：

| 字段 | 说明 |
|------|------|
| `userId` | 最可靠，来自导出 JSON 的 `users` |
| `account` | 登录账号，如 `132369` |
| `displayName` / `objectName` | 显示名，如 `Albee Liu` |
| `normalizedName` | 仅字母数字（导出里已算好） |

**Agent 注意**：新建 `assignments` 时务必带上 `userId` + `account` + `displayName` 三件套（从导出文件的 `employees` 复制）。

## 4. 班次标签 `tagItemName`（与 importdata / Chronoscape 对齐）

| tagItemName | 导入后 mode | code | 说明 |
|-------------|-------------|------|------|
| `RDO` | leave | RDO | 休息日 |
| `AL` | leave | AL | 年假 |
| `PHCL` | leave | PHCL | 公众假期补休 |
| `RDOC` | leave | RDOC | |
| `24 hours available on mobile` | work | STANDBY24 | 24 小时手机待命 |
| `1330-2306` | work | 1330-2306 | 时段班（格式 `HHMM-HHMM`） |
| `1030-2006` | work | 1030-2006 | |
| `1100-2036` | work | 1100-2036 | |
| `12:00-21:36` | work | 12:00-21:36 | 可含冒号 |

休假类：`tagItemName` = `code` = 上表左列。  
工作类：时段字符串会写入 `shift_types`；待命必须用完整英文 `24 hours available on mobile`。

## 5. Agent 生成 assignments 的检查清单

- [ ] 日期格式 `YYYY-MM-DD`，且在 `dateRange` 内  
- [ ] 每个 `userId` 存在于导出的 `users`  
- [ ] 每条有 `tagItemName`（或 `mode`+`code`）  
- [ ] 休假用 `mode:"leave"` + `code:"RDO"` 等  
- [ ] 工作时段班 `tagItemName` 无多余空格（`1330-2306` 而非 `13:30-23:06` 除非样例如此）  
- [ ] 不要为 admin 账号（`role: admin`）写排班  

## 6. 从 Excel / 表格生成 JSON 的提示

表头建议：

| account | displayName | date | tagItemName |
|---------|-------------|------|-------------|
| 132369 | Albee Liu | 2026-05-12 | 1330-2306 |

脚本步骤：

1. 读取导出 JSON 的 `users`，建 `account → userId` 映射。  
2. 每行生成一条 `assignment`，填入 `userId`、`account`、`displayName`、`objectName`、`date`、`tagItemName`。  
3. 根据第 4 节表推断 `mode` / `code`（或只填 `tagItemName` 让服务端 `mapTagToSchedule` 处理）。

## 7. 常见错误

| 现象 | 原因 |
|------|------|
| 未匹配 K 条 | 姓名/账号与 `users` 不一致；检查拼写与空格 |
| 员工 0 | 用了旧版后端导出；应含 `users` 数组，或前端会从 `/v1/auth/users` 补全 |
| 跳过增多 | `date` 空、`tagItemName` 无法识别 |
| 导入 404 | 需 `bash scripts/dev-up.sh` 重启 gateway/admin |

## 8. 仓库内相关文件

| 路径 | 用途 |
|------|------|
| `importdata/SCHEDULE_GRID_AGENT_GUIDE.md` | 本指南（给 Agent） |
| `20260517.json` | Chronoscape 完整样例（25 人 × 14 天） |
| `app/admin/schedule_grid_admin.go` | 导入/导出实现 |
| `data/intervoice.db` | SQLite 用户与排班数据 |

## 9. 最小可导入示例

```json
{
  "schema": "intervoice.scheduleGrid.v1",
  "assignments": [
    {
      "userId": 132369,
      "account": "132369",
      "displayName": "Albee Liu",
      "date": "2026-05-17",
      "tagItemName": "RDO"
    }
  ]
}
```

单条也可导入；建议生产环境使用完整导出模板再批量修改 `assignments`。
