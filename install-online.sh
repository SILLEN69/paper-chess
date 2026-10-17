#!/bin/sh
# Pinned release bootstrap; download fully before executing this script.
set -eu
umask 077
VERSION=v0.1.2
REPOSITORY=SILLEN69/paper-chess
ARCHIVE=paper-chess-remarkable2.tar.gz
SHA256=4a1dc4f629cd48b58617499db71ab47eda6d4df387f0200827219b3ef2226c72
fail() { printf '%s\n' "Paper Chess: $*" >&2; exit 1; }
[ "$(id -u)" = 0 ] || fail 'SSH into the reMarkable as root first.'
[ "$(uname -m)" = armv7l ] || fail 'Only reMarkable 2 (ARMv7) is supported.'
[ "$(sed -n 's/REMARKABLE_RELEASE_VERSION=//p' /usr/share/remarkable/update.conf)" = 3.29.0.153 ] || fail 'Only firmware 3.29.0.153 is verified. Nothing installed.'
[ "$(sha256sum /usr/bin/xochitl | cut -d' ' -f1)" = 53547bc30584b9057d5b05ada6c5fb2df3561c3a097c1c2b1aae527d5ddd2e26 ] || fail 'The writing application differs from the verified version. Nothing installed.'
for command in wget sha256sum tar mktemp; do
    command -v "$command" >/dev/null 2>&1 || fail "Missing required tool: $command. See the manual install guide."
done
[ "$(df -Pk /home/root | awk 'END {print $4}')" -ge 50000 ] || fail 'At least 50 MB of free space is required.'
STAGING=$(mktemp -d /home/root/.paper-chess-download.XXXXXX)
# STAGING is created by mktemp under this fixed data-directory path.
trap 'rm -rf "$STAGING"' EXIT
trap 'exit 1' HUP INT TERM
printf '%s\n' "Downloading Paper Chess $VERSION..."
attempt=0
until wget -T 60 -O "$STAGING/$ARCHIVE" "https://github.com/$REPOSITORY/releases/download/$VERSION/$ARCHIVE"; do
    attempt=$((attempt+1))
    [ "$attempt" -lt 3 ] || fail 'Download failed after three attempts. Nothing installed.'
    echo 'Download interrupted; retrying...' >&2
    sleep 2
done
printf '%s  %s\n' "$SHA256" "$ARCHIVE" > "$STAGING/checksum"
(cd "$STAGING" && sha256sum -c checksum) || fail 'Download checksum mismatch. Nothing installed.'
tar -xzf "$STAGING/$ARCHIVE" -C "$STAGING"
sh "$STAGING/paper-chess/install.sh"
