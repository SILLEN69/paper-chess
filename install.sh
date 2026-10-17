#!/bin/sh
# Install a verified release. Game saves live outside the program directory.
set -eu
umask 077
SOURCE=$(CDPATH= cd "$(dirname "$0")" && pwd)
DEST=/home/root/paper-chess
UNITS=/etc/systemd/system
NAMES='paper-chess-backend paper-chess-shortcut paper-chess-ui'
fail() { printf '%s\n' "Paper Chess: $*" >&2; exit 1; }
[ "$(id -u)" = 0 ] || fail 'Run this on the tablet as root.'
[ "$(uname -m)" = armv7l ] || fail 'Only reMarkable 2 (ARMv7) is supported.'
[ -e /dev/input/by-path/platform-30370000.snvs:snvs-powerkey-event ] || fail 'The reMarkable 2 power-button device was not found.'
[ "$(sed -n 's/REMARKABLE_RELEASE_VERSION=//p' /usr/share/remarkable/update.conf)" = 3.29.0.153 ] || fail 'Only firmware 3.29.0.153 is verified. Nothing installed.'
[ "$(sha256sum /usr/bin/xochitl | cut -d' ' -f1)" = 53547bc30584b9057d5b05ada6c5fb2df3561c3a097c1c2b1aae527d5ddd2e26 ] || fail 'The writing application differs from the verified version. Nothing installed.'
for command in systemctl flock mktemp cp mv sha256sum; do
    command -v "$command" >/dev/null 2>&1 || fail "Missing required tool: $command"
done
[ -w "$UNITS" ] || fail 'System service directory is read-only. Nothing installed.'
[ ! -L "$DEST" ] || fail 'The installation directory is a symlink. Nothing installed.'
if [ -e "$DEST" ]; then [ -d "$DEST" ] || fail 'The installation path is not a directory.'; fi
[ "$(df -Pk /home/root | awk 'END {print $4}')" -ge 50000 ] || fail 'At least 50 MB of free space is required.'
(cd "$SOURCE" && sha256sum -c SHA256SUMS >/dev/null) || fail 'Release checksum verification failed. Nothing installed.'
for name in paper-chess chess-ui.so Main.qml session install.sh disable.sh $NAMES; do
    case "$name" in paper-chess-*) name=$name.service ;; esac
    [ -f "$SOURCE/$name" ] && [ ! -L "$SOURCE/$name" ] || fail "Missing or unsafe release file: $name"
done
for name in paper-chess.conf zz-paper-chess.conf; do
    [ ! -e "/run/systemd/system/xochitl.service.d/$name" ] || fail 'Reboot to Notes before upgrading the old chess launcher.'
done

# Serialize installers separately from the session controller's own lock.
exec 8>/home/root/.paper-chess-install.lock
flock -n 8 || fail 'Another Paper Chess installation is already running.'
WORK=$(mktemp -d /home/root/.paper-chess-install.XXXXXX)
changed=0; had_old=0; shortcut_enabled=0; committed=0
systemctl is-enabled --quiet paper-chess-shortcut.service && shortcut_enabled=1 || true
cleanup() {
    result=$?
    trap - EXIT HUP INT TERM
    if [ "$committed" = 0 ] && [ "$changed" = 1 ]; then
        echo 'Setup failed. Restoring the previous installation and Notes.' >&2
        set +e
        systemctl disable --now paper-chess-shortcut.service >/dev/null 2>&1
        systemctl stop paper-chess-ui.service paper-chess-backend.service >/dev/null 2>&1
        restored=1
        if [ -d "$WORK/old" ]; then
            [ ! -e "$DEST" ] || mv "$DEST" "$WORK/failed" || restored=0
            mv "$WORK/old" "$DEST" || restored=0
        elif [ "$had_old" = 0 ] && [ -e "$DEST" ]; then
            mv "$DEST" "$WORK/failed" || restored=0
        fi
        for name in $NAMES; do
            rm -f "$UNITS/$name.service" || restored=0
            if [ -e "$WORK/units/$name.service" ] || [ -L "$WORK/units/$name.service" ]; then
                cp -P "$WORK/units/$name.service" "$UNITS/$name.service" || restored=0
            fi
        done
        systemctl daemon-reload || restored=0
        systemctl start xochitl.service || restored=0
        if [ "$shortcut_enabled" = 1 ]; then
            systemctl enable --now paper-chess-shortcut.service || restored=0
        fi
        if [ "$restored" != 1 ]; then
            echo "Recovery needs attention. Backup retained at $WORK. See docs/INSTALL.md." >&2
            exit 1
        fi
    fi
    # WORK is created above with mktemp in this fixed directory, never user input.
    rm -rf "$WORK"
    exit "$result"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

mkdir "$WORK/new" "$WORK/units"
cp -R "$SOURCE/." "$WORK/new/"
chmod 755 "$WORK/new" "$WORK/new/assets" "$WORK/new/paper-chess" "$WORK/new/session" "$WORK/new/install.sh" "$WORK/new/disable.sh"
chmod 644 "$WORK/new/Main.qml" "$WORK/new/chess-ui.so" "$WORK/new"/*.service "$WORK/new/assets"/*.svg
for name in $NAMES; do
    if [ -e "$UNITS/$name.service" ] || [ -L "$UNITS/$name.service" ]; then
        cp -P "$UNITS/$name.service" "$WORK/units/$name.service"
    fi
done
[ ! -e "$DEST" ] || had_old=1
changed=1
systemctl disable --now paper-chess-shortcut.service >/dev/null 2>&1 || true
if [ -x "$DEST/session" ]; then "$DEST/session" recover; fi
systemctl stop paper-chess-backend.service 2>/dev/null || true
if [ "$had_old" = 1 ]; then mv "$DEST" "$WORK/old"; fi
mv "$WORK/new" "$DEST"
for name in $NAMES; do
    rm -f "$UNITS/$name.service"
    cp "$DEST/$name.service" "$UNITS/$name.service"
    chmod 644 "$UNITS/$name.service"
done
systemctl daemon-reload
systemctl enable --now paper-chess-shortcut.service
systemctl is-active --quiet paper-chess-shortcut.service
committed=1
printf '\n%s\n' 'Paper Chess installed.' 'Triple-press power within two seconds to open it. Tap Notes to return.' 'Allow about 20–30 seconds when reopening. Saves are preserved.' 'Disable: /home/root/paper-chess/disable.sh'
