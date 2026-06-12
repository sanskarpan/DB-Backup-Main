#!/usr/bin/env bash

# ============================================================================
# DB Backup Manager - Shell Completion Installation Script
# ============================================================================
# Automatically detects your shell and installs the appropriate completion
# ============================================================================

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}==========================================${NC}"
echo -e "${BLUE}DB Backup Manager - Completion Installer${NC}"
echo -e "${BLUE}==========================================${NC}"
echo ""

# ============================================================================
# Detect Shell
# ============================================================================

detect_shell() {
    # Check SHELL environment variable
    if [[ -n "$SHELL" ]]; then
        case "$SHELL" in
            */bash)
                echo "bash"
                return
                ;;
            */zsh)
                echo "zsh"
                return
                ;;
            */fish)
                echo "fish"
                return
                ;;
        esac
    fi

    # Fallback to detecting from process
    if ps -p $$ | grep -q bash; then
        echo "bash"
    elif ps -p $$ | grep -q zsh; then
        echo "zsh"
    elif ps -p $$ | grep -q fish; then
        echo "fish"
    else
        echo "unknown"
    fi
}

# ============================================================================
# Install Functions
# ============================================================================

install_bash() {
    echo -e "${BLUE}Installing Bash completion...${NC}"

    local install_path=""

    # Determine installation path
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        if [[ -d "/usr/local/etc/bash_completion.d" ]]; then
            install_path="/usr/local/etc/bash_completion.d/db-backup"
        elif [[ -d "/opt/homebrew/etc/bash_completion.d" ]]; then
            install_path="/opt/homebrew/etc/bash_completion.d/db-backup"
        else
            # User-local
            install_path="$HOME/.bash_completion.d/db-backup"
            mkdir -p "$HOME/.bash_completion.d"
        fi
    else
        # Linux
        if [[ -w "/etc/bash_completion.d" ]]; then
            install_path="/etc/bash_completion.d/db-backup"
        else
            # User-local
            install_path="$HOME/.bash_completion.d/db-backup"
            mkdir -p "$HOME/.bash_completion.d"
        fi
    fi

    # Copy completion script
    if cp "$SCRIPT_DIR/bash/db-backup" "$install_path"; then
        echo -e "${GREEN}✓${NC} Installed to: $install_path"
    else
        echo -e "${RED}✗${NC} Failed to install"
        echo -e "${YELLOW}  Try running with sudo:${NC} sudo $0"
        return 1
    fi

    # Add source line to bashrc if needed
    if [[ "$install_path" == *".bash_completion.d"* ]]; then
        local bashrc="$HOME/.bashrc"
        local source_line="[[ -f $install_path ]] && source $install_path"

        if ! grep -q "db-backup" "$bashrc" 2>/dev/null; then
            echo "" >> "$bashrc"
            echo "# DB Backup Manager completion" >> "$bashrc"
            echo "$source_line" >> "$bashrc"
            echo -e "${GREEN}✓${NC} Added source line to $bashrc"
        fi
    fi

    echo ""
    echo -e "${YELLOW}To activate completion:${NC}"
    echo -e "  source $install_path"
    echo -e "  ${BLUE}or restart your shell${NC}"
}

install_zsh() {
    echo -e "${BLUE}Installing Zsh completion...${NC}"

    local install_path=""

    # Check for oh-my-zsh
    if [[ -d "$HOME/.oh-my-zsh/completions" ]]; then
        install_path="$HOME/.oh-my-zsh/completions/_db-backup"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        if [[ -d "/usr/local/share/zsh/site-functions" ]]; then
            install_path="/usr/local/share/zsh/site-functions/_db-backup"
        elif [[ -d "/opt/homebrew/share/zsh/site-functions" ]]; then
            install_path="/opt/homebrew/share/zsh/site-functions/_db-backup"
        else
            # User-local
            install_path="$HOME/.zfunc/_db-backup"
            mkdir -p "$HOME/.zfunc"
        fi
    else
        # Linux
        if [[ -w "/usr/local/share/zsh/site-functions" ]]; then
            install_path="/usr/local/share/zsh/site-functions/_db-backup"
        else
            # User-local
            install_path="$HOME/.zfunc/_db-backup"
            mkdir -p "$HOME/.zfunc"
        fi
    fi

    # Copy completion script
    if cp "$SCRIPT_DIR/zsh/_db-backup" "$install_path"; then
        echo -e "${GREEN}✓${NC} Installed to: $install_path"
    else
        echo -e "${RED}✗${NC} Failed to install"
        echo -e "${YELLOW}  Try running with sudo:${NC} sudo $0"
        return 1
    fi

    # Add fpath to zshrc if needed
    if [[ "$install_path" == *".zfunc"* ]]; then
        local zshrc="$HOME/.zshrc"
        local fpath_line='fpath=(~/.zfunc $fpath)'

        if ! grep -q "\.zfunc" "$zshrc" 2>/dev/null; then
            echo "" >> "$zshrc"
            echo "# DB Backup Manager completion" >> "$zshrc"
            echo "$fpath_line" >> "$zshrc"
            echo "autoload -Uz compinit && compinit" >> "$zshrc"
            echo -e "${GREEN}✓${NC} Added fpath to $zshrc"
        fi
    fi

    echo ""
    echo -e "${YELLOW}To activate completion:${NC}"
    echo -e "  exec zsh"
    echo -e "  ${BLUE}or restart your shell${NC}"
}

