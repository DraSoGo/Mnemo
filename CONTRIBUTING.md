# 🚀 Contributing to Mnemo

First off — thanks for taking the time to consider contributing! ❤️

Mnemo is a small project with a clear scope: make zsh history search and AI completion fast, reliable, and unsurprising. Contributions that share that philosophy are very welcome.

---

## 📑 Table of Contents

- [Code of Conduct](#-code-of-conduct)
- [How Can I Contribute?](#-how-can-i-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Features](#suggesting-features)
  - [Pull Requests](#pull-requests)
- [Development Setup](#-development-setup)
- [Project Structure](#-project-structure)
- [Coding Standards](#-coding-standards)
  - [Go](#go)
  - [Zsh](#zsh)
- [Testing](#-testing)
- [Commit Messages](#-commit-messages)
- [Pull Request Checklist](#-pull-request-checklist)
- [Recognition](#-recognition)

---

## 🤝 Code of Conduct

Be excellent to each other. We follow the spirit of the [Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/): respectful, focused on the work, no harassment of any kind. Disagreements about technical direction are welcome — disagreements about people are not.

---

## 🛠 How Can I Contribute?

### Reporting Bugs

A good bug report saves everyone time. Please open an issue using the **Bug Report** template and include:

- **Environment**: zsh version (`zsh --version`), OS, terminal emulator, oh-my-zsh version (if used).
- **Plugins**: list of zsh plugins loaded around Mnemo (especially `zsh-autosuggestions`, `zsh-syntax-highlighting`).
- **Steps to reproduce**: the exact key sequence and command line.
- **Expected vs. actual behavior**: what should have happened, what did happen.
- **Logs**: relevant output from `mnemo predict "..."`, `mnemo pick`, or `mnemo warmup`.
- **Screenshot or recording**: terminal output is hard to convey in text — a screenshot helps a lot.

> 🔎 Before filing, search existing [issues](https://github.com/drasogun/mnemo/issues) — yours may already be tracked.

### Suggesting Features

Open a **Feature Request** issue with:

1. **The problem** you're trying to solve (not the solution).
2. **Why existing features can't solve it.**
3. **A proposed user experience** (keybinding, behavior, configuration).
4. **Tradeoffs** you've considered.

We tend to say no to features that add hooks, async polling, or state that survives across keystrokes — they're the source of most past bugs. Features that fit cleanly into the trigger-key + Go-binary model are easier to merge.

### Pull Requests

PRs are very welcome. For anything beyond a typo or one-line fix, please **open an issue first** so we can agree on the approach before you spend time on code.

---

## 💻 Development Setup

### Prerequisites

- **Go** 1.22 or newer ([install](https://go.dev/dl/))
- **zsh** 5.3 or newer (for `add-zle-hook-widget`)
- **Ollama** (only if you're working on the predict path) — see [README](README.md#-brain-setup-ollama-optional)

### Get the Code

```bash
git clone https://github.com/drasogun/mnemo.git
cd mnemo
```

### Build & Test

```bash
go build -o mnemo .
go test ./...
```

### Live Test in a Real Shell

```bash
# Symlink (or copy) to your oh-my-zsh plugin dir, then reload:
ln -sfn "$PWD" "${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}/plugins/mnemo"
source ~/.zshrc
```

After every code change:

```bash
go build -o mnemo .
source ~/.oh-my-zsh/custom/plugins/mnemo/mnemo.plugin.zsh
```

---

## 📂 Project Structure

```
.
├── main.go              — entrypoint + subcommand dispatch
├── history.go           — zsh_history parser, dedupe, ordering
├── history_test.go      — parser unit tests
├── tui.go               — Bubble Tea picker model + view
├── predict.go           — Ollama HTTP client + warmup
├── mnemo.plugin.zsh     — zsh widgets + keybindings
├── go.mod / go.sum      — Go module manifest
├── README.md            — user-facing docs
├── CONTRIBUTING.md      — this file
└── LICENSE              — GPL-3.0
```

**Module responsibilities — please respect these boundaries:**

- `main.go` owns CLI dispatch only. It must not know about Ollama, history, or Bubble Tea internals.
- `history.go` owns history reading and ordering. It must not import the TUI.
- `tui.go` owns rendering and keystroke handling for the picker. It must not read files or talk to Ollama.
- `predict.go` owns HTTP I/O with Ollama. It must not draw to the terminal.
- The zsh plugin owns line-editor mutation. It must not parse history or call HTTP — those go through the binary.

If a change crosses these lines, expect a request to refactor.

---

## ✏️ Coding Standards

### Go

- Run `go fmt ./...` before committing — CI rejects unformatted code.
- Run `go vet ./...` — no warnings.
- Prefer the standard library. New runtime dependencies need a strong justification.
- Errors must be returned, not logged-and-swallowed. Top-level handlers in `main.go` are the only place that prints to stderr and calls `os.Exit`.
- Public exported names need a `// Foo does X.` doc comment. Internal helpers do not.
- Avoid `init()` functions. Initialization belongs in constructors called from `main`.
- No global mutable state (the picker model is value-typed for this reason).

### Zsh

- Every widget function starts with `emulate -L zsh` to avoid surprises from the user's options.
- All function and variable names are namespaced with `_mnemo_` / `_MNEMO_`.
- Wrap built-in widgets (e.g. `self-insert`) only with re-source guards:

  ```zsh
  if ! zle -l _mnemo_orig_self_insert &>/dev/null; then
      zle -A self-insert _mnemo_orig_self_insert
  fi
  ```

  Without this, `source ~/.zshrc` twice causes infinite recursion.
- Never call `zle -M` directly — go through `_mnemo_msg` so we can clear our own messages without clobbering other plugins.
- Don't add `zle-line-pre-redraw` hooks. They were the source of every bug in the prototype phase.

---

## 🧪 Testing

### Go Tests

```bash
go test ./...
go test -run TestStripExtended       # run a single test
go test -cover ./...                 # coverage report
```

Test files live next to the code they test (`history_test.go` next to `history.go`). New parsing/HTTP code requires tests. UI code may be exercised manually.

### Manual Test Checklist

After any change to the plugin or binary, run through:

1. `Ctrl+R` opens the picker. Type to filter. Up/Down navigates. Enter fills buffer. Esc cancels.
2. `Ctrl+F` with Ollama running shows ghost text within 5 s (warm) or up to 30 s (cold). `Tab` accepts. `→` accepts at boundary. Any other key clears.
3. `Ctrl+F` with Ollama **down** shows `[Ollama unavailable …]` once and clears on the next keystroke (no message bleed).
4. Re-run `source ~/.zshrc`. Plugin loads without errors. No infinite recursion.
5. With `zsh-autosuggestions` active: AI ghost suppresses autosuggestions; clearing AI ghost re-enables them.
6. `mnemo warmup` runs in <1 s when Ollama already has the model loaded.

---

## 📝 Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/):

```
<type>(<scope>): <short description>

<optional body explaining the why>

<optional footer with breaking-change notes or issue refs>
```

**Common types:**

| Type     | Use for |
|----------|---------|
| `feat`   | New user-visible feature |
| `fix`    | Bug fix |
| `refactor` | Code change that neither adds a feature nor fixes a bug |
| `docs`   | README / CONTRIBUTING / comment changes only |
| `test`   | Adding or fixing tests |
| `chore`  | Build, deps, repo housekeeping |

**Examples:**

```
feat(predict): add MNEMO_OLLAMA_URL env override
fix(plugin): clear stale zle -M message on self-insert
refactor(tui): extract scroll math into helper
docs(readme): document Ctrl+F workflow
```

Keep the subject line under 72 characters. Use the body to explain the **why**, not the **what** — the diff already shows what changed.

---

## ✅ Pull Request Checklist

Before requesting review:

- [ ] `go fmt ./...` — formatted
- [ ] `go vet ./...` — no warnings
- [ ] `go test ./...` — all green
- [ ] `zsh -n mnemo.plugin.zsh` — syntax OK
- [ ] Manual test checklist above — all six steps pass
- [ ] README / CONTRIBUTING updated if behavior changed
- [ ] Commit messages follow Conventional Commits
- [ ] No new runtime dependencies added without discussion
- [ ] PR description references the related issue (`Closes #123`)

---

## 🌟 Recognition

Contributors are listed in release notes and in the repository's contributor graph. Significant or recurring contributors may be invited as maintainers.

Thank you for helping make Mnemo better! 🚀
