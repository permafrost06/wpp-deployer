#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

# Detect shell and set completion directory
detect_shell_and_install() {
    local shell_name="$1"
    local completion_file="$2"
    local completion_dir=""
    local install_path=""

    case "$shell_name" in
        bash)
            # Common bash completion directories
            if [[ -d "/usr/local/share/bash-completion/completions" ]]; then
                completion_dir="/usr/local/share/bash-completion/completions"
            elif [[ -d "/usr/share/bash-completion/completions" ]]; then
                completion_dir="/usr/share/bash-completion/completions"
            elif [[ -d "$HOME/.local/share/bash-completion/completions" ]]; then
                completion_dir="$HOME/.local/share/bash-completion/completions"
            else
                mkdir -p "$HOME/.local/share/bash-completion/completions"
                completion_dir="$HOME/.local/share/bash-completion/completions"
            fi
            install_path="$completion_dir/wpp-deployer"
            ;;
        zsh)
            # Common zsh completion directories
            if [[ -d "/usr/local/share/zsh/site-functions" ]]; then
                completion_dir="/usr/local/share/zsh/site-functions"
            elif [[ -d "/usr/share/zsh/site-functions" ]]; then
                completion_dir="/usr/share/zsh/site-functions"
            else
                # Use user's zsh function directory
                completion_dir="$HOME/.local/share/zsh/site-functions"
                mkdir -p "$completion_dir"
                
                # Add to fpath if not already there
                if [[ -f "$HOME/.zshrc" ]] && ! grep -q "$completion_dir" "$HOME/.zshrc"; then
                    echo "fpath=($completion_dir \$fpath)" >> "$HOME/.zshrc"
                    print_warning "Added $completion_dir to fpath in ~/.zshrc"
                fi
            fi
            install_path="$completion_dir/_wpp-deployer"
            ;;
        *)
            print_error "Unsupported shell: $shell_name"
            return 1
            ;;
    esac

    # Check if we need sudo
    if [[ "$completion_dir" == /usr/* ]]; then
        if [[ $EUID -ne 0 ]]; then
            print_warning "Installing to system directory requires sudo"
            sudo cp "$completion_file" "$install_path"
        else
            cp "$completion_file" "$install_path"
        fi
    else
        cp "$completion_file" "$install_path"
    fi

    print_status "Installed $shell_name completion to $install_path"
}

main() {
    echo "Installing wpp-deployer shell completions..."
    
    # Check if completion files exist
    if [[ ! -f "completions/wpp-deployer.bash" ]]; then
        print_error "Bash completion file not found: completions/wpp-deployer.bash"
        exit 1
    fi
    
    if [[ ! -f "completions/_wpp-deployer" ]]; then
        print_error "Zsh completion file not found: completions/_wpp-deployer"
        exit 1
    fi

    # Install bash completion
    if command -v bash >/dev/null 2>&1; then
        detect_shell_and_install "bash" "completions/wpp-deployer.bash"
    else
        print_warning "Bash not found, skipping bash completion"
    fi

    # Install zsh completion  
    if command -v zsh >/dev/null 2>&1; then
        detect_shell_and_install "zsh" "completions/_wpp-deployer"
    else
        print_warning "Zsh not found, skipping zsh completion"
    fi

    echo
    print_status "Installation complete!"
    echo
    echo "To enable completions in your current shell session:"
    echo "  Bash: source /etc/bash_completion or restart your terminal"
    echo "  Zsh:  run 'compinit' or restart your terminal"
    echo
    echo "Try: wpp-deployer <TAB> to test completion"
}

main "$@" 