# linuxdo：LINUX DO 的热门帖子

LINUX DO（linux.do，Discourse 搭的中文技术社区）的热门帖子，**只读公开内容、不用账号**（Persona 的 `docs/plugins.md`）：

| 扩展点 | 贡献 | 做什么 |
| --- | --- | --- |
| `ui` | `hot`（`plugin_page`「热门帖子」） | 设置 › 插件 › LINUX DO 里的一页：热门 / 今日最热 / 本周最热 / 最新，按你的兴趣过滤；点一条在这一页里读正文和热门回复，附「在网页打开」 |
| `connectors` | `linuxdo`（status、tools） | 助手的工具 `linuxdo_hot`、`linuxdo_topic`：问助手「LINUX DO 今天有什么值得看的」，它读列表、挑、读帖子再总结 |

## 为什么这样读

- hub 所在的机器多半是服务器：LINUX DO 的防火墙对服务器地址的 JSON 列表（`/latest.json`、`/top.json`）要浏览器验证，**RSS 不要**。
  所以列表读 RSS（`/hot.rss`、`/top.rss?period=daily|weekly`、`/latest.rss`，各 30 条：标题、作者、分类、时间、摘要、帖数和人数），
  帖子读公开的 `/t/topic/<id>.json`（前 20 楼）。2026-11-04 在日本的 VPS 上实测都能读。
- 一次只发一个请求、间隔至少 2 秒，列表缓存 5 分钟、帖子 10 分钟；遇到 429 停 10 分钟（有 `Retry-After` 按它），这期间用缓存里的旧数据。
- 大陆网络访问不了 linux.do：页面里的正文是 hub 读来画的，手机不用代理；「在网页打开」在 App 内打开网页，这一步要手机自己能访问。

## 配置

| 项 | 说明 |
| --- | --- |
| `list` | 打开时看哪个：热门（默认）/ 今日最热 / 本周最热 / 最新 |
| `categories`、`exclude_categories` | 只看 / 不看这些分类（分类名，如「开发调优」「前沿快讯」「搞七捻三」） |
| `keywords`、`exclude_keywords` | 标题或摘要里有其中一个才显示 / 就不显示（不分大小写） |
| `hide_nsfw` | 标题里标了 NSFW 的不显示（默认开） |
| `hide_pinned` | 站务置顶的公告不显示（默认开：热门列表最前面常是几条一两年前的置顶，RSS 的 `discourse:topicPinned`） |
| `tz` | 页面和工具里的时间用的时区 |

页面上可以临时「看全部」（不按兴趣过滤）。工具默认也按这些设置过滤，`filtered: false` 看全部。

## 页面

- 列表：「看什么」——UI 版本 2 的 App 是一个分段控件（热门 / 今日 / 本周 / 最新），更早的 App 是四行（当前的标「正在看」）；列表一节的标题是「热门 · 12 / 30 条」，页脚写按什么过滤；每行是标题和「分类 · 作者 · 75 帖 · 59 人 · 3 小时前」；最后「读于 15:04」和「刷新」。
- 时间用插件设置的时区；没设时用 hub 告诉插件的用户时区（UI 版本 2 的请求带 `tz`），再没有用 hub 机器的。
- 帖子：标题、「作者 · 时间 · 浏览 · 赞 · 帖 · 人 · 标签」、正文（最多 12000 字，代码块用代码样式，图片写成 [图片]）、前 20 楼里最热的 10 条回复（按站点的 score）、「在网页打开」「返回列表」。
- View 守 Persona 的 `plugin.CheckView`（每页最多 300 条、每段 2000 字、链接只用 https），测试里有同样的检查。
- hub 已经能给 App 画插件页（Persona #1284，UI 插槽 #1476），iOS / Mac 的画法在 Persona #1328 里做；以后可以再贡献一个 `nav_page`（主导航入口），
  但旧 hub 不认识这个插槽、整个插件会装不上，所以现在只有 `plugin_page`。

## 测试

- `go test ./linuxdo`：RSS 解析、过滤、正文转文字（列表、代码、图片、表情、头像、链接预览、引用）、热门回复的排序、页面（列表、切换、看全部、帖子、返回、刷新、缓存、帖子不存在、长帖截断）、
  站点的防火墙页 / 429 / 缓存兜底和连接器状态、工具、id 和链接、经协议跑一遍（握手、ui.render / ui.action、工具、`no_contribution`、`unknown_method`）、清单和代码一致，UI 版本 2 的分段控件和用户时区。
- 2026-11-04 对着 Persona master（f850c8b2）的 hub 代码另跑过一遍（不进仓库，插件编成指向假站点）：`dir:` 预览（权限两项：运行程序、访问 linux.do）、安装（不用设置就能跑）、
  表单（「打开时看」是选择器，不在选项里的值被 hub 拒绝）、hub 列出这一页、经 hub 画列表和打开帖子（过 `plugin.CheckView`）、连接器 `ok`、经 hub 调 `linuxdo_hot`。
- 同一天用编好的程序经机房出口读真实站点：热门、今日最热、本周最热、一个帖子都读到（没有被防火墙拦），连接器状态 `ok`；看到热门最前面是置顶公告后加了 `hide_pinned`。

## 打包、安装

```sh
./scripts/package.sh linuxdo          # dist/linuxdo-<版本>.tar.gz
# hub 机器上
personad plugin install file:/path/to/linuxdo-<版本>.tar.gz
```
