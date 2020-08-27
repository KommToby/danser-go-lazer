package framebuffer

import (
	"github.com/wieku/danser-go/app/settings"
	"runtime"

	"github.com/faiface/mainthread"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/wieku/danser-go/framework/graphics/texture"
)

var bindHistory []int32

// Framebuffer is a fixed resolution texture that you can draw on.
type Framebuffer struct {
	obj           uint32
	last          int32
	tex           *texture.TextureSingle
	multisampled  bool
	helperObj     uint32
	helperTexture uint32
}

// NewFrame creates a new fully transparent Framebuffer with given dimensions in pixels.
func NewFrame(width, height int, smooth, depth bool) *Framebuffer {
	f := new(Framebuffer)

	f.tex = texture.NewTextureSingle(width, height, 0)

	gl.GenFramebuffers(1, &f.obj)

	f.Begin()
	f.tex.Bind(0)
	gl.FramebufferTextureLayerARB(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, f.tex.GetID(), 0, 0)

	if depth {
		var depthRenderBuffer uint32
		gl.GenRenderbuffers(1, &depthRenderBuffer)
		gl.BindRenderbuffer(gl.RENDERBUFFER, depthRenderBuffer)
		gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT, int32(width), int32(height))
		gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, depthRenderBuffer)
	}

	f.End()

	runtime.SetFinalizer(f, (*Framebuffer).delete)

	return f
}

func NewFrameMultisample(width, height int, smooth, depth bool) *Framebuffer {
	f := new(Framebuffer)

	f.tex = texture.NewTextureSingle(width, height, 0)

	gl.GenFramebuffers(1, &f.helperObj)

	gl.GetIntegerv(gl.FRAMEBUFFER_BINDING, &f.last)
	bindHistory = append(bindHistory, f.last)
	gl.BindFramebuffer(gl.FRAMEBUFFER, f.helperObj)

	f.tex.Bind(0)
	gl.FramebufferTextureLayerARB(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, f.tex.GetID(), 0, 0)

	lst := bindHistory[len(bindHistory)-1]
	bindHistory = bindHistory[:len(bindHistory)-1]
	gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(lst))

	gl.GenFramebuffers(1, &f.obj)

	f.Begin()

	gl.GenTextures(1, &f.helperTexture)
	gl.BindTexture(gl.TEXTURE_2D_MULTISAMPLE, f.helperTexture)

	gl.TexImage2DMultisample(gl.TEXTURE_2D_MULTISAMPLE, settings.Graphics.MSAA, gl.RGBA8, int32(width), int32(height), true)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D_MULTISAMPLE, f.helperTexture, 0)

	if depth {
		var depthRenderBuffer uint32
		gl.GenRenderbuffers(1, &depthRenderBuffer)
		gl.BindRenderbuffer(gl.RENDERBUFFER, depthRenderBuffer)
		gl.RenderbufferStorageMultisample(gl.RENDERBUFFER, settings.Graphics.MSAA, gl.DEPTH_COMPONENT, int32(width), int32(height))
		gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, depthRenderBuffer)
	}

	f.End()

	runtime.SetFinalizer(f, (*Framebuffer).delete)

	f.multisampled = true
	return f
}

func (f *Framebuffer) delete() {
	mainthread.CallNonBlock(func() {
		gl.DeleteFramebuffers(1, &f.obj)
	})
}

// ID returns the OpenGL framebuffer ID of this Framebuffer.
func (f *Framebuffer) ID() uint32 {
	return f.obj
}

// Begin binds the Framebuffer. All draw operations will target this Framebuffer until End is called.
func (f *Framebuffer) Begin() {
	gl.GetIntegerv(gl.FRAMEBUFFER_BINDING, &f.last)
	bindHistory = append(bindHistory, f.last)
	gl.BindFramebuffer(gl.FRAMEBUFFER, f.obj)
}

// End unbinds the Framebuffer. All draw operations will go to whatever was bound before this Framebuffer.
func (f *Framebuffer) End() {
	if f.multisampled {
		gl.BindFramebuffer(gl.READ_FRAMEBUFFER, f.obj)
		gl.ReadBuffer(gl.COLOR_ATTACHMENT0)
		gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, f.helperObj)
		gl.DrawBuffer(gl.COLOR_ATTACHMENT0)

		gl.BlitFramebuffer(0, 0, f.tex.GetWidth(), f.tex.GetHeight(), 0, 0, f.tex.GetWidth(), f.tex.GetHeight(), gl.COLOR_BUFFER_BIT, gl.LINEAR)

		gl.BindFramebuffer(gl.READ_FRAMEBUFFER, 0)
		gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, 0)
	}

	lst := bindHistory[len(bindHistory)-1]
	bindHistory = bindHistory[:len(bindHistory)-1]
	gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(lst))
}

// Texture returns the Framebuffer's underlying Texture that the Framebuffer draws on.
func (f *Framebuffer) Texture() texture.Texture {
	return f.tex
}