install_fish() {
    echo -e "${BLUE}Installing Fish completion...${NC}"

    local install_path=""

    # User config
    install_path="$HOME/.config/fish/completions/db-backup.fish"
    mkdir -p "$HOME/.config/fish/completions"

    # Copy completion script
    if cp "$SCRIPT_DIR/fish/db-backup.fish" "$install_path"; then
        echo -e "${GREEN}✓${NC} Installed to: $install_path"
    else
        echo -e "${RED}✗${NC} Failed to install"
        return 1
    fi

    echo ""
    echo -e "${YELLOW}To activate completion:${NC}"
    echo -e "  exec fish"
    echo -e "  ${BLUE}or restart your shell${NC}"
}

install_powershell() {
    echo -e "${BLUE}Installing PowerShell completion...${NC}"

    if [[ "$OSTYPE" != "msys" ]] && [[ "$OSTYPE" != "cygwin" ]] && [[ "$OSTYPE" != "win32" ]]; then
        echo -e "${RED}✗${NC} PowerShell completion is only available on Windows"
        return 1
    fi

    echo -e "${YELLOW}For PowerShell, please run:${NC}"
    echo -e "  db-backup completion powershell > db-backup.ps1"
    echo -e "  ${BLUE}then add to your PowerShell profile${NC}"
}

# ============================================================================
# Uninstall Functions
# ============================================================================

uninstall_bash() {
    echo -e "${BLUE}Uninstalling Bash completion...${NC}"

    local paths=(
        "/usr/local/etc/bash_completion.d/db-backup"
        "/opt/homebrew/etc/bash_completion.d/db-backup"
        "/etc/bash_completion.d/db-backup"
        "$HOME/.bash_completion.d/db-backup"
    )

    local removed=false
    for path in "${paths[@]}"; do
        if [[ -f "$path" ]]; then
            if rm "$path" 2>/dev/null; then
                echo -e "${GREEN}✓${NC} Removed: $path"
                removed=true
            fi
        fi
    done

    if [[ "$removed" == "false" ]]; then
        echo -e "${YELLOW}⚠${NC}  No installation found"
    fi
}

uninstall_zsh() {
    echo -e "${BLUE}Uninstalling Zsh completion...${NC}"

    local paths=(
        "$HOME/.oh-my-zsh/completions/_db-backup"
        "/usr/local/share/zsh/site-functions/_db-backup"
        "/opt/homebrew/share/zsh/site-functions/_db-backup"
        "$HOME/.zfunc/_db-backup"
    )

    local removed=false
    for path in "${paths[@]}"; do
        if [[ -f "$path" ]]; then
            if rm "$path" 2>/dev/null; then
                echo -e "${GREEN}✓${NC} Removed: $path"
                removed=true
            fi
        fi
    done

    if [[ "$removed" == "false" ]]; then
        echo -e "${YELLOW}⚠${NC}  No installation found"
    fi
}

uninstall_fish() {
    echo -e "${BLUE}Uninstalling Fish completion...${NC}"

    local path="$HOME/.config/fish/completions/db-backup.fish"

    if [[ -f "$path" ]]; then
        if rm "$path"; then
            echo -e "${GREEN}✓${NC} Removed: $path"
        fi
    else
        echo -e "${YELLOW}⚠${NC}  No installation found"
    fi
}

# ============================================================================
# Main Script
# ============================================================================

# Parse arguments
case "${1:-}" in
    bash|zsh|fish)
        SHELL_TYPE="$1"
        ;;
    uninstall)
        SHELL_TYPE=$(detect_shell)
        echo -e "Detected shell: ${GREEN}$SHELL_TYPE${NC}"
        echo ""

        case "$SHELL_TYPE" in
            bash) uninstall_bash ;;
            zsh) uninstall_zsh ;;
            fish) uninstall_fish ;;
            *) echo -e "${RED}Unsupported shell: $SHELL_TYPE${NC}"; exit 1 ;;
        esac
        exit 0
        ;;
    --help|-h)
        echo "Usage: $0 [bash|zsh|fish|uninstall]"
        echo ""
        echo "Install or uninstall shell completions for db-backup."
        echo ""
        echo "Options:"
        echo "  bash        Install Bash completion"
        echo "  zsh         Install Zsh completion"
        echo "  fish        Install Fish completion"
        echo "  uninstall   Uninstall completion for current shell"
        echo "  --help      Show this help message"
        echo ""
        echo "If no argument is provided, the script will auto-detect your shell."
        exit 0
        ;;
    "")
        SHELL_TYPE=$(detect_shell)
        if [[ "$SHELL_TYPE" == "unknown" ]]; then
            echo -e "${RED}✗${NC} Could not detect shell"
            echo ""
            echo "Please specify your shell:"
            echo "  $0 bash"
            echo "  $0 zsh"
            echo "  $0 fish"
            exit 1
        fi
        ;;
    *)
        echo -e "${RED}✗${NC} Unknown option: $1"
        echo "Run '$0 --help' for usage information"
        exit 1
        ;;
esac

echo -e "Detected shell: ${GREEN}$SHELL_TYPE${NC}"
echo ""

# Install for detected/specified shell
case "$SHELL_TYPE" in
    bash)
        install_bash
        ;;
    zsh)
        install_zsh
        ;;
    fish)
        install_fish
        ;;
    *)
        echo -e "${RED}✗${NC} Unsupported shell: $SHELL_TYPE"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}==========================================${NC}"
echo -e "${GREEN}Installation complete!${NC}"
echo -e "${GREEN}==========================================${NC}"
