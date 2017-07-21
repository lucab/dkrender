// Copyright 2017 CoreOS Inc.
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

package dkrender

import (
	"io"
	"os"
	"testing"
)

// SquashLayers squash the image layers and render them as a tgz to a writer.
func TestSquashLayers(t *testing.T) {
	cases := []struct {
		desc   string
		layers []*os.File
		writer io.Writer
		errExp bool
	}{
		{
			"no layers",
			nil,
			nil,
			true,
		},
		{
			"nil writer",
			[]*os.File{nil},
			nil,
			true,
		},
	}

	for _, tt := range cases {
		t.Logf("testing %q", tt.desc)
		err := SquashLayers(tt.layers, tt.writer)
		if tt.errExp && err == nil {
			t.Errorf("expected error, got nil")
		}
		if err != nil && !tt.errExp {
			t.Errorf("unexpected error: %s", err)
		}

	}
}
