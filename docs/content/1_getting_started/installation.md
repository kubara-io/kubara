# Installation

kubara is distributed as prebuilt release archives. You do not need Go installed to run the CLI.

## 1. Using Homebrew (macOS)

The simplest way to install on macOS is via Homebrew:

```shell
brew tap kubara-io/homebrew-tap
brew install kubara
```

## 2. Using APT (Ubuntu/Debian)

### 1.Download the key securely
curl -fsSL https://docs.kubara.io/apt-public.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubara.gpg

### 2. Add the repo
echo "deb [signed-by=/etc/apt/keyrings/kubara.gpg] https://docs.kubara.io stable main" | sudo tee /etc/apt/sources.list.d/kubara.list > /dev/null

### 3. Update and install
sudo apt update
sudo apt install -y kubara

## 2. Using the Install Script (macOS / Linux)

You can automatically download and install the CLI by executing our install script:

```shell
curl -sSLf [https://raw.githubusercontent.com/kubara-io/kubara/refs/heads/main/install.sh](https://raw.githubusercontent.com/kubara-io/kubara/refs/heads/main/install.sh) | sh
```

## 3. Manual Download (Release Assets)

If you prefer to install manually or are using Windows, you can download the prebuilt binaries directly.

Get binaries from: <https://github.com/kubara-io/kubara/releases>

Current release artifacts:
- Linux: `kubara_<version>_linux_amd64.tar.gz`, `kubara_<version>_linux_arm64.tar.gz`
- macOS: `kubara_<version>_darwin_amd64.tar.gz`, `kubara_<version>_darwin_arm64.tar.gz`
- Windows: `kubara_<version>_windows_amd64.zip`, `kubara_<version>_windows_arm64.zip`

### Linux / macOS (Terminal)

Download the appropriate `.tar.gz` file for your system, then run the following commands in your terminal:

```bash
tar -xzf kubara_<version>_<os>_<arch>.tar.gz
chmod +x kubara
sudo mv kubara /usr/local/bin/kubara
kubara --help
```

### Windows

1. Download the matching Windows `.zip` release asset and extract it.
2. Open a terminal (PowerShell) in the extracted folder and run:

```powershell
.\kubara.exe --help
```

*Optional: Move `kubara.exe` to a directory included in your system's `PATH` variable for easier access.*

### Verify Checksums (Optional)

Each release includes a checksum file. To verify the integrity of your download on Linux or macOS, run the following in your terminal:

**Linux:**
```bash
sha256sum kubara_<version>_<os>_<arch>.<ext>
```

**macOS:**
```bash
shasum -a 256 kubara_<version>_<os>_<arch>.<ext>
```