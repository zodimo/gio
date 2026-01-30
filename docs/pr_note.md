# Android Emulator Fix - Pull Request Preparation

## Summary

The `fix-android` branch addresses a black screen rendering issue on Android emulators that don't support `EGL_KHR_surfaceless_context`. This issue is documented in [gio issue #671](https://todo.sr.ht/~eliasnaur/gio/671).

## Final Change Set

✅ **Ready for upstream submission** (2 files, minimal footprint)

| File | Changes |
|------|---------|
| [app/egl_android.go](file:///home/jaco/SecondBrain/1-Projects/GoCompose/clones/gio/app/egl_android.go) | +29/-6 lines |
| [internal/egl/egl.go](file:///home/jaco/SecondBrain/1-Projects/GoCompose/clones/gio/internal/egl/egl.go) | +4 lines |

---

## Change Details

### 1. [internal/egl/egl.go](file:///home/jaco/SecondBrain/1-Projects/GoCompose/clones/gio/internal/egl/egl.go) - New API

Added `HasSurface()` method to check if EGL context has a valid surface:

```go
func (c *Context) HasSurface() bool {
    return c.eglSurf != nilEGLSurface
}
```

### 2. [app/egl_android.go](file:///home/jaco/SecondBrain/1-Projects/GoCompose/clones/gio/app/egl_android.go) - Lock() Fix

Modified `Lock()` to call `Refresh()` if no surface exists, preventing failures on emulators without surfaceless context support:

```go
func (c *androidContext) Lock() error {
    if c.eglSurf == nil && !c.Context.HasSurface() {
        if err := c.Refresh(); err != nil {
            return err
        }
    }
    // ... rest of Lock
}
```

### 3. [app/egl_android.go](file:///home/jaco/SecondBrain/1-Projects/GoCompose/clones/gio/app/egl_android.go) - Refresh() Fix

Modified `Refresh()` to preserve existing surfaces when the visual ID hasn't changed:

```go
func (c *androidContext) Refresh() error {
    if c.Context.HasSurface() {
        // Surface exists - only proceed if setVisual needs to change
        if err := c.win.setVisual(visID); err != nil {
            return err
        }
        return nil  // Cache hit - keep existing surface
    }
    // No surface - create new one
    // ...
}
```

---

## Suggested Commit Message

```
app: fix black screen on Android emulators without EGL_KHR_surfaceless_context

On Android emulators that don't support EGL_KHR_surfaceless_context,
calling Lock() before a surface exists causes MakeCurrent() to fail.

This change:
- Adds HasSurface() to egl.Context to check for valid surface
- Modifies Lock() to call Refresh() if no surface exists
- Modifies Refresh() to preserve existing surfaces when visual unchanged

Tested on Android Emulator (API 35) with all Gio example apps
rendering correctly.

Fixes: https://todo.sr.ht/~eliasnaur/gio/671
Signed-off-by: Your Name <your.email@example.com>
```

---

## Submission Steps

> [!IMPORTANT]
> The Gio project uses **email-based patches** submitted via [sr.ht lists](https://lists.sr.ht/~eliasnaur/gio-patches). GitHub PRs are not accepted.

### 1. Squash commits
```bash
git rebase -i origin/main
# Mark all commits except the first as 'squash'
```

### 2. Amend commit message
```bash
git commit --amend
# Use the suggested commit message above
```

### 3. Remove documentation file (optional)
```bash
git rm ANDROID_EMULATOR_FIX.md
git commit --amend --no-edit
```

### 4. Generate patch
```bash
git format-patch origin/main
```

### 5. Send patch
```bash
git send-email --to="~eliasnaur/gio-patches@lists.sr.ht" *.patch
```

Or manually email the patch to `~eliasnaur/gio-patches@lists.sr.ht`.

---

## Notes

- **Backward compatible** - only adds safety checks
- **Minimal footprint** - only 2 files modified
- **Clean API addition** - `HasSurface()` is a useful method for the EGL context