# GitPicker ⛏️

**GitPicker** is a "vibe-coded" terminal interface designed to make your daily Git workflow significantly nicer. 

Stop copy-pasting long file paths for `git add` or `git diff`. GitPicker gives you an interactive list of your changed files, allowing you to review changes and stage/unstage them with single keystrokes, all while staying inline within your terminal history.

## ✨ Features

* **Interactive List:** Navigate through modified, added, deleted, and untracked files using arrow keys or Vim bindings (`j`/`k`).
* **Smart Diffing:**
    * Press `Enter` to view a diff in your native pager (like `less`).
    * Works perfectly with **untracked files** (shows the full file content as a diff).
    * Automatically distinguishes between staged (`--cached`) and unstaged changes.
* **Rapid Staging:** Toggle "Add Mode" to stage files instantly with `Enter`.
* **Smart Unstaging:** Press `u` to unstage files. It automatically handles the difference between restoring modified files (`git restore --staged`) and removing new files (`git rm --cached`).
* **Inline Experience:** Unlike many TUIs that clear your screen, GitPicker runs inline (except for the diff view), preserving your terminal context and history.

## 🛠️ Installation

### Prerequisites
You need to have [Go](https://go.dev/dl/) installed on your machine (Mac or Linux).

### Option 1: Install via `go install` (Recommended)

If you have your Go path set up, you can install it directly:

```bash
go install [github.com/YOUR_USERNAME/gitpicker@latest](https://github.com/YOUR_USERNAME/gitpicker@latest)
