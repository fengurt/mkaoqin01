# UI 设计参考（Intervoice H5）

## 灵感来源

- **站点**: [designprompts.dev](https://www.designprompts.dev/) — 多风格并排对比、可复制的 AI 设计提示词（本次抓取超时，依据公开介绍与站内常见分类整理）。
- **选用风格组合**（与综合度假村 H5 场景匹配）:
  - **Luxury / Editorial**：克制高光、暖金点缀、深墨底栏，强调「礼遇感」而非霓虹博彩感。
  - **Swiss Minimalist**：清晰网格、稳定字号阶梯、充足留白，保证考勤/申报信息密度下仍可读。

## 本应用落地原则（结构不变）

- **布局**: 保留首页 Metro 分区（上班 / 下班 / 申报 / 线索 / 好运 + 班次 + 实况），仅调整圆角、阴影、渐变与字重。
- **色彩**: 深炭底栏 + 香槟金强调 + 雾面浅底内容区；磁贴保留语义色（到离岗/申报）但降低饱和度冲突。
- **字体**: 标题/日期使用 **Noto Serif SC** 增强「礼宾简报」气质；正文与按钮保持 **DM Sans** / 系统无衬线以保证中文与数字混排。
- **动效**: 尽量静态；磁贴 hover/active 仅用轻微亮度与阴影变化。

## CSS 变量映射

| 语义 | 变量名 | 说明 |
|------|--------|------|
| 页面雾面底 | `--brand-bg` | `#f3f0ea` |
| 香槟金强调 | `--accent-gold` | `#b8954f` |
| 顶栏 | `--metro-header-bg` | 深炭渐变（见 `style.css`） |
| 正文字体 | `body` | **DM Sans** + 系统中文栈 |
| 标题气质 | `.metro-date` 等 | **Noto Serif SC** |

## 维护

- 大改版时同步更新本文件与 `frontend/src/style.css` 的 `:root` 块。
- 若从 designprompts.dev 选定某一具体命名风格（如 Monochrome 页），可把对应提示词 URL 记在本节下方备查。

参考链接（站内风格示例）: [Monochrome | Design Prompts](https://www.designprompts.dev/monochrome)
