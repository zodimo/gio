# Android Emulator Rendering Fixes

This document tracks issues discovered and fixes applied to resolve black screen rendering on Android emulators that don't support `EGL_KHR_surfaceless_context`.

## Problem Summary

When running Gio applications on certain Android emulators, the screen renders black despite the application correctly painting frames (visible in logcat as "Painted CYAN Screen"). The root cause is an infinite loop in the EGL context initialization sequence.

## Root Cause Analysis

### Issue 1: Lock() Called Before Refresh()

**Location:** `app/window.go:148` and `app/egl_android.go`

**Problem:** In `validateAndProcess()`, when a new context is created:
```go
if w.ctx, err = w.driver.NewContext(); err != nil {
    return err
}
if err = w.ctx.Lock(); err != nil {  // <-- Lock called immediately
    w.destroyGPU()
    return err
}
sync = true  // <-- sync set AFTER Lock
```

On emulators without `EGL_KHR_surfaceless_context` support:
1. `NewContext()` creates an EGL context (no surface yet)
2. `Lock()` calls `MakeCurrent()` which requires a valid surface
3. `MakeCurrent()` fails with: "no surface created yet EGL_KHR_surfaceless_context is not supported"
4. Error causes `destroyGPU()` to be called
5. Loop repeats indefinitely

**Fix:** Modified `Lock()` in `egl_android.go` to call `Refresh()` internally if no surface exists:

```go
func (c *androidContext) Lock() error {
    // If no surface exists yet, we need to call Refresh() first.
    // This happens when Lock() is called before Refresh() in window.go,
    // and the emulator doesn't support EGL_KHR_surfaceless_context.
    if c.eglSurf == nil && !c.Context.HasSurface() {
        if err := c.Refresh(); err != nil {
            return err
        }
    }
    // ... rest of Lock()
}
```

### Issue 2: Refresh() Destroys Existing Surfaces

**Location:** `app/egl_android.go:Refresh()`

**Problem:** The original `Refresh()` always calls `ReleaseSurface()` at the start:
```go
func (c *androidContext) Refresh() error {
    c.Context.ReleaseSurface()  // Always destroys surface!
    // ...
}
```

When `Refresh()` is called multiple times (once from our `Lock()` fix, once from `window.go` when `sync=true`), the second call destroys the surface we just created.

**Fix:** Modified `Refresh()` to preserve existing surfaces when the visual ID hasn't changed:

```go
func (c *androidContext) Refresh() error {
    if c.Context.HasSurface() {
        // Surface exists - only recreate if visual changes
        if err := c.win.setVisual(visID); err != nil {
            return err
        }
        // Cache hit - keep existing surface
        return nil
    }
    // No surface - create new one
    c.Context.ReleaseSurface()
    // ...
}
```

## Files Modified

1. **`app/egl_android.go`**
   - Modified `Lock()` to call `Refresh()` if no surface exists
   - Modified `Refresh()` to preserve existing surfaces

2. **`internal/egl/egl.go`**
   - Added `HasSurface()` method to check if surface exists


## New API Added

### `egl.Context.HasSurface() bool`

Returns `true` if the EGL context has a valid surface attached.

```go
func (c *Context) HasSurface() bool {
    return c.eglSurf != nilEGLSurface
}
```

## Testing

**Status: ✅ VERIFIED WORKING**

Tested on:
- Android Emulator (API 35) without `EGL_KHR_surfaceless_context` support
- Date: 2026-01-30

Results:
1. Cyan screen test: ✅ Renders correctly
2. Text rendering (Material H1): ✅ Renders correctly
3. Background fill: ✅ Renders correctly
4. Layout (Center): ✅ Works correctly
5. No infinite loop in logs: ✅ Confirmed
6. **Full Kitchen Demo**: ✅ All components render correctly
   - Top App Bar: ✅
   - Buttons (Filled, Outlined, Text, Elevated): ✅
   - Icon Buttons: ✅
   - Bottom Navigation Bar: ✅
   - Multiple screens/categories: ✅

## Notes for Upstream

1. The fix is backward compatible - it only adds extra safety checks
2. The `HasSurface()` method is a clean addition to the EGL API
3. Consider whether `window.go` should call `Refresh()` before `Lock()` when sync is needed (architectural fix)

## Debug Logging

All debug logging has been removed. The implementation is ready for upstream submission.
