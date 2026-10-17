"""Exercise the real shell scripts against fake device paths and services."""
import hashlib
import os
import re
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import unittest

REPO = Path(__file__).resolve().parents[1]
SHELL = os.environ.get('PAPER_CHESS_TEST_SHELL') or shutil.which('sh')


def shellpath(path):
    path = Path(path).resolve().as_posix()
    return '/' + path[0].lower() + path[2:] if len(path) > 1 and path[1] == ':' else path


@unittest.skipUnless(SHELL, 'A POSIX shell is required')
class InstallerTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix='paper-chess-test-')
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.home = self.root / 'home/root'
        self.units = self.root / 'etc/systemd/system'
        self.package = self.root / 'package'
        self.bin = self.root / 'bin'
        for folder in (self.home, self.units, self.package / 'assets', self.bin):
            folder.mkdir(parents=True)
        (self.root / 'button').touch()
        self.config = self.root / 'usr/share/remarkable/update.conf'
        self.config.parent.mkdir(parents=True)
        self.config.write_text('REMARKABLE_RELEASE_VERSION=3.29.0.153\n')
        self.mock('id', 'echo 0')
        self.mock('uname', 'echo armv7l')
        self.mock('df', "printf 'Filesystem 1024-blocks Used Available Capacity Mounted\\nmock 200000 0 200000 0%% /\\n'")
        self.mock('flock', 'exit 0')
        actual_sha = '/usr/bin/sha256sum' if os.name == 'nt' else shutil.which('sha256sum')
        self.mock('sha256sum', '''case "$1" in
 */usr/bin/xochitl) echo '53547bc30584b9057d5b05ada6c5fb2df3561c3a097c1c2b1aae527d5ddd2e26  xochitl';;
 *) exec "''' + actual_sha + '''" "$@";;
esac''')
        self.mock('systemctl', '''printf '%s\n' "$*" >> "$TEST_LOG"
case "$1" in
 is-enabled) [ "${TEST_OLD_ENABLED:-0}" = 1 ]; exit $?;;
 is-active)
   if [ "${TEST_FAIL_START:-0}" = 1 ] && [ ! -e "$TEST_FAILURE_MARKER" ]; then
     : > "$TEST_FAILURE_MARKER"; exit 1
   fi;;
esac
exit 0''')
        self.mock('wget', '''[ "${TEST_DOWNLOAD_FAIL:-0}" != 1 ] || exit 22
while [ "$#" -gt 0 ]; do
 if [ "$1" = -O ]; then
   shift
   if [ -n "${TEST_DOWNLOAD_FILE:-}" ]; then cp "$TEST_DOWNLOAD_FILE" "$1"; else printf 'wrong archive bytes' > "$1"; fi
   exit 0
 fi
 shift
done
exit 1''')
        self.env = dict(os.environ, TEST_LOG=shellpath(self.root / 'services.log'),
                        TEST_FAILURE_MARKER=shellpath(self.root / 'failed-once'), TEST_BIN=shellpath(self.bin))
        runtime = ['paper-chess', 'chess-ui.so', 'Main.qml', 'session', 'disable.sh',
                   'paper-chess-backend.service', 'paper-chess-shortcut.service',
                   'paper-chess-ui.service', 'assets/wk.svg']
        for name in runtime:
            body = '#!/bin/sh\nexit 0\n' if name in ('session', 'disable.sh', 'paper-chess') else 'fixture\n'
            (self.package / name).write_text(body,newline='\n')
            (self.package / name).chmod(0o755)
        self.adapt('install.sh', self.package / 'install.sh')
        self.adapt('install-online.sh', self.root / 'online.sh')
        names = runtime + ['install.sh']
        sums = ''.join(hashlib.sha256((self.package / n).read_bytes()).hexdigest() + '  ' + n + '\n' for n in names)
        (self.package / 'SHA256SUMS').write_text(sums,newline='\n')

    def mock(self, name, body):
        target = self.bin / name
        target.write_text('#!/bin/sh\n' + body + '\n', newline='\n')
        target.chmod(0o755)

    def adapt(self, name, target):
        text = (REPO / name).read_text().replace('RELEASE_SHA256_PLACEHOLDER','0'*64)
        text = text.replace('/dev/input/by-path/platform-30370000.snvs:snvs-powerkey-event',shellpath(self.root / 'button'))
        for prefix in ('/home/root', '/etc/systemd/system', '/usr/share/remarkable', '/usr/bin/xochitl', '/run/systemd/system'):
            text = text.replace(prefix, shellpath(self.root) + prefix)
        target.write_text(text, newline='\n')

    def run_script(self, online=False):
        script = self.root / 'online.sh' if online else self.package / 'install.sh'
        return subprocess.run([SHELL, '-c', 'export PATH="$TEST_BIN:$PATH"; exec sh "$1"', 'test', shellpath(script)],
                              env=self.env, capture_output=True, text=True, timeout=30)

    def seed_existing(self):
        old = self.home / 'paper-chess'
        old.mkdir()
        (old / 'version').write_text('previous-release')
        (old / 'session').write_text('#!/bin/sh\nexit 0\n',newline='\n')
        (old / 'session').chmod(0o755)
        saves = self.home / '.local/share/paper-chess'
        saves.mkdir(parents=True)
        (saves / 'game.json').write_text('{"sentinel":"keep my game"}')
        for name in ('backend', 'shortcut', 'ui'):
            (self.units / ('paper-chess-' + name + '.service')).write_text('old-' + name)
        self.env['TEST_OLD_ENABLED'] = '1'
        return saves / 'game.json'

    def test_fresh_install_enables_only_shortcut(self):
        result = self.run_script()
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)
        self.assertTrue((self.home / 'paper-chess/Main.qml').exists())
        log = (self.root / 'services.log').read_text()
        self.assertIn('enable --now paper-chess-shortcut.service', log)
        self.assertNotIn('enable --now paper-chess-ui.service', log)
        self.assertFalse(list(self.home.glob('.paper-chess-install.??????')))

    def test_reinstall_preserves_saved_game(self):
        save = self.seed_existing()
        before = save.read_bytes()
        result = self.run_script()
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)
        self.assertEqual(save.read_bytes(), before)
        self.assertFalse((self.home / 'paper-chess/version').exists())

    def test_failed_shortcut_rolls_back_program_units_and_game(self):
        save = self.seed_existing()
        self.env['TEST_FAIL_START'] = '1'
        result = self.run_script()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Restoring the previous installation', result.stderr)
        self.assertEqual((self.home / 'paper-chess/version').read_text(), 'previous-release')
        self.assertEqual((self.units / 'paper-chess-ui.service').read_text(), 'old-ui')
        self.assertIn('keep my game', save.read_text())
        self.assertIn('start xochitl.service', (self.root / 'services.log').read_text())

    def test_failed_fresh_install_removes_partial_program(self):
        self.env['TEST_FAIL_START'] = '1'
        result = self.run_script()
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse((self.home / 'paper-chess').exists())
        self.assertEqual(list(self.units.iterdir()), [])

    def test_unsupported_firmware_has_no_service_changes(self):
        self.config.write_text('REMARKABLE_RELEASE_VERSION=0.0.0\n')
        result = self.run_script()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Only firmware 3.29.0.153', result.stderr)
        self.assertFalse((self.root / 'services.log').exists())

    def test_corrupt_runtime_has_no_service_changes(self):
        (self.package / 'Main.qml').write_text('corrupted')
        result = self.run_script()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('checksum verification failed', result.stderr)
        self.assertFalse((self.root / 'services.log').exists())

    def test_corrupt_download_is_not_extracted(self):
        result = self.run_script(online=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Download checksum mismatch', result.stderr)
        self.assertFalse((self.home / 'paper-chess').exists())
        self.assertFalse(list(self.home.glob('.paper-chess-download.??????')))

    def test_download_failure_has_no_install(self):
        self.env['TEST_DOWNLOAD_FAIL'] = '1'
        result = self.run_script(online=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse((self.home / 'paper-chess').exists())

    def test_verified_download_runs_complete_install(self):
        archive = self.root / 'fixture.tar.gz'
        with tarfile.open(archive, 'w:gz') as stream:
            stream.add(self.package, arcname='paper-chess')
        digest = hashlib.sha256(archive.read_bytes()).hexdigest()
        online = self.root / 'online.sh'
        online.write_text(re.sub(r'^SHA256=.*$', 'SHA256=' + digest, online.read_text(), flags=re.M), newline='\n')
        self.env['TEST_DOWNLOAD_FILE'] = shellpath(archive)
        result = self.run_script(online=True)
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)
        self.assertTrue((self.home / 'paper-chess/Main.qml').exists())
        self.assertFalse(list(self.home.glob('.paper-chess-download.??????')))


if __name__ == '__main__':
    unittest.main()
