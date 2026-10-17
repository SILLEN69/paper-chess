# Build from source

## Tools

- Go 1.23 or newer (release tested with Go 1.27.2).
- Zig **0.15.2** for the ARMv7 bridge cross-compile.
- Python 3.10+ for packaging and installer checks.
- Qt **6.10.3** `qmllint` for the native QML version.

Chess rules are vendored as a local Go module. Building does not require downloading Go modules. The source tree deliberately excludes the official writing executable, Qt libraries, device credentials and notebook files.

## Check and compile

From the repository root, on Linux/macOS with a suitable shell:

```sh
go test ./...
go vet ./...
python3 -m unittest discover -s tests -v
qmllint Main.qml

GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 \
  go build -trimpath -ldflags '-s -w' -o paper-chess
zig cc -target arm-linux-gnueabihf.2.28 -mcpu=cortex_a7 \
  -shared -fPIC -O2 -Wall -s bridge.c -o chess-ui.so -ldl -lpthread
python3 build_release.py dist/paper-chess-remarkable2.tar.gz
```

`qmllint` needs the QtQuick import path for Qt 6.10.3 if it cannot find it automatically (`-I /path/to/qt/qml`). The tablet is ARMv7 hard-float with an older glibc; do not replace the Zig target with a host build.

On Windows use PowerShell environment variables for the Go target, and the same Zig arguments. The installer tests use a POSIX shell; they also support Git for Windows' `sh.exe` through `PAPER_CHESS_TEST_SHELL`.

The packager uses an explicit file list, fixes Unix file modes, writes LF checksums and verifies each runtime file inside the finished archive. It includes source and licenses. It does not package saves or arbitrary files in the working directory.

## Install a local build

Copy the archive and `.sha256` file to the tablet, verify with `sha256sum -c`, then follow the manual install in [INSTALL.md](INSTALL.md). A different firmware is unsupported even if compilation succeeds.

## Publishing a release

1. Run the checks and build both ARMv7 binaries.
2. Generate the release archive and checksum.
3. Set `VERSION` and `SHA256` in `install-online.sh` to that exact release and archive digest. The bootstrap is outside the archive to avoid a circular hash dependency.
4. Update pinned install links in the README and install guide, then commit and tag.
5. Publish the archive, checksum and bootstrap as release assets. Verify the public download digest before announcing the release.

The bootstrap is pinned to a version, not `latest`. Do not overwrite an existing release asset with different bytes; publish a new version instead.
