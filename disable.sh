#!/bin/sh
set -eu
systemctl disable --now paper-chess-shortcut.service
/home/root/paper-chess/session recover
systemctl stop paper-chess-backend.service || true
echo 'Normal interface restored. Chess is disabled; your saved game is preserved.'
