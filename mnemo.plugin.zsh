# mnemo: zsh integration for the Go binary.
#   Ctrl+R  → fuzzy history picker (TUI)
#   Alt+A   → Ollama AI ghost-text prediction (inline)
#   Tab     → accept ghost (or normal completion)
#   Right   → accept ghost when at boundary (or normal forward-char)
#
# Binary lookup order:
#   1. $MNEMO_BIN
#   2. `mnemo` on $PATH
#   3. ./mnemo next to this plugin file

autoload -Uz add-zle-hook-widget

typeset -g _MNEMO_PLUGIN_DIR="${0:A:h}"

# AI ghost-text state
typeset -g _MNEMO_AI_GHOST_LEN=0
typeset -g _MNEMO_AI_RH_ENTRY=""
typeset -g _MNEMO_MSG_ACTIVE=0   # we currently own the zle -M area

_mnemo_resolve_bin() {
    if [[ -n "$MNEMO_BIN" && -x "$MNEMO_BIN" ]]; then
        print -r -- "$MNEMO_BIN"
        return
    fi
    if (( ${+commands[mnemo]} )); then
        print -r -- "${commands[mnemo]}"
        return
    fi
    if [[ -x "${_MNEMO_PLUGIN_DIR}/mnemo" ]]; then
        print -r -- "${_MNEMO_PLUGIN_DIR}/mnemo"
        return
    fi
    return 1
}

_mnemo_clear_ai_ghost() {
    (( _MNEMO_AI_GHOST_LEN == 0 )) && return
    BUFFER="${BUFFER:0:$(( ${#BUFFER} - _MNEMO_AI_GHOST_LEN ))}"
    [[ -n "$_MNEMO_AI_RH_ENTRY" ]] && \
        region_highlight=("${(@)region_highlight:#$_MNEMO_AI_RH_ENTRY}")
    _MNEMO_AI_GHOST_LEN=0
    _MNEMO_AI_RH_ENTRY=""
    (( ${+functions[_zsh_autosuggest_enable]} )) && _zsh_autosuggest_enable
}

