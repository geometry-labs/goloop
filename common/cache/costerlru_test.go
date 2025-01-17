/*
 * Copyright 2023 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type value int

func (v value) Cost() int {
	return int(v)
}

func assertGetEqual[K comparable, V Coster](t assert.TestingT, exp1 V, exp2 bool, sc *CosterLRU[K, V], key K) {
	val, ok := sc.Get(key)
	assert.Equal(t, exp1, val)
	assert.Equal(t, exp2, ok)
}

func TestCosterLRU_Basics(t *testing.T) {
	c := NewCosterLRU[string, value](10)

	assertGetEqual(t, 0, false, c, "k0")

	c.Put("k0", 3)
	assertGetEqual(t, 3, true, c, "k0")

	c.Put("k1", 4)
	assertGetEqual(t, 3, true, c, "k0")
	assertGetEqual(t, 4, true, c, "k1")

	c.Put("k2", 5)
	assertGetEqual(t, 0, false, c, "k0")
	assertGetEqual(t, 4, true, c, "k1")
	assertGetEqual(t, 5, true, c, "k2")
}

func TestCosterLRU_RejectTooHeavyValue(t *testing.T) {
	c := NewCosterLRU[string, value](10)
	c.Put("k0", 11)
	assertGetEqual(t, 0, false, c, "k0")

	c.Put("k0", 10)
	assertGetEqual(t, 10, true, c, "k0")
}

func TestCosterLRU_Update(t *testing.T) {
	c := NewCosterLRU[string, value](10)

	c.Put("k0", 3)
	c.Put("k1", 3)
	c.Put("k2", 3)
	assertGetEqual(t, 3, true, c, "k0")
	assertGetEqual(t, 3, true, c, "k1")
	assertGetEqual(t, 3, true, c, "k2")

	c.Put("k2", 4)
	assertGetEqual(t, 3, true, c, "k0")
	assertGetEqual(t, 3, true, c, "k1")
	assertGetEqual(t, 4, true, c, "k2")

	c.Put("k2", 5)
	assertGetEqual(t, 0, false, c, "k0")
	assertGetEqual(t, 3, true, c, "k1")
	assertGetEqual(t, 5, true, c, "k2")

	c.Put("k0", 11)
	assertGetEqual(t, 0, false, c, "k0")
	assertGetEqual(t, 3, true, c, "k1")
	assertGetEqual(t, 5, true, c, "k2")
}

func TestMakeCosterLRU(t *testing.T) {
	c := MakeCosterLRU[string, value](10)
	c.Put("k0", 3)
	assertGetEqual(t, 3, true, &c, "k0")
}
