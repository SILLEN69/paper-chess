"""Package program and source without games, notebook data or local diagnostics."""
import hashlib
from pathlib import Path
import shutil
import sys
import tarfile

root=Path(__file__).resolve().parent
output=Path(sys.argv[1]) if len(sys.argv)>1 else Path('paper-chess-remarkable2.tar.gz')
output.parent.mkdir(parents=True,exist_ok=True)
runtime=['paper-chess','chess-ui.so','Main.qml','session','paper-chess-backend.service','paper-chess-shortcut.service','paper-chess-ui.service','install.sh','disable.sh']
runtime += [f.relative_to(root).as_posix() for f in sorted((root/'assets').glob('*.svg'))]
checksums=''.join(hashlib.sha256((root/name).read_bytes()).hexdigest()+'  '+name+'\n' for name in runtime)
(root/'SHA256SUMS').write_bytes(checksums.encode('ascii'))
files=runtime+['README.md','LICENSE','INKLING-LICENSE','go.mod','bridge.c','make_assets.py','build_release.py','SHA256SUMS']
files += [f.relative_to(root).as_posix() for f in sorted((root/'docs').glob('*.md'))]
files += [f.relative_to(root).as_posix() for f in sorted((root/'docs/images').glob('*.png'))]
files += [f.name for f in root.glob('*.go')]
files += [f.relative_to(root).as_posix() for f in (root/'third_party/chess').rglob('*') if f.is_file() and (f.suffix in {'.go','.mod','.json','.pgn'} or f.name=='LICENSE')]
executables={'paper-chess','session','install.sh','disable.sh'}
with tarfile.open(output,'w:gz') as archive:
    for name in sorted(set(files)):
        source=root/name
        info=archive.gettarinfo(str(source),'paper-chess/'+name)
        info.mode=0o755 if name in executables else 0o644
        info.uid=info.gid=0
        info.uname=info.gname='root'
        with source.open('rb') as stream:archive.addfile(info,stream)
digest=hashlib.sha256(output.read_bytes()).hexdigest()
output.with_suffix('.gz.sha256').write_bytes((digest+'  '+output.name+'\n').encode('ascii'))
shutil.copyfile(root/'README.md',output.parent/'README.md')
with tarfile.open(output) as archive:
    for row in checksums.splitlines():
        expected,name=row.split('  ',1)
        assert hashlib.sha256(archive.extractfile('paper-chess/'+name).read()).hexdigest()==expected
print(f'Package verified: {output.stat().st_size} bytes, SHA256 {digest}')
