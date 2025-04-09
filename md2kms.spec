# -*- mode: python ; coding: utf-8 -*-


a = Analysis(
    ['src/markdown_to_confluence/main.py'],
    pathex=[],
    binaries=[],
    datas=[('src/markdown_to_confluence/config.yml', 'markdown_to_confluence')],
    hiddenimports=['PIL._tkinter', 'PIL._imagingtk'],
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[],
    noarchive=False,
)
pyz = PYZ(a.pure)

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.datas,
    [],
    name='md2kms',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=True,
    disable_windowed_traceback=False,
    argv_emulation=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
)
