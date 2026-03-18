# Installation

kubara is distributed as prebuilt release archives.
You do not need Go installed to run the CLI.

## Using brew (MacOS)

```shell
brew tap kubara-io/homebrew-tap
brew install kubara
```

## Using the install script

Execute our install script:

```shell
curl -sSLf https://raw.githubusercontent.com/kubara-io/kubara/refs/heads/main/install.sh | sh
```

## Download Release Assets

Get binaries from:

<https://github.com/kubara-io/kubara/releases>

Optional: verify checksums after download (see [Verify Checksums](#verify-checksums)).

Current release artifacts:

- Linux: `kubara_<version>_linux_amd64.tar.gz`, `kubara_<version>_linux_arm64.tar.gz`
- macOS: `kubara_<version>_darwin_amd64.tar.gz`, `kubara_<version>_darwin_arm64.tar.gz`
- Windows: `kubara_<version>_windows_amd64.zip`, `kubara_<version>_windows_arm64.zip`

## Linux / macOS (Terminal)

Run the following commands in your terminal:

```bash
tar -xzf kubara_<version>_<os>_<arch>.tar.gz
chmod +x kubara
sudo mv kubara /usr/local/bin/kubara
kubara --help
```

## Windows

Download the matching Windows `.zip` release asset and extract it.

Open a terminal (PowerShell) in the extracted folder and run:

```powershell
.\kubara.exe --help
```

Optional: move `kubara.exe` to a directory in your `PATH`.

## Verify Checksums

Each release includes a checksum file.
Run these checksum commands in your terminal on Linux/macOS:

```bash
sha256sum kubara_<version>_<os>_<arch>.<ext>
```

On macOS you can also use:

```bash
shasum -a 256 kubara_<version>_<os>_<arch>.<ext>
```
