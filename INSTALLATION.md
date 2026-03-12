# Installation

Kubara is distributed as prebuilt release archives.
You do not need Go installed to run the CLI.

## Download Release Assets

Get binaries from:

<https://github.com/kubara-io/kubara/releases>

Current release artifacts:

- Linux: `kubara_<version>_linux_amd64.tar.gz`, `kubara_<version>_linux_arm64.tar.gz`
- macOS: `kubara_<version>_darwin_amd64.tar.gz`, `kubara_<version>_darwin_arm64.tar.gz`
- Windows: `kubara_<version>_windows_amd64.zip`, `kubara_<version>_windows_arm64.zip`

## Verify Checksums

Each release includes a checksum file.
Verify the downloaded archive before extracting:

```bash
sha256sum kubara_<version>_<os>_<arch>.<ext>
```

On macOS you can also use:

```bash
shasum -a 256 kubara_<version>_<os>_<arch>.<ext>
```

## Linux / macOS

```bash
tar -xzf kubara_<version>_<os>_<arch>.tar.gz
chmod +x kubara
sudo mv kubara /usr/local/bin/kubara
kubara --help
```

## Windows (PowerShell)

```powershell
Expand-Archive .\kubara_<version>_windows_<arch>.zip -DestinationPath .
.\kubara.exe --help
```

Optional: move `kubara.exe` to a directory in your `PATH`.
