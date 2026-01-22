# Agent Configuration for the `x` Project

## 1. Build / Lint / Test Commands

The `x` repository is a lightweight shell script that ships as a single executable.  Because it contains no compiled code or external dependencies, the build step is simply **installing** the binary into `$PATH`.  For convenience the following commands are provided:

| Purpose | Command | Notes |
|---------|---------|-------|
| Install globally (requires sudo) | `bash install.sh` | Installs to `/usr/local/bin/x`. If you prefer a local installation, set `INSTALL_DIR=$HOME/.local/bin` before running the script. |
| Verify installation | `x --version` | The script prints its git commit hash and version from `VERSION`. |
| Lint shell code | `shellcheck *.sh` | Installs via Homebrew or apt (`sudo apt-get install -y shellcheck`). |
| Run a single test | **(See Tests section)** |
|
### Running the Built‑in Test Suite
The repository ships with a minimal test harness located in `tests/`.  Each test is a plain shell script that exits with status `0` on success. To run all tests: 
```bash
./run-tests.sh
```
To execute just one test file: 
```bash
bash tests/test_foo.sh
```
If you want to use the project's CI runner, add:
```yaml
- name: Run tests
  run: bash ./run-tests.sh
```

## 2. Code‑Style Guidelines
The project follows a strict but lightweight style that is easy for humans and CI to enforce.

### 2.1 File Structure & Naming
| Component | Convention |
|-----------|------------|
| Shell scripts (`*.sh`) | `kebab-case.sh` (e.g., `install.sh`, `run-tests.sh`). |
| Test scripts (`tests/*.sh`) | `test_*.sh`. |
| Configuration files | `*.conf` or `*.ini` if needed. |

### 2.2 Imports / Sourcing
All helper functions are kept in a single file `lib/utils.sh`. Scripts source it with:
```bash
source "$(dirname "$0")/../lib/utils.sh"
```
Avoid circular dependencies; keep sourced files pure.

### 2.3 Formatting & Indentation
* Use **two spaces** per indentation level.
* End every file with a single newline.
* Avoid tabs – the repository enforces `shfmt -w` in CI.

### 2.4 Bash Strict Mode
All scripts should begin with:
```bash
#!/usr/bin/env bash
set -euo pipefail
```
This ensures:
* `-e` exits on any command failure.
* `-u` treats unset variables as errors.
* `-o pipefail` propagates failures through pipelines.

### 2.5 Error Handling
* Wrap commands that may fail in a function that prints an error message and exits.
```bash
function run_cmd() {
    local cmd="$1"
    echo "$cmd"
    eval "$cmd" || { echo "❌ $cmd failed" >&2; exit 1; }
}
```
* Use `trap` to clean up temporary files on exit.

### 2.6 Naming Conventions
| Element | Convention |
|---------|------------|
| Functions | `snake_case()` |
| Variables | Upper‑case for constants (`readonly`) and lower‑case for locals |
| Exit codes | `0` success, non‑zero error (use 1–125 for generic errors) |

### 2.7 Documentation
* Every function should have a one‑line comment describing its purpose.
* Use `# @param` style comments for arguments if the function is complex.

## 3. Cursor Rules
The repository does not ship any `.cursor/rules/` or `.cursorrules` files, so there are no special cursor rules to document.

## 4. Copilot Instructions
No `copilot-instructions.md` file exists in this repo; agents should rely on the style guide above.

## 5. CI / GitHub Actions
The project uses a minimal workflow located at `.github/workflows/ci.yml`. It performs:
1. Checkout
2. Install `shellcheck`
3. Run `shellcheck` on all `*.sh`
4. Execute `./run-tests.sh`

Feel free to extend this workflow with additional linting tools such as `shfmt`.

---

> **Tip for agents**: When making changes, run `bash ./run-tests.sh` locally first and ensure `shellcheck -x *.sh` passes before committing.  This keeps the repo fast and reliable.
