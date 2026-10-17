# Install and recovery

## Before installing

Use a reMarkable 2 running **3.29.0.153** with the original writing application. Other tablets and firmware are refused. There is no force flag. Do not downgrade or reset your device for this project.

Back up your notebooks, charge the tablet, and connect it to Wi-Fi. Close the notebook you are writing in. You need working root SSH access and about **50 MB of free space**. Installation downloads about 3 MB; Wi-Fi is only needed for the download.

Find your SSH password and address in the tablet's settings. From a computer:

```sh
ssh root@YOUR_TABLET_IP
```

The USB-network address is usually `10.11.99.1` when the tablet is connected by USB. Use the address shown by your device. Keep your password private.

## One command on the tablet

```sh
wget -q -T 60 -O /tmp/paper-chess-install.sh https://raw.githubusercontent.com/SILLEN69/paper-chess/v0.1.1/install-online.sh && sh /tmp/paper-chess-install.sh
```

The full script downloads before execution. Its release URL and expected SHA256 are pinned to `v0.1.1`. It checks the device/firmware, downloads with HTTPS and retries, verifies the archive, then calls the local installer. It does not use an unverified "latest" archive.

The local installer verifies runtime checksums, stages program files on the data partition, saves the previous program/services during setup, and checks that the shortcut service starts. A normal setup failure triggers rollback. This is not protection against power loss or a forcibly killed installer; keep the tablet charged.

Re-running the same command reinstalls that release and preserves saved games in `/home/root/.local/share/paper-chess`. It leaves the tablet in Notes. Only the shortcut listener is enabled at boot; the chess UI is opened manually.

## Playing

Triple-press the power button within two seconds. Tap **New game** for computer or local two-player mode. **Learn** has nine beginner lessons. Tap **Notes** to exit; repeating the shortcut will not exit chess.

Reopening waits for the native interface to initialize, usually about 20–30 seconds. Normal auto-sleep applies. Notebook cloud sync pauses while chess is open and resumes in Notes.

## Offline/manual installation

On a computer, download `paper-chess-remarkable2.tar.gz` and its `.sha256` file from the [v0.1.1 release](https://github.com/SILLEN69/paper-chess/releases/tag/v0.1.1). Copy both to `/home/root/` using your SSH client, then on the tablet:

```sh
cd /home/root
sha256sum -c paper-chess-remarkable2.tar.gz.sha256
# Continue only if verification says OK.
mkdir -p paper-chess-unpack
tar -xzf paper-chess-remarkable2.tar.gz -C paper-chess-unpack
sh paper-chess-unpack/paper-chess/install.sh
```

The release archive includes executables. The GitHub **Source code** zip does not: build that source first using [BUILD.md](BUILD.md).

## Disable or remove

Return to Notes and disable the shortcut while keeping your game:

```sh
/home/root/paper-chess/disable.sh
```

To remove program files after disabling, run the following exact paths on the tablet:

```sh
systemctl disable --now paper-chess-shortcut.service
/home/root/paper-chess/session recover
systemctl stop paper-chess-backend.service
rm -f /etc/systemd/system/paper-chess-backend.service /etc/systemd/system/paper-chess-shortcut.service /etc/systemd/system/paper-chess-ui.service
systemctl daemon-reload
rm -rf /home/root/paper-chess
```

Your chess saves remain in `/home/root/.local/share/paper-chess`; your notebooks are elsewhere and are not removed. No notebook backup restore is needed to remove chess. The original writing executable and its service are not patched by this installer.

## If something fails

- **Unsupported firmware / writing application differs:** stop. A port needs new native ABI validation. Do not bypass the checks.
- **Download failed:** check Wi-Fi and retry. Nothing is installed until the full verified archive is available.
- **Checksum mismatch:** do not extract or run the download. Retry from the pinned release.
- **Read-only service directory:** stop and report your firmware. The installer does not automatically remount system partitions.
- **Old launcher detected:** reboot into Notes before reinstalling.
- **Shortcut does nothing:** wake the tablet, wait for Notes to finish loading, then make three short presses within two seconds. Check `journalctl -u paper-chess-shortcut.service -n 40 --no-pager` over SSH.
- **Chess will not close:** run `/home/root/paper-chess/session recover`. A reboot starts the stock interface; the UI service is not enabled at boot.
- **Rollback needs attention:** keep the printed transaction directory. It contains the previous program under `old/` and saved unit files under `units/`. Disable the listener, return to Notes, and inspect the failure before moving files. Do not restore notebooks to repair an app install.

This is experimental software using private Qt internals. A process crash is still possible, even with exact compatibility checks.
