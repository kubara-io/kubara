# Installation

kubara is distributed via Homebrew and as prebuilt release archives.
You do not need Go installed to run the CLI.

## Installation Methods

=== "Homebrew"

    **Install**

    ```bash
    brew tap kubara-io/tap
    brew install kubara
    kubara --help
    ```

    **Update**

    ```bash
    brew upgrade kubara
    ```

    **Uninstall**

    ```bash
    brew uninstall kubara
    ```

=== "APT"

    **Install**

    ```bash
    sudo install -d -m 0755 /etc/apt/keyrings
    curl -fsSL https://apt.kubara.io/apt-public.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubara.gpg
    echo "deb [signed-by=/etc/apt/keyrings/kubara.gpg] https://apt.kubara.io stable main" | sudo tee /etc/apt/sources.list.d/kubara.list > /dev/null
    sudo apt update
    sudo apt install -y kubara
    ```

    **Uninstall**

    ```bash
    sudo apt remove -y kubara
    sudo rm -f /etc/apt/sources.list.d/kubara.list /etc/apt/keyrings/kubara.gpg
    sudo apt update
    ```
=== "Docker"

    For commands that need cluster access (e.g. `bootstrap`), mount your kubeconfig:

    ```bash
    docker run --rm \
      -u $(id -u):$(id -g) \
      -v ~/.kube/config:/kubeconfig:ro \
      -v $(pwd):/workspace \
      -w /workspace \
      ghcr.io/kubara-io/kubara <your-command>
    ```

    For local-only commands (e.g. `init`, `generate`, `schema`), kubeconfig is not required:

    ```bash
    docker run --rm \
      -u $(id -u):$(id -g) \
      -v $(pwd):/workspace \
      -w /workspace \
      ghcr.io/kubara-io/kubara <your-command>
    ```

=== "Install Script"

    ```bash
    curl -sSLf https://raw.githubusercontent.com/kubara-io/kubara/refs/heads/main/install.sh | sh
    kubara --help
    ```

    The script downloads the latest release for your platform and verifies checksums automatically.

=== "Manual (macOS/Linux)"

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

=== "Manual (Windows)"

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

## Shell Completion

kubara supports shell completion for bash, zsh, fish and powershell.

```shell
# add the following line to your .bashrc
$ source <(kubara completion bash)
# or for zsh
$ source <(kubara completion zsh)
# after loading your rc file or opening a new terminal you will have tab completion for kubara
$ kubara <tab>
bootstrap  -- Bootstrap ArgoCD onto the specified cluster with optional external-secrets and prometheus CRD
generate   -- generates files from embedded templates and the config file; by default for both Helm and Terraform
help       -- Shows a list of commands or help for one command
init       -- Initialize a new kubara directory
schema     -- Generate JSON schema file for config structure
```
