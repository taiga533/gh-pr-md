# gh-pr-md

[English](README.md) | 日本語

[GitHub CLI](https://cli.github.com/) の拡張機能です。プルリクエストの**全体情報** — レビューコメント、インラインコメント（diff hunk付き）、suggested changes を含む — をMarkdown形式でターミナルに出力します。

**AI Agent** と**人間**の両方が、プルリクエストの全体像を一つのMarkdownストリームで把握できるように設計されています。

## インストール

```bash
gh extension install taiga533/gh-pr-md
```

> [GitHub CLI](https://cli.github.com/) (`gh`) v2.0+ が必要です

## 使い方

```bash
gh pr-md [<number> | <url> | <branch>]
```

**引数**（すべて省略可能 — 省略時は現在のブランチから自動推定）:

| 引数 | 例 |
|------|-----|
| PR番号 | `gh pr-md 123` |
| GitHub URL | `gh pr-md https://github.com/owner/repo/pull/123` |
| ブランチ名 | `gh pr-md feature-branch` |

**フラグ:**

| フラグ | 説明 |
|--------|------|
| `-R`, `--repo [HOST/]OWNER/REPO` | 対象リポジトリを指定 |
| `--no-diff` | diff hunkを省略し、ファイル参照のみ表示（例: `abc123@path/to/file.go#L10-20`） |
| `--no-color` | ANSIカラー出力を無効化 |

**パイプ対応:** 標準出力がTTYでない場合（パイプやファイルリダイレクト時）、レンダリングなしのプレーンMarkdownを出力します。AIエージェントや他のツールへの入力に最適です。

```bash
# AIエージェントに渡す
gh pr-md 123 | your-ai-tool

# ファイルに保存
gh pr-md 123 > pr-review.md
```

## なぜ gh-pr-md？

既存の `gh pr view --comments` には大きな盲点があります: **レビューコメントやインラインコードコメントを取得できません。** コードレビューで最も重要なのは、行単位のフィードバック、suggested changes、そしてスレッド化された議論です — PRをマージに導くこれらの情報が欠落しています。

`gh pr-md` はこのギャップを埋めます:

- **レビューコメント & インラインコメント** — コードの特定行に付けられたコメントを含む、すべてのコメントタイプを取得します。
- **diff hunk** — インラインコメントが付けられた箇所のコードコンテキストを、関連行に絞って表示します。
- **suggested changes** — GitHubの ` ```suggestion ` ブロックを明確にレンダリングし、提案内容を確認できます。
- **時系列タイムライン** — issueコメント、レビュー、インラインコメントを一つの時系列ストリームに統合します。
- **スレッド表示** — インラインコメントの返信チェーンをグループ化し、会話の流れを自然に読めます。
- **ターミナルレンダリング** — [Glamour](https://github.com/charmbracelet/glamour) によるシンタックスハイライト付きのスタイル付きMarkdown表示。
- **プレーンMarkdown出力** — パイプ時は生のMarkdownを出力。AIエージェントやLLMが直接パースできます。

## 出力フォーマット

出力は以下の構造になります:

```
---
number: 123
title: "Add feature X"
author: username
assignees: reviewer1, reviewer2
---

## Description

PRの本文...

---

**@commenter** commented on 2024-01-15 10:30:00

コメント本文...

---

**@reviewer** requested changes on 2024-01-15 11:00:00

レビュー本文...

> **@reviewer** commented on `path/to/file.go`
>
> ```diff
>  func example() {
> -    old code
> +    new code
>  }
> ```
>
> インラインコメント本文...
```

## ライセンス

[MIT](LICENSE)
