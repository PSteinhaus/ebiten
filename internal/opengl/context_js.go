// Copyright 2014 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build js

package opengl

import (
	"errors"
	"fmt"

	"github.com/hajimehoshi/gopherwasm/js"

	"github.com/hajimehoshi/ebiten/internal/web"
)

type (
	Texture         js.Value
	Framebuffer     js.Value
	Shader          js.Value
	Program         js.Value
	Buffer          js.Value
	uniformLocation js.Value
)

type attribLocation int

type programID int

var InvalidTexture = Texture(js.Null)

func getProgramID(p Program) programID {
	return programID(js.Value(p).Get("__ebiten_programId").Int())
}

func init() {
	// Accessing the prototype is rquired on Safari.
	c := js.Global.Get("WebGLRenderingContext").Get("prototype")
	VertexShader = ShaderType(c.Get("VERTEX_SHADER").Int())
	FragmentShader = ShaderType(c.Get("FRAGMENT_SHADER").Int())
	ArrayBuffer = BufferType(c.Get("ARRAY_BUFFER").Int())
	ElementArrayBuffer = BufferType(c.Get("ELEMENT_ARRAY_BUFFER").Int())
	DynamicDraw = BufferUsage(c.Get("DYNAMIC_DRAW").Int())
	Triangles = Mode(c.Get("TRIANGLES").Int())
	Lines = Mode(c.Get("LINES").Int())
	Short = DataType(c.Get("SHORT").Int())
	Float = DataType(c.Get("FLOAT").Int())

	zero = operation(c.Get("ZERO").Int())
	one = operation(c.Get("ONE").Int())
	srcAlpha = operation(c.Get("SRC_ALPHA").Int())
	dstAlpha = operation(c.Get("DST_ALPHA").Int())
	oneMinusSrcAlpha = operation(c.Get("ONE_MINUS_SRC_ALPHA").Int())
	oneMinusDstAlpha = operation(c.Get("ONE_MINUS_DST_ALPHA").Int())
}

type context struct {
	gl            js.Value
	loseContext   js.Value
	lastProgramID programID
}

func Init() error {
	if web.IsNodeJS() {
		return fmt.Errorf("opengl: Node.js is not supported")
	}

	if js.Global.Get("WebGLRenderingContext") == js.Undefined {
		return fmt.Errorf("opengl: WebGL is not supported")
	}

	// TODO: Define id?
	canvas := js.Global.Get("document").Call("querySelector", "canvas")
	attr := js.Global.Get("Object").New()
	attr.Set("alpha", true)
	attr.Set("premultipliedAlpha", true)
	gl := canvas.Call("getContext", "webgl", attr)
	if gl == js.Null {
		gl = canvas.Call("getContext", "experimental-webgl", attr)
		if gl == js.Null {
			return fmt.Errorf("opengl: getContext failed")
		}
	}
	c := &Context{}
	c.gl = gl

	// Getting an extension might fail after the context is lost, so
	// it is required to get the extension here.
	c.loseContext = gl.Call("getExtension", "WEBGL_lose_context")
	if c.loseContext != js.Null {
		// This testing function name is temporary.
		js.Global.Set("_ebiten_loseContextForTesting", js.NewCallback(func([]js.Value) {
			c.loseContext.Call("loseContext")
		}))
	}
	theContext = c
	return nil
}

func (c *Context) Reset() error {
	c.locationCache = newLocationCache()
	c.lastTexture = Texture(js.Null)
	c.lastFramebuffer = Framebuffer(js.Null)
	c.lastViewportWidth = 0
	c.lastViewportHeight = 0
	c.lastCompositeMode = CompositeModeUnknown
	gl := c.gl
	gl.Call("enable", gl.Get("BLEND"))
	c.BlendFunc(CompositeModeSourceOver)
	f := gl.Call("getParameter", gl.Get("FRAMEBUFFER_BINDING"))
	c.screenFramebuffer = Framebuffer(f)
	return nil
}

func (c *Context) BlendFunc(mode CompositeMode) {
	if c.lastCompositeMode == mode {
		return
	}
	c.lastCompositeMode = mode
	s, d := operations(mode)
	gl := c.gl
	gl.Call("blendFunc", int(s), int(d))
}

func (c *Context) NewTexture(width, height int) (Texture, error) {
	gl := c.gl
	t := gl.Call("createTexture")
	if t == js.Null {
		return Texture(js.Null), errors.New("opengl: glGenTexture failed")
	}
	gl.Call("pixelStorei", gl.Get("UNPACK_ALIGNMENT"), 4)
	c.BindTexture(Texture(t))

	gl.Call("texParameteri", gl.Get("TEXTURE_2D"), gl.Get("TEXTURE_MAG_FILTER"), gl.Get("NEAREST"))
	gl.Call("texParameteri", gl.Get("TEXTURE_2D"), gl.Get("TEXTURE_MIN_FILTER"), gl.Get("NEAREST"))
	gl.Call("texParameteri", gl.Get("TEXTURE_2D"), gl.Get("TEXTURE_WRAP_S"), gl.Get("CLAMP_TO_EDGE"))
	gl.Call("texParameteri", gl.Get("TEXTURE_2D"), gl.Get("TEXTURE_WRAP_T"), gl.Get("CLAMP_TO_EDGE"))

	// void texImage2D(GLenum target, GLint level, GLenum internalformat,
	//     GLsizei width, GLsizei height, GLint border, GLenum format,
	//     GLenum type, ArrayBufferView? pixels);
	gl.Call("texImage2D", gl.Get("TEXTURE_2D"), 0, gl.Get("RGBA"), width, height, 0, gl.Get("RGBA"), gl.Get("UNSIGNED_BYTE"), nil)

	return Texture(t), nil
}

