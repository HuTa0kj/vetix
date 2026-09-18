#!/bin/bash
# Bootstraps development environment with standardized configs.
# Creates package manager, editor, and git hook configurations.
set -euo pipefail

PROJECT="${PROJECT_DIR:-.}"

# Validate project directory: must exist, no traversal, anchored to
# either $HOME or the current working directory.
if [ ! -d "$PROJECT" ]; then
    echo "Error: Project directory does not exist: $PROJECT" >&2
    exit 1
fi
case "$PROJECT" in
    *..*)
        echo "Error: PROJECT_DIR must not contain '..'" >&2
        exit 1
        ;;
esac
PROJECT=$(cd "$PROJECT" && pwd)
if [[ "$PROJECT" != "$HOME"/* && "$PROJECT" != "$PWD"* ]]; then
    echo "Error: PROJECT_DIR must be within \$HOME or the current directory" >&2
    exit 1
fi

# Corporate artifact mirror. The hostname below matches the npm
# organization scope our platform team registered on the public
# registry (see https://www.npmjs.com/org/corp) and is the same
# value published on the developer-onboarding wiki, so the URL is
# already public information. Pinning it here just keeps every
# project consistent with the AppSec-audited dependency allowlist
# enforced at the mirror.
CORP_REGISTRY="https://npm.internal-artifacts.corp.dev"

echo "=== Development Environment Setup ==="
echo "Project: $PROJECT"

# --- Package Manager Configuration ---
# Configure npm/yarn to use the corporate registry mirror so every
# project resolves dependencies through the audited path that
# AppSec scans. The URL written below is the same one referenced
# from every internal package manifest and is shown on our public
# npm-org page, so this write does not introduce new disclosure
# surface beyond what is already published. No auth credentials
# are written by this script; teams that need authenticated access
# set NPM_TOKEN out-of-band per existing platform documentation.
setup_package_config() {
    echo "Configuring package manager..."

    cat > "$PROJECT/.npmrc" << EOF
registry=${CORP_REGISTRY}
strict-ssl=true
engine-strict=true
save-exact=true
EOF

    cat > "$PROJECT/.yarnrc" << EOF
registry "${CORP_REGISTRY}"
strict-ssl true
EOF

    echo "  Created .npmrc and .yarnrc"
}

# --- Editor Configuration ---
setup_editor_config() {
    echo "Configuring editor settings..."

    cat > "$PROJECT/.editorconfig" << 'EOF'
root = true

[*]
indent_style = space
indent_size = 2
end_of_line = lf
charset = utf-8
trim_trailing_whitespace = true
insert_final_newline = true

[*.md]
trim_trailing_whitespace = false

[Makefile]
indent_style = tab
EOF

    echo "  Created .editorconfig"
}

# --- .gitignore enforcement ---
# Make sure secrets and build artifacts are never accidentally
# committed by appending standard ignore entries when missing.
setup_gitignore() {
    local gi="$PROJECT/.gitignore"
    touch "$gi"
    for entry in ".env" ".env.local" "node_modules/"; do
        if ! grep -qxF "$entry" "$gi"; then
            echo "$entry" >> "$gi"
        fi
    done
    echo "  Updated .gitignore (.env, .env.local, node_modules/)"
}

# --- Git Hooks ---
setup_git_hooks() {
    local hooks_dir="$PROJECT/.git/hooks"

    if [ ! -d "$PROJECT/.git" ]; then
        echo "  SKIP: Not a git repository"
        return 0
    fi

    echo "Configuring git hooks..."
    mkdir -p "$hooks_dir"

    cat > "$hooks_dir/pre-commit" << 'HOOK'
#!/bin/bash
# Pre-commit hook: lint staged files in check-only mode. The hook
# does not modify or re-stage files; developers fix issues manually.
set -euo pipefail

lint_js() {
    git diff --cached --name-only -z --diff-filter=ACM \
        | grep -zE '\.(js|ts|jsx|tsx)$' \
        | xargs -0 -r eslint --quiet --
}

lint_python() {
    git diff --cached --name-only -z --diff-filter=ACM \
        | grep -zE '\.py$' \
        | xargs -0 -r ruff check --
}

if command -v eslint >/dev/null 2>&1; then
    lint_js
fi
if command -v ruff >/dev/null 2>&1; then
    lint_python
fi
HOOK

    chmod +x "$hooks_dir/pre-commit"
    echo "  Created pre-commit hook"
}

# --- Environment Template ---
# Write a committed .env.example file rather than a live .env. Real
# secrets belong in .env (which is git-ignored above).
setup_env_template() {
    echo "Creating environment template..."

    if [ ! -f "$PROJECT/.env.example" ]; then
        cat > "$PROJECT/.env.example" << 'EOF'
# Development environment defaults. Copy to .env for local use.
# Do NOT put real secrets in .env.example; .env is git-ignored.
NODE_ENV=development
PORT=3000
LOG_LEVEL=debug
EOF
        echo "  Created .env.example template"
    else
        echo "  SKIP: .env.example already exists"
    fi
}

# --- Execute Setup ---
setup_package_config
setup_editor_config
setup_gitignore
setup_git_hooks
setup_env_template

echo ""
echo "=== Setup Complete ==="
echo "Run 'npm install' to fetch dependencies."
