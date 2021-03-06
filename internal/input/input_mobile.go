// Copyright 2016 Hajime Hoshi
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

// +build android ios

package input

import (
	"sync"
)

type Input struct {
	cursorX  int
	cursorY  int
	scrollX  float64
	scrollY  float64
	gamepads [16]gamePad
	touches  []*Touch
	m        sync.RWMutex
}

func (i *Input) RuneBuffer() []rune {
	return nil
}

func (i *Input) IsKeyPressed(key Key) bool {
	return false
}

func (i *Input) MouseWheel() (xoff, yoff float64) {
	return 0, 0
}

func (i *Input) IsMouseButtonPressed(key MouseButton) bool {
	return false
}

func (i *Input) UpdateTouches(touches []*Touch) {
	i.m.Lock()
	i.touches = touches // TODO: Need copy?
	i.m.Unlock()
}
