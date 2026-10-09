#!/bin/sh

# Exit immediately if a command exits with a non-zero status
set -e

REPO="kubara-io/kubara"

# Wrapper for curl to handle HTTP errors gracefully and show progress for downloads
curl_wrap() {
    _sc_is_dl=0
    _sc_out_file=""

    # Parse arguments to see if an output file (-o) is specified
    for _sc_arg do
        if [ "$_sc_is_dl" -eq 2 ]; then
            _sc_out_file="$_sc_arg"
            _sc_is_dl=1
        elif [ "$_sc_arg" = "-o" ]; then
            _sc_is_dl=2
        fi
    done

    _sc_tmp_err=$(mktemp)
    _sc_tmp_out=$(mktemp)

    if [ "$_sc_is_dl" -eq 1 ]; then
        # File download mode.
        # We use -# for a progress bar and DO NOT redirect stderr, so the user sees it.
        _sc_http_code=$(curl -# -w "%{http_code}" "$@")

        case "$_sc_http_code" in
            2*|3*) ;; # Success
            *)
                # Print a newline because the progress bar might leave the cursor hanging
                echo "" >&2
                echo "Error: curl failed with HTTP status ${_sc_http_code:-UNKNOWN}" >&2
                if [ -f "$_sc_out_file" ] && [ -s "$_sc_out_file" ]; then
                    echo "Server response:" >&2
                    cat "$_sc_out_file" >&2
                    echo "" >&2
                fi
                rm -f "$_sc_tmp_err" "$_sc_tmp_out"
                return 1
                ;;
        esac
    else
        # Standard output mode (e.g., API requests in pipelines)
        # We keep this strictly silent (-sS) so it doesn't break grep/sed pipes
        _sc_http_code=$(curl -sS -w "%{http_code}" -o "$_sc_tmp_out" "$@" 2>"$_sc_tmp_err")
        case "$_sc_http_code" in
            2*|3*) ;; # Success
            *)
                echo "Error: curl failed with HTTP status ${_sc_http_code:-UNKNOWN}" >&2
                cat "$_sc_tmp_err" >&2
                if [ -s "$_sc_tmp_out" ]; then
                    echo "Server response:" >&2
                    cat "$_sc_tmp_out" >&2
                    echo "" >&2
                fi
                rm -f "$_sc_tmp_err" "$_sc_tmp_out"
                return 1
                ;;
        esac
        # Output the successful response to stdout
        cat "$_sc_tmp_out"
    fi

    rm -f "$_sc_tmp_err" "$_sc_tmp_out"
}

LATEST_TAG=${KUBARA_VERSION:-latest}
if [ "$LATEST_TAG" = latest ]; then
    echo "Fetching the latest release version..."
    LATEST_TAG=$(curl_wrap "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
else
    LATEST_TAG="v${LATEST_TAG#v}"
fi

if ! printf '%s\n' "$LATEST_TAG" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$'; then
    echo "Error: KUBARA_VERSION must be latest or a release version." >&2
    exit 1
fi

INSTALL_DIR=${KUBARA_INSTALL_DIR:-"$HOME/.local/bin"}
mkdir -p "$INSTALL_DIR"
INSTALL_DIR=$(cd "$INSTALL_DIR" && pwd)

VERSION=${LATEST_TAG#v}
echo "Installing release: $LATEST_TAG"

# Detect Operating System
OS_NAME=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS_NAME" in
    linux*) OS="linux" ;;
    darwin*) OS="darwin" ;;
    *) echo "Error: Unsupported OS '$OS_NAME'"; exit 1 ;;
esac

# Detect Architecture
ARCH_NAME=$(uname -m)
case "$ARCH_NAME" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Error: Unsupported architecture '$ARCH_NAME'"; exit 1 ;;
esac

echo "Detected Platform: $OS ($ARCH)"

FILENAME="kubara_${VERSION}_${OS}_${ARCH}.tar.gz"
CHECKSUM_FILE="kubara_${VERSION}_checksums.txt"

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_TAG/$FILENAME"
CHECKSUM_URL="https://github.com/$REPO/releases/download/$LATEST_TAG/$CHECKSUM_FILE"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT
cd "$TMP_DIR"

echo "Downloading $FILENAME..."
curl_wrap -L -o "$FILENAME" "$DOWNLOAD_URL"

echo "Downloading checksum file..."
curl_wrap -L -o "$CHECKSUM_FILE" "$CHECKSUM_URL"

echo "Verifying checksum..."
awk -v file="$FILENAME" '$2 == file {print}' "$CHECKSUM_FILE" > checksum_check.txt
test "$(wc -l < checksum_check.txt)" -eq 1

if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c checksum_check.txt
elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 -c checksum_check.txt
else
    echo "Error: Neither 'sha256sum' nor 'shasum' is installed. Cannot verify checksum."
    exit 1
fi

echo "Checksum verification successful."

echo "Extracting binary..."
tar -xzf "$FILENAME"

if [ ! -f "kubara" ]; then
    echo "Error: 'kubara' binary was not found in the extracted archive."
    exit 1
fi

echo "Installing kubara to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"

mv kubara "$INSTALL_DIR/kubara"
chmod +x "$INSTALL_DIR/kubara"

cd - > /dev/null
rm -rf "$TMP_DIR"

echo "Installation complete!"

# Use POSIX compliant string matching to check if INSTALL_DIR is already in PATH
if case ":$PATH:" in *":$INSTALL_DIR:"*) false;; *) true;; esac; then

    # Detect user's default shell
    USER_SHELL=$(basename "$SHELL" 2>/dev/null || echo "bash")

    case "$USER_SHELL" in
        zsh)  RC_FILE="$HOME/.zshrc" ;;
        bash) RC_FILE="$HOME/.bashrc" ;;
        fish) RC_FILE="$HOME/.config/fish/config.fish" ;;
        *)    RC_FILE="$HOME/.profile" ;;
    esac

    echo ""
    echo "============================================================"
    echo " NOTE: '$INSTALL_DIR' is not in your PATH."
    echo " To use the 'kubara' command globally, run the following:"
    echo "============================================================"

    if [ "$USER_SHELL" = "fish" ]; then
        echo "  fish_add_path $INSTALL_DIR"
    else
        echo "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> $RC_FILE"
        echo "  source $RC_FILE"
    fi
    echo "============================================================"
else
    echo ""
    echo "You can verify the installation by running:"
    echo "  kubara --version"
fi
