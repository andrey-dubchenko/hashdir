# hashdir
Program calculates files hash of the target directory recursively and saves result to file.

Usage: go run hashdir [sha256/md5] <directory>

If the algorithm is omitted, sha256 is used by default.

Version: 1.2.0
Author: Andrey Dubchenko - https://github.com/andrey-dubchenko

## Build

`make` works the same on Windows (it hands every target to `make.ps1`) and on Linux/macOS.

```
make                 # build for this machine
make windows         # dist/windows-amd64/hashdir.exe (icon and version info linked in)
make linux           # dist/linux-amd64/: binary, desktop entry, icons, install.sh
make dist            # both of the above
make check           # gofmt + go vet + tests
make VERSION=1.2.0 build    # version printed by `hashdir -v` (default: git describe)
make icons           # redraw assets/icon.ico and the PNG icons
make resources       # recompile hashdir_windows_amd64.syso (needs windres from MinGW)
make clean
```

On Linux, `sh install.sh` from `dist/linux-amd64` installs the binary, desktop entry and icons into `~/.local`
(`PREFIX=/usr/local` for system-wide, `sh install.sh uninstall` to remove).