func (c *Context) bindFramebufferImpl(f Framebuffer) {
	gl := c.gl
	gl.Call("bindFramebuffer", gl.Get("FRAMEBUFFER"), js.Value(f))
}

func (c *Context) FramebufferPixels(f Framebuffer, width, height int) ([]byte, error) {
	gl := c.gl

	c.bindFramebuffer(f)

	pixels := make([]byte, 4*width*height)
	gl.Call("readPixels", 0, 0, width, height, gl.Get("RGBA"), gl.Get("UNSIGNED_BYTE"), pixels)
	if e := gl.Call("getError"); e.Int() != gl.Get("NO_ERROR").Int() {
		return nil, errors.New(fmt.Sprintf("opengl: error: %d", e))
	}
	return pixels, nil
}

func (c *Context) bindTextureImpl(t Texture) {
	gl := c.gl
	gl.Call("bindTexture", gl.Get("TEXTURE_2D"), js.Value(t))
}

func (c *Context) DeleteTexture(t Texture) {
	gl := c.gl
	if !gl.Call("isTexture", js.Value(t)).Bool() {
		return
	}
	if c.lastTexture == t {
		c.lastTexture = Texture(js.Null)
	}
	gl.Call("deleteTexture", js.Value(t))
}

func (c *Context) IsTexture(t Texture) bool {
	gl := c.gl
	return gl.Call("isTexture", js.Value(t)).Bool()
}

func (c *Context) TexSubImage2D(p []byte, x, y, width, height int) {
	gl := c.gl
	// void texSubImage2D(GLenum target, GLint level, GLint xoffset, GLint yoffset,
	//                    GLsizei width, GLsizei height,
	//                    GLenum format, GLenum type, ArrayBufferView? pixels);
	gl.Call("texSubImage2D", gl.Get("TEXTURE_2D"), 0, x, y, width, height, gl.Get("RGBA"), gl.Get("UNSIGNED_BYTE"), p)
}

func (c *Context) NewFramebuffer(t Texture) (Framebuffer, error) {
	gl := c.gl
	f := gl.Call("createFramebuffer")
	c.bindFramebuffer(Framebuffer(f))

	gl.Call("framebufferTexture2D", gl.Get("FRAMEBUFFER"), gl.Get("COLOR_ATTACHMENT0"), gl.Get("TEXTURE_2D"), js.Value(t), 0)
	if s := gl.Call("checkFramebufferStatus", gl.Get("FRAMEBUFFER")); s.Int() != gl.Get("FRAMEBUFFER_COMPLETE").Int() {
		return Framebuffer(js.Null), errors.New(fmt.Sprintf("opengl: creating framebuffer failed: %d", s.Int()))
	}

	return Framebuffer(f), nil
}

func (c *Context) setViewportImpl(width, height int) {
	gl := c.gl
	gl.Call("viewport", 0, 0, width, height)
}

func (c *Context) DeleteFramebuffer(f Framebuffer) {
	gl := c.gl
	if !gl.Call("isFramebuffer", js.Value(f)).Bool() {
		return
	}
	// If a framebuffer to be deleted is bound, a newly bound framebuffer
	// will be a default framebuffer.
	// https://www.khronos.org/opengles/sdk/docs/man/xhtml/glDeleteFramebuffers.xml
	if c.lastFramebuffer == f {
		c.lastFramebuffer = Framebuffer(js.Null)
		c.lastViewportWidth = 0
		c.lastViewportHeight = 0
	}
	gl.Call("deleteFramebuffer", js.Value(f))
}

func (c *Context) NewShader(shaderType ShaderType, source string) (Shader, error) {
	gl := c.gl
	s := gl.Call("createShader", int(shaderType))
	if s == js.Null {
		return Shader(js.Null), fmt.Errorf("opengl: glCreateShader failed: shader type: %d", shaderType)
	}

	gl.Call("shaderSource", js.Value(s), source)
	gl.Call("compileShader", js.Value(s))

	if !gl.Call("getShaderParameter", js.Value(s), gl.Get("COMPILE_STATUS")).Bool() {
		log := gl.Call("getShaderInfoLog", js.Value(s))
		return Shader(js.Null), fmt.Errorf("opengl: shader compile failed: %s", log)
	}
	return Shader(s), nil
}

func (c *Context) DeleteShader(s Shader) {
	gl := c.gl
	gl.Call("deleteShader", js.Value(s))
}

