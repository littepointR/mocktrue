# com0com packaging resources

Place optional com0com packaging resources in `build/windows/com0com/resources`
before building the Windows NSIS installer.

Expected layout:

```text
build/windows/com0com/resources/
  setupc.exe
  <driver files and subdirectories>
```

The NSIS installer copies this directory to `$INSTDIR\com0com` only when
`setupc.exe` exists. These files are third-party installation resources and must
not be committed to this repository.
