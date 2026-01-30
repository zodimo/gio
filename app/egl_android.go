// SPDX-License-Identifier: Unlicense OR MIT

//go:build !noopengl

package app

/*
#include <android/native_window_jni.h>
#include <EGL/egl.h>
*/
import "C"

import (
	"unsafe"

	"gioui.org/internal/egl"
)

type androidContext struct {
	win     *window
	eglSurf egl.NativeWindowType
	*egl.Context
}

func init() {
	newAndroidGLESContext = func(w *window) (context, error) {
		ctx, err := egl.NewContext(nil)
		if err != nil {
			return nil, err
		}
		return &androidContext{win: w, Context: ctx}, nil
	}
}

func (c *androidContext) Release() {
	if c.Context != nil {
		c.Context.Release()
		c.Context = nil
	}
}

func (c *androidContext) Refresh() error {
	// Get the native window
	win, _, _ := c.win.nativeWindow()
	newSurf := egl.NativeWindowType(unsafe.Pointer(win))

	// Check if we need to recreate the surface.
	// Only release if we already have a surface and the visual ID changes.
	visID := c.Context.VisualID()

	if c.Context.HasSurface() {
		if err := c.win.setVisual(visID); err != nil {
			return err
		}
		// If setVisual succeeded without changing anything (cache hit),
		// we can keep the existing surface.
		return nil
	}

	// No surface yet, or surface was destroyed - set up for creation.
	c.Context.ReleaseSurface() // Safe to call, no-op if no surface
	if err := c.win.setVisual(visID); err != nil {
		return err
	}
	c.eglSurf = newSurf
	return nil
}

func (c *androidContext) Lock() error {
	// If no surface exists yet, we need to call Refresh() first.
	// This handles the case where Lock() is called before Refresh() in window.go,
	// on emulators that don't support EGL_KHR_surfaceless_context.
	if c.eglSurf == nil && !c.Context.HasSurface() {
		if err := c.Refresh(); err != nil {
			return err
		}
	}
	if c.eglSurf != nil {
		if err := c.Context.CreateSurface(c.eglSurf); err != nil {
			return err
		}
		c.eglSurf = nil
	}
	return c.Context.MakeCurrent()
}

func (c *androidContext) Unlock() {
	c.Context.ReleaseCurrent()
}
