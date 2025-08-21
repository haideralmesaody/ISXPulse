# 🚨 MANDATORY BUILD RULES FOR ISX PULSE PROJECT

## ⚠️ CRITICAL: These Rules MUST Be Followed By Everyone

This document defines the **ABSOLUTE BUILD RULES** that must be followed by all developers, CI/CD systems, and AI assistants (including Claude Code) when working on this project.

---

## 🛑 THE GOLDEN RULE

```
════════════════════════════════════════════════════════════════
    NEVER BUILD IN THE api/ OR web/ DIRECTORIES - EVER!
    ALWAYS USE ./build.bat FROM PROJECT ROOT
    ALL BUILDS OUTPUT TO dist/ DIRECTORY ONLY
════════════════════════════════════════════════════════════════
```

---

## 📋 Complete Build Rules

### Rule 1: NO Building in Source Directories
- **NEVER** run `npm run build` in `web/`
- **NEVER** run `go build` anywhere in `api/`
- **NEVER** run `next build` in `web/`
- **NEVER** run `npm run export` in `web/`
- **NEVER** create `.next/` directory in `web/`
- **NEVER** create `out/` directory in `web/`

### Rule 2: ALWAYS Use build.bat
The **ONLY** approved way to build:
```bash
# From project root (C:\ISXDailyReportsScrapper)
./build.bat                    # Build everything
./build.bat -target=frontend   # Build frontend only
./build.bat -target=web        # Build web server only
./build.bat -target=release    # Build release version
```

### Rule 3: Logs Are Cleared Automatically
- Every build via `./build.bat` automatically clears log files
- This includes logs in:
  - `dist/logs/`
  - `api/logs/`
  - Root directory `*.log` files

### Rule 4: Build Output Location
- ALL builds output to `dist/` directory
- Source code stays in `api/` and `web/` directories
- Build artifacts in `api/` or `web/` are **FORBIDDEN**

### Rule 5: Verification Required
Run verification script regularly:
```bash
./tools/verify-no-dev-builds.bat
```

---

## 🎯 Quick Reference

### ✅ CORRECT Commands
```bash
# From project root:
./build.bat                         # YES - Builds to dist/
./build.bat -target=clean          # YES - Cleans artifacts
./build.bat -target=test           # YES - Runs tests
./build.bat -target=release        # YES - Release build
```

### ❌ FORBIDDEN Commands
```bash
cd web && npm run build            # NO - NEVER!
cd api && go build ./...           # NO - NEVER!
cd web && next build               # NO - NEVER!
cd web && npm run export           # NO - NEVER!
```

### ✅ ALLOWED Development Commands
```bash
cd web
npm run dev        # OK - Dev server only
npm run test       # OK - Testing
npm run lint       # OK - Linting
npm run type-check # OK - Type checking

cd api
go run cmd/web-licensed/main.go  # OK - Dev run
go test ./...                     # OK - Testing
```

---

## 🔍 Enforcement Mechanisms

### 1. Automatic in build.go
- Clears logs before every build
- Outputs only to `dist/` directory
- Cleans `api/` and `web/` artifacts if found

### 2. Git Ignore Rules
The `.gitignore` blocks all build artifacts in source directories:
- `web/.next/`
- `web/out/`
- `api/**/*.exe`
- `web/**/build/`
- `api/**/dist/`

### 3. Verification Script
`tools/verify-no-dev-builds.bat`:
- Checks for forbidden artifacts
- Removes violations automatically
- Reports compliance status

### 4. Claude Code Enforcement
Claude Code is programmed to:
- **REFUSE** to run forbidden build commands
- **SUGGEST** `./build.bat` instead
- **DELETE** any `web/.next` if found

---

## 📝 For Claude Code and AI Assistants

When asked to build, you MUST:

1. **Check current directory** - Must be project root
2. **Use ONLY** `./build.bat` command
3. **NEVER** navigate to `web` to build
4. **REFUSE** any request to `npm run build` in dev
5. **DELETE** `.next/` or `out/` if found in dev
6. **REMIND** user of proper build procedure

Example response:
```
User: "Build the frontend"
Claude: "I'll build the frontend using the proper build system:
        ./build.bat -target=frontend
        
        Note: I cannot run 'npm run build' directly in web 
        as per project build rules. All builds must use ./build.bat 
        which outputs to the dist/ directory."
```

---

## 🚀 Build Workflow

### Standard Build Process
1. **Position**: Ensure you're in project root
2. **Clean**: Logs are auto-cleared
3. **Build**: Use `./build.bat` with appropriate target
4. **Output**: Find results in `dist/` directory
5. **Verify**: Run `./tools/verify-no-dev-builds.bat`

### Release Build Process
```bash
# From project root only:
./build.bat -target=release
# Creates optimized build in dist/
# Clears all logs automatically
# Ready for deployment
```

---

## ❓ FAQ

**Q: Why can't I build in web?**
A: The api/ and web/ directories are for source code only. Mixing source and build artifacts causes confusion, git issues, and deployment problems.

**Q: What if I accidentally built in api/ or web/?**
A: Run `./tools/verify-no-dev-builds.bat` to clean up, then use `./build.bat` properly.

**Q: Can I run the dev server in web?**
A: Yes! `npm run dev` is allowed and encouraged for development.

**Q: What about CI/CD pipelines?**
A: CI/CD must also use `./build.bat` exclusively.

---

## 📊 Compliance Monitoring

Run this check before any commit:
```bash
# Verify no build artifacts in dev
./tools/verify-no-dev-builds.bat

# If violations found, they're auto-removed
# Then rebuild properly:
./build.bat
```

---

## 🎖️ Remember

- **api/** and **web/** = SOURCE CODE ONLY
- **dist/** = BUILD OUTPUT ONLY
- **./build.bat** = THE ONLY WAY TO BUILD

Following these rules ensures:
- Clean git history
- Consistent builds
- Easy deployments
- No confusion about artifacts
- Professional project structure

---

**Last Updated**: 2025-01-08
**Enforcement Level**: MANDATORY
**Applies To**: All developers, CI/CD, and AI assistants