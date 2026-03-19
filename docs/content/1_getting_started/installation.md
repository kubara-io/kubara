# Installation

kubara is distributed via Homebrew and as prebuilt release archives.
You do not need Go installed to run the CLI.

## Installation Methods

=== "Homebrew"

    ```bash
    brew tap kubara-io/tap
    brew install kubara
    kubara --help
    ```

    Update:

    ```bash
    brew upgrade kubara
    ```

    Uninstall:

    ```bash
    brew uninstall kubara
    ```

=== "Install Script (macOS / Linux)"

    ```bash
    curl -sSLf https://raw.githubusercontent.com/kubara-io/kubara/refs/heads/main/install.sh | sh
    kubara --help
    ```

    The script downloads the latest release for your platform and verifies checksums automatically.

=== "Manual Download (macOS / Linux)"

    Download the matching release archive from:

    <https://github.com/kubara-io/kubara/releases>

    Current release artifacts:
    - Linux: `kubara_<version>_linux_amd64.tar.gz`, `kubara_<version>_linux_arm64.tar.gz`
    - macOS: `kubara_<version>_darwin_amd64.tar.gz`, `kubara_<version>_darwin_arm64.tar.gz`

    ```bash
    tar -xzf kubara_<version>_<os>_<arch>.tar.gz
    chmod +x kubara
    sudo mv kubara /usr/local/bin/kubara
    kubara --help
    ```

=== "Manual Download (Windows)"

    Download the matching Windows `.zip` release asset from:

    <https://github.com/kubara-io/kubara/releases>

    Current release artifacts:
    - `kubara_<version>_windows_amd64.zip`
    - `kubara_<version>_windows_arm64.zip`

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
