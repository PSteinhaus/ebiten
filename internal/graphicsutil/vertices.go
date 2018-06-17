// Copyright 2017 The Ebiten Authors
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

package graphicsutil

import (
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

var (
	theVerticesBackend = &verticesBackend{}
)

type verticesBackend struct {
	backend []float32
	head    int
}

func (v *verticesBackend) sliceForOneQuad() []float32 {
	const num = 256
	size := 4 * graphics.VertexSizeInBytes() / opengl.Float.SizeInBytes()
	if v.backend == nil {
		v.backend = make([]float32, size*num)
	}
	s := v.backend[v.head : v.head+size]
	v.head += size
	if v.head+size > len(v.backend) {
		v.backend = nil
		v.head = 0
	}
	return s
}

type GeoM interface {
	Apply(x, y float32) (x2, y2 float32)
}

func QuadVertices(width, height int, sx0, sy0, sx1, sy1 int, geom GeoM) []float32 {
	if sx0 >= sx1 || sy0 >= sy1 {
		return nil
	}
	if sx1 <= 0 || sy1 <= 0 {
		return nil
	}

	x0, y0 := float32(0.0), float32(0.0)
	x1, y1 := float32(sx1-sx0), float32(sy1-sy0)

	// it really feels like we should be able to cache this computation
	// but it may not matter.
	w := 1
	h := 1
	for w < width {
		w *= 2
	}
	for h < height {
		h *= 2
	}
	wf := float32(w)
	hf := float32(h)
	u0, v0, u1, v1 := float32(sx0)/wf, float32(sy0)/hf, float32(sx1)/wf, float32(sy1)/hf
	return quadVerticesImpl(u0, v0, u1, v1, x0, y0, x1, y1, geom)
}

func quadVerticesImpl(u0, v0, u1, v1, x0, y0, x1, y1 float32, geom GeoM) []float32 {
	vs := theVerticesBackend.sliceForOneQuad()

	x, y := geom.Apply(x0, y0)
	// Vertex coordinates
	vs[0] = x
	vs[1] = y

	// Texture coordinates: first 2 values indicates the actual coodinate, and
	// the second indicates diagonally opposite coodinates.
	// The second is needed to calculate source rectangle size in shader programs.
	vs[2] = u0
	vs[3] = v0
	vs[4] = u1
	vs[5] = v1

	// and the same for the other three coordinates
	x, y = geom.Apply(x1, y0)
	vs[6] = x
	vs[7] = y
	vs[8] = u1
	vs[9] = v0
	vs[10] = u0
	vs[11] = v1

	x, y = geom.Apply(x0, y1)
	vs[12] = x
	vs[13] = y
	vs[14] = u0
	vs[15] = v1
	vs[16] = u1
	vs[17] = v0

	x, y = geom.Apply(x1, y1)
	vs[18] = x
	vs[19] = y
	vs[20] = u1
	vs[21] = v1
	vs[22] = u0
	vs[23] = v0

	return vs
}
