package main

import "strings"

type texts struct {
	files, search, searchHint, noFiles, missingConfig string
	create, review, path, text, noOverwrite, back     string
}

var locales = map[string]*texts{
	"en": {
		files: "Nextcloud files", search: "Search your files by filename", searchHint: "Search for a file, or create a new Markdown note.", noFiles: "No files matched.", missingConfig: "Set the Nextcloud instance URL, username and app password in Settings › Extensions first.",
		create: "Create Markdown note", review: "Review", path: "New relative path (for example Notes/reading.md)", text: "Markdown content", noOverwrite: "The target must be new. If-None-Match:* prevents overwriting an existing file.", back: "Back",
	},
	"ja": {
		files: "Nextcloud ファイル", search: "ファイル名で検索", searchHint: "ファイルを検索するか、新しい Markdown ノートを作成します。", noFiles: "一致するファイルはありません。", missingConfig: "設定 › 拡張機能で Nextcloud のインスタンス URL、ユーザー名、アプリパスワードを設定してください。",
		create: "Markdown ノートを作成", review: "確認", path: "新しい相対パス（例 Notes/reading.md）", text: "Markdown 本文", noOverwrite: "新しいパスが必要です。If-None-Match:* により既存ファイルを上書きしません。", back: "戻る",
	},
	"zh-Hans": {
		files: "Nextcloud 文件", search: "按文件名搜索自己的文件", searchHint: "搜索文件，或新建 Markdown 笔记。", noFiles: "没有匹配的文件。", missingConfig: "先在「设置 › 扩展」中填写 Nextcloud 实例地址、用户名和应用密码。",
		create: "新建 Markdown 笔记", review: "预览", path: "新的相对路径（例如 Notes/reading.md）", text: "Markdown 正文", noOverwrite: "目标必须是新文件。If-None-Match:* 会阻止覆盖已有文件。", back: "返回",
	},
}

func textsFor(locale string) *texts {
	if t := locales[locale]; t != nil {
		return t
	}
	if i := strings.IndexAny(locale, "-_"); i > 0 {
		if t := locales[locale[:i]]; t != nil {
			return t
		}
	}
	return locales["en"]
}
