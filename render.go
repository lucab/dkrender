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
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"

	"github.com/Sirupsen/logrus"
	pkgtar "github.com/coreos/torcx/pkg/tar"
	"github.com/pkg/errors"
)

// SquashLayers squash the image layers and render them as a tgz to a writer.
func SquashLayers(layerFiles []*os.File, streamWriter io.Writer) error {
	layerLen := len(layerFiles)
	if layerLen == 0 {
		return errors.New("got no layers to squash")
	}
	if streamWriter == nil {
		return errors.New("got a nil stream writer")
	}

	// First, render to a temporary directory
	unpackDir, err := ioutil.TempDir(os.TempDir(), "dkrimage_")
	if err != nil {
		return err
	}
	defer os.RemoveAll(unpackDir)
	untarCfg := pkgtar.ExtractCfg{}.Default()

	for i, fp := range layerFiles {
		layerNum := i + 1
		logrus.Infof("untarring layer %d/%d", layerNum, layerLen)
		gz, err := gzip.NewReader(fp)
		if err != nil {
			return errors.Wrapf(err, "uncompressing layer %d/d", layerNum, layerLen)
		}
		defer gz.Close()
		trd := tar.NewReader(gz)

		if err := pkgtar.ChrootUntar(trd, unpackDir, untarCfg); err != nil {
			return errors.Wrapf(err, "untarring layer %d/%d", layerNum, layerLen)
		}
		// TODO(lucab): remove whiteouts
	}

	gw := gzip.NewWriter(streamWriter)
	defer gw.Close()
	if err := pkgtar.Create(gw, unpackDir); err != nil {
		return err
	}
	// Capture failures to close - they are real errors
	if err := gw.Close(); err != nil {
		return err
	}

	return nil
}

// CloseLayers close an array of file pointers to layers.
//
// If multiple errors occurs, the last one is returned
func CloseLayers(layers []*os.File) error {
	var errRet error
	for _, fp := range layers {
		if err := fp.Close(); err != nil {
			errRet = err
		}
	}
	return errRet
}
