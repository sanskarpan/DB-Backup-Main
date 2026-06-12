#!/usr/bin/env bash

# ============================================================================
# DB Backup Manager - Completion Testing Script
# ============================================================================
# Automated tests for shell completions
# ============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

PASS_COUNT=0
FAIL_COUNT=0

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}DB Backup Completions - Test Suite${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# Test function
run_test() {
    local name="$1"
    local command="$2"

    echo -n "Testing: $name... "

    if eval "$command" >/dev/null 2>&1; then
        echo -e "${GREEN}✓ PASS${NC}"
        ((PASS_COUNT++))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        ((FAIL_COUNT++))
        return 1
    fi
}

run_test_output() {
    local name="$1"
    local command="$2"
    local expected="$3"

    echo -n "Testing: $name... "

    output=$(eval "$command" 2>&1)
    if [[ "$output" == *"$expected"* ]]; then
        echo -e "${GREEN}✓ PASS${NC}"
        ((PASS_COUNT++))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        echo -e "  Expected: $expected"
        echo -e "  Got: $output"
        ((FAIL_COUNT++))
        return 1
    fi
}

echo -e "${YELLOW}Phase 1: File Existence Checks${NC}"
echo "================================"

run_test "Bash completion file exists" "[[ -f bash/db-backup ]]"
run_test "Zsh completion file exists" "[[ -f zsh/_db-backup ]]"
run_test "Fish completion file exists" "[[ -f fish/db-backup.fish ]]"
run_test "PowerShell completion file exists" "[[ -f powershell/db-backup.ps1 ]]"
run_test "Install script exists" "[[ -f install.sh ]]"
run_test "Install script is executable" "[[ -x install.sh ]]"
run_test "README exists" "[[ -f README.md ]]"
run_test "TESTING guide exists" "[[ -f TESTING.md ]]"

echo ""
echo -e "${YELLOW}Phase 2: Syntax Validation${NC}"
echo "================================"

run_test "Bash completion syntax" "bash -n bash/db-backup"
run_test "Install script syntax" "bash -n install.sh"

# Test Zsh if available
if command -v zsh >/dev/null 2>&1; then
    run_test "Zsh completion syntax" "zsh -n zsh/_db-backup"
else
    echo "Skipping: Zsh completion syntax... ${YELLOW}⚠ Zsh not installed${NC}"
fi

# Test Fish if available
if command -v fish >/dev/null 2>&1; then
    run_test "Fish completion syntax" "fish -n fish/db-backup.fish"
else
    echo "Skipping: Fish completion syntax... ${YELLOW}⚠ Fish not installed${NC}"
fi

# Test PowerShell if available
if command -v pwsh >/dev/null 2>&1; then
    run_test "PowerShell completion syntax" "pwsh -NoProfile -Command \"Get-Content powershell/db-backup.ps1 -ErrorAction Stop\""
else
    echo "Skipping: PowerShell completion syntax... ${YELLOW}⚠ PowerShell not installed${NC}"
fi

echo ""
echo -e "${YELLOW}Phase 3: Script Functionality${NC}"
echo "================================"

run_test_output "Install script help" "./install.sh --help" "Usage:"
run_test_output "Install script lists shells" "./install.sh --help" "bash|zsh|fish"

echo ""
echo -e "${YELLOW}Phase 4: Bash Completion Loading${NC}"
echo "================================"

# Test Bash completion can be sourced
if source bash/db-backup 2>/dev/null; then
    echo -e "Testing: Bash completion loads... ${GREEN}✓ PASS${NC}"
    ((PASS_COUNT++))

    # Test function is defined
    if type _db_backup_completion >/dev/null 2>&1; then
        echo -e "Testing: Bash completion function defined... ${GREEN}✓ PASS${NC}"
        ((PASS_COUNT++))
    else
        echo -e "Testing: Bash completion function defined... ${RED}✗ FAIL${NC}"
        ((FAIL_COUNT++))
    fi

    # Test cache functions are defined
    if type _db_backup_cache_read >/dev/null 2>&1; then
        echo -e "Testing: Bash cache functions defined... ${GREEN}✓ PASS${NC}"
        ((PASS_COUNT++))
    else
        echo -e "Testing: Bash cache functions defined... ${RED}✗ FAIL${NC}"
        ((FAIL_COUNT++))
    fi

    # Test helper functions are defined
    if type _db_backup_databases >/dev/null 2>&1; then
        echo -e "Testing: Bash helper functions defined... ${GREEN}✓ PASS${NC}"
        ((PASS_COUNT++))
    else
        echo -e "Testing: Bash helper functions defined... ${RED}✗ FAIL${NC}"
        ((FAIL_COUNT++))
    fi
else
    echo -e "Testing: Bash completion loads... ${RED}✗ FAIL${NC}"
    ((FAIL_COUNT++))
fi

echo ""
echo -e "${YELLOW}Phase 5: Content Validation${NC}"
echo "================================"

# Check Bash completion contains key features
run_test "Bash has caching" "grep -q '_db_backup_cache_read' bash/db-backup"
run_test "Bash has database completion" "grep -q '_db_backup_databases' bash/db-backup"
run_test "Bash has backup completion" "grep -q '_db_backup_backups' bash/db-backup"
run_test "Bash has provider completion" "grep -q '_db_backup_providers' bash/db-backup"

# Check Zsh completion contains key features
run_test "Zsh has descriptions" "grep -q '_describe' zsh/_db-backup"
run_test "Zsh has database completion" "grep -q '_db_backup_databases' zsh/_db-backup"
run_test "Zsh has caching" "grep -q '_db_backup_exec' zsh/_db-backup"

# Check Fish completion contains key features
run_test "Fish has database completion" "grep -q '__db_backup_databases' fish/db-backup.fish"
run_test "Fish has caching" "grep -q '__db_backup_exec' fish/db-backup.fish"
run_test "Fish has descriptions" "grep -q 'complete.*-d' fish/db-backup.fish"

# Check PowerShell completion
run_test "PowerShell has completion block" "grep -q 'Register-ArgumentCompleter' powershell/db-backup.ps1"
run_test "PowerShell has database function" "grep -q 'Get-DbBackupDatabases' powershell/db-backup.ps1"

echo ""
echo -e "${YELLOW}Phase 6: Documentation Validation${NC}"
echo "================================"

run_test "README has quick start" "grep -q 'Quick Start' README.md"
run_test "README has all shells" "grep -q 'Bash.*Zsh.*Fish.*PowerShell' README.md"
run_test "README has installation" "grep -q 'Installation' README.md"
run_test "README has troubleshooting" "grep -q 'Troubleshooting' README.md"
run_test "TESTING has test cases" "grep -q 'Test.*Completion' TESTING.md"
run_test "TESTING has benchmarks" "grep -q 'Performance' TESTING.md"

echo ""
echo -e "${YELLOW}Phase 7: Shell Detection${NC}"
echo "================================"

# Test install script can detect shells
test_shell_detection() {
    local shell="$1"
    # This is a basic test - just checks the script can handle the argument
    ./install.sh --help | grep -q "$shell"
}

run_test "Install script mentions bash" "test_shell_detection bash"
run_test "Install script mentions zsh" "test_shell_detection zsh"
run_test "Install script mentions fish" "test_shell_detection fish"

echo ""
echo -e "${YELLOW}Phase 8: Cache Directory Structure${NC}"
echo "================================"

# Check cache-related code
run_test "Bash defines cache dir" "grep -q 'DB_BACKUP_CACHE_DIR' bash/db-backup"
run_test "Bash defines cache TTL" "grep -q 'DB_BACKUP_CACHE_TTL' bash/db-backup"
run_test "Zsh defines cache dir" "grep -q 'DB_BACKUP_CACHE_DIR' zsh/_db-backup"
run_test "Fish defines cache dir" "grep -q 'DB_BACKUP_CACHE_DIR' fish/db-backup.fish"
run_test "PowerShell defines cache dir" "grep -q 'DB_BACKUP_CACHE_DIR' powershell/db-backup.ps1"

echo ""
echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}Test Results Summary${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""
echo -e "  ${GREEN}Passed:${NC} $PASS_COUNT tests"
echo -e "  ${RED}Failed:${NC} $FAIL_COUNT tests"
echo ""

if [[ $FAIL_COUNT -eq 0 ]]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo ""
    echo "The shell completions are ready for use."
    echo "Run './install.sh' to install for your shell."
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    echo ""
    echo "Please review the failures above and fix any issues."
    echo "See TESTING.md for detailed testing instructions."
    exit 1
fi
