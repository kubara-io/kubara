#!/bin/sh

# Exit immediately if a command exits with a non-zero status
set -e

REPO="kubara-io/kubara"

# Wrapper for curl to handle HTTP errors gracefully and print API/Server responses
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
        # File download mode (curl writes body to $_sc_out_file directly)
        _sc_http_code=$(curl -sS -w "%{http_code}" "$@" 2>"$_sc_tmp_err")
        case "$_sc_http_code" in
            2*|3*) ;; # Success
            *)
                echo "Error: curl failed with HTTP status ${_sc_http_code:-UNKNOWN}" >&2
                cat "$_sc_tmp_err" >&2
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
        # Output the successful response to stdout so pipelines (like grep) still work
        cat "$_sc_tmp_out"
    fi

    rm -f "$_sc_tmp_err" "$_sc_tmp_out"
}

echo "Fetching the latest release version..."
# Fetch the latest release tag via GitHub API
# Using curl_wrap. On rate-limit, it will print the GH JSON and return 1, failing the script.
LATEST_TAG=$(curl_wrap "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
    echo "Error: Failed to fetch the latest version."
    exit 1
fi

# The GitHub tag has a 'v' (e.g. v0.6.1), but filenames do not.
# We use POSIX parameter expansion to strip the 'v'
VERSION=${LATEST_TAG#v}

echo "Latest version found: $LATEST_TAG"

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

# Construct filenames based on detection
FILENAME="kubara_${VERSION}_${OS}_${ARCH}.tar.gz"
CHECKSUM_FILE="kubara_${VERSION}_checksums.txt"

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_TAG/$FILENAME"
CHECKSUM_URL="https://github.com/$REPO/releases/download/$LATEST_TAG/$CHECKSUM_FILE"

# Use a temporary directory for safe downloading and extraction
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

echo "Downloading $FILENAME..."
# Note: Added -L so curl follows redirects, which GitHub releases require
curl_wrap -L -o "$FILENAME" "$DOWNLOAD_URL"

echo "Downloading checksum file..."
curl_wrap -L -o "$CHECKSUM_FILE" "$CHECKSUM_URL"

echo "Verifying checksum..."
# Isolate the exact file's checksum line so verification tools don't complain about missing OS/Arch files
grep "$FILENAME" "$CHECKSUM_FILE" > checksum_check.txt

# Linux usually uses sha256sum; macOS defaults to shasum
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

# Ensure the binary exists after extraction
if [ ! -f "kubara" ]; then
    echo "Error: 'kubara' binary was not found in the extracted archive."
    exit 1
fi

INSTALL_DIR="$HOME/.local/bin"

echo "Installing kubara to $INSTALL_DIR..."
# Ensure the local bin directory exists
mkdir -p "$INSTALL_DIR"

# Move the binary without sudo
mv kubara "$INSTALL_DIR/kubara"
chmod +x "$INSTALL_DIR/kubara"

# Clean up
cd - > /dev/null
rm -rf "$TMP_DIR"

echo "Installation complete!"
echo "Make sure '$INSTALL_DIR' is in your system PATH."
echo "You can verify the installation by running:"
echo "kubara --version"