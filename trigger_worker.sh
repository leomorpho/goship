#!/usr/bin/env bash
set -euo pipefail

PANE='%0'

tail_pane() {
  tmux capture-pane -p -J -t "$PANE" -S -120
}

is_ready() {
  tail_pane | grep -q '__READY__'
}

send_prompt() {
  tmux load-buffer /tmp/codex-prompt.txt
  tmux paste-buffer -t "$PANE"
  tmux send-keys -t "$PANE" Enter
}

# initial kick
send_prompt

while true; do
  if is_ready; then
    send_prompt
  fi
  sleep 60
done