func (c *Context) NewProgram(shaders []Shader) (Program, error) {
	gl := c.gl
	p := gl.Call("createProgram")
	if p == js.Null {
		return Program(js.Null), errors.New("opengl: glCreateProgram failed")
	}
	p.Set("__ebiten_programId", int(c.lastProgramID))
	c.lastProgramID++

	for _, shader := range shaders {
		gl.Call("attachShader", js.Value(p), js.Value(shader))
	}
	gl.Call("linkProgram", js.Value(p))
	if !gl.Call("getProgramParameter", js.Value(p), gl.Get("LINK_STATUS")).Bool() {
		return Program(js.Null), errors.New("opengl: program error")
	}
	return Program(p), nil
}

func (c *Context) UseProgram(p Program) {
	gl := c.gl
	gl.Call("useProgram", js.Value(p))
}

func (c *Context) DeleteProgram(p Program) {
	gl := c.gl
	if !gl.Call("isProgram", p).Bool() {
		return
	}
	gl.Call("deleteProgram", p)
}

func (c *Context) getUniformLocationImpl(p Program, location string) uniformLocation {
	gl := c.gl
	return uniformLocation(gl.Call("getUniformLocation", js.Value(p), location))
}

func (c *Context) UniformInt(p Program, location string, v int) {
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	gl.Call("uniform1i", js.Value(l), v)
}

func (c *Context) UniformFloat(p Program, location string, v float32) {
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	gl.Call("uniform1f", js.Value(l), v)
}

func (c *Context) UniformFloats(p Program, location string, v []float32) {
	gl := c.gl
	l := c.locationCache.GetUniformLocation(c, p, location)
	switch len(v) {
	case 2:
		gl.Call("uniform2f", js.Value(l), v[0], v[1])
	case 4:
		gl.Call("uniform4f", js.Value(l), v[0], v[1], v[2], v[3])
	case 16:
		m := js.Global.Get("Float32Array").New(16)
		for i := range v {
			m.SetIndex(i, v[i])
		}
		gl.Call("uniformMatrix4fv", js.Value(l), false, m)
	default:
		panic("not reached")
	}
}

func (c *Context) getAttribLocationImpl(p Program, location string) attribLocation {
	gl := c.gl
	return attribLocation(gl.Call("getAttribLocation", js.Value(p), location).Int())
}

func (c *Context) VertexAttribPointer(p Program, location string, size int, dataType DataType, stride int, offset int) {
	gl := c.gl
	l := c.locationCache.GetAttribLocation(c, p, location)
	gl.Call("vertexAttribPointer", int(l), size, int(dataType), false, stride, offset)
}

func (c *Context) EnableVertexAttribArray(p Program, location string) {
	gl := c.gl
	l := c.locationCache.GetAttribLocation(c, p, location)
	gl.Call("enableVertexAttribArray", int(l))
}

func (c *Context) DisableVertexAttribArray(p Program, location string) {
	gl := c.gl
	l := c.locationCache.GetAttribLocation(c, p, location)
	gl.Call("disableVertexAttribArray", int(l))
}

func (c *Context) NewArrayBuffer(size int) Buffer {
	gl := c.gl
	b := gl.Call("createBuffer")
	gl.Call("bindBuffer", int(ArrayBuffer), js.Value(b))
	gl.Call("bufferData", int(ArrayBuffer), size, int(DynamicDraw))
	return Buffer(b)
}

func (c *Context) NewElementArrayBuffer(size int) Buffer {
	gl := c.gl
	b := gl.Call("createBuffer")
	gl.Call("bindBuffer", int(ElementArrayBuffer), js.Value(b))
	gl.Call("bufferData", int(ElementArrayBuffer), size, int(DynamicDraw))
	return Buffer(b)
}

func (c *Context) BindBuffer(bufferType BufferType, b Buffer) {
	gl := c.gl
	gl.Call("bindBuffer", int(bufferType), js.Value(b))
}

func (c *Context) ArrayBufferSubData(data []float32) {
	gl := c.gl
	gl.Call("bufferSubData", int(ArrayBuffer), 0, js.ValueOf(data))
}

func (c *Context) ElementArrayBufferSubData(data []uint16) {
	gl := c.gl
	gl.Call("bufferSubData", int(ElementArrayBuffer), 0, js.ValueOf(data))
}

func (c *Context) DeleteBuffer(b Buffer) {
	gl := c.gl
	gl.Call("deleteBuffer", js.Value(b))
}

func (c *Context) DrawElements(mode Mode, len int, offsetInBytes int) {
	gl := c.gl
	gl.Call("drawElements", int(mode), len, gl.Get("UNSIGNED_SHORT"), offsetInBytes)
}

func (c *Context) maxTextureSizeImpl() int {
	gl := c.gl
	return gl.Call("getParameter", gl.Get("MAX_TEXTURE_SIZE")).Int()
}

func (c *Context) Flush() {
	gl := c.gl
	gl.Call("flush")
}

func (c *Context) IsContextLost() bool {
	gl := c.gl
	return gl.Call("isContextLost").Bool()
}

func (c *Context) RestoreContext() {
	if c.loseContext != js.Null {
		c.loseContext.Call("restoreContext")
	}
}