_mnemo_accept_ai_ghost() {
    (( _MNEMO_AI_GHOST_LEN == 0 )) && return 1
    CURSOR=${#BUFFER}
    [[ -n "$_MNEMO_AI_RH_ENTRY" ]] && \
        region_highlight=("${(@)region_highlight:#$_MNEMO_AI_RH_ENTRY}")
    _MNEMO_AI_GHOST_LEN=0
    _MNEMO_AI_RH_ENTRY=""
    (( ${+functions[_zsh_autosuggest_enable]} )) && _zsh_autosuggest_enable
    return 0
}

# Clear our own zle -M message (only if we set one).
_mnemo_clear_msg() {
    (( _MNEMO_MSG_ACTIVE == 0 )) && return
    zle -M ""
    _MNEMO_MSG_ACTIVE=0
}

# Wrapper around `zle -M` that also marks ownership.
_mnemo_msg() {
    zle -M "$1"
    _MNEMO_MSG_ACTIVE=1
}

# ── Ctrl+R: fuzzy history picker ──────────────────────────────────────────────

_mnemo_pick() {
    emulate -L zsh
    _mnemo_clear_ai_ghost
    _mnemo_clear_msg

    local bin
    bin=$(_mnemo_resolve_bin) || {
        _mnemo_msg "  mnemo: binary not found (set \$MNEMO_BIN or build it)"
        return
    }

    local selected
    selected=$("$bin" "$BUFFER" </dev/tty)
    local rc=$?

    zle reset-prompt
    if (( rc == 0 )) && [[ -n "$selected" ]]; then
        BUFFER="$selected"
        CURSOR=${#BUFFER}
    fi
}
zle -N _mnemo_pick

# ── Alt+A: Ollama AI ghost-text prediction ────────────────────────────────────

_mnemo_predict() {
    emulate -L zsh
    [[ -z "$BUFFER" ]] && return

    local bin
    bin=$(_mnemo_resolve_bin) || {
        _mnemo_msg "  mnemo: binary not found"
        return
    }

    _mnemo_clear_ai_ghost
    _mnemo_clear_msg

    local real_buf="$BUFFER"
    _mnemo_msg "  [Asking Ollama...]"
    zle -R

    local prediction
    prediction=$("$bin" predict "$real_buf" 2>/dev/null)
    local rc=$?
    _mnemo_clear_msg

    if (( rc != 0 )) || [[ -z "$prediction" ]]; then
        _mnemo_msg "  [Ollama unavailable — run: ollama serve]"
        return
    fi

    local ghost
    if [[ "$prediction" == ${real_buf}* ]]; then
        ghost="${prediction:${#real_buf}}"
    else
        # Prediction diverges — show as tab-prefixed alternative
        ghost=$'\t'"${prediction}"
    fi
    [[ -z "$ghost" ]] && return

    local buf_pos=${#BUFFER}
    BUFFER+="$ghost"
    _MNEMO_AI_GHOST_LEN=${#ghost}
    _MNEMO_AI_RH_ENTRY="${buf_pos} $(( buf_pos + ${#ghost} )) fg=14,dim"
    region_highlight+=("$_MNEMO_AI_RH_ENTRY")
    CURSOR=$buf_pos

    (( ${+functions[_zsh_autosuggest_disable]} )) && _zsh_autosuggest_disable
}
zle -N _mnemo_predict

# ── Tab: accept ghost or fall through to normal completion ────────────────────

_mnemo_tab() {
    emulate -L zsh
    if _mnemo_accept_ai_ghost; then
        return
    fi
    zle expand-or-complete
}
zle -N _mnemo_tab

# ── Right arrow: accept ghost at boundary or normal forward-char ──────────────

_mnemo_right() {
    emulate -L zsh
    if (( _MNEMO_AI_GHOST_LEN > 0 && CURSOR == ${#BUFFER} - _MNEMO_AI_GHOST_LEN )); then
        _mnemo_accept_ai_ghost
        return
    fi
    zle forward-char
}
zle -N _mnemo_right

# ── Wrap edit widgets to clear ghost on user input ────────────────────────────

if ! zle -l _mnemo_orig_self_insert &>/dev/null; then
    zle -A self-insert _mnemo_orig_self_insert
fi
_mnemo_self_insert() {
    _mnemo_clear_ai_ghost
    _mnemo_clear_msg
    zle _mnemo_orig_self_insert
}
zle -N self-insert _mnemo_self_insert

if ! zle -l _mnemo_orig_backward_delete &>/dev/null; then
    zle -A backward-delete-char _mnemo_orig_backward_delete
fi
_mnemo_backward_delete() {
    _mnemo_clear_ai_ghost
    _mnemo_clear_msg
    zle _mnemo_orig_backward_delete
}
zle -N backward-delete-char _mnemo_backward_delete

# ── Cleanup on Enter ──────────────────────────────────────────────────────────

_mnemo_line_finish() {
    _mnemo_clear_ai_ghost
    _mnemo_clear_msg
}
add-zle-hook-widget -Uz zle-line-finish _mnemo_line_finish

# ── Keybindings ───────────────────────────────────────────────────────────────

: ${MNEMO_KEYBIND:='^R'}
bindkey "$MNEMO_KEYBIND" _mnemo_pick

: ${MNEMO_PREDICT_KEY:='\ea'}
bindkey "$MNEMO_PREDICT_KEY" _mnemo_predict

bindkey '^I'   _mnemo_tab     # Tab
bindkey '\e[C' _mnemo_right   # Right arrow (standard)
bindkey '\eOC' _mnemo_right   # Right arrow (application/keypad)

# ── Optional: warm Ollama model on shell startup so first Alt+A is fast ───────
# Set MNEMO_WARMUP=0 to disable.
: ${MNEMO_WARMUP:=1}
_mnemo_warmup_bg() {
    local bin
    bin=$(_mnemo_resolve_bin) || return
    "$bin" warmup &>/dev/null &!
}
if (( MNEMO_WARMUP )); then
    _mnemo_warmup_bg
fi
