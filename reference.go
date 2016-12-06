// Copyright (c) 2013 Patrick Gundlach, speedata (Berlin, Germany)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package gogit

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Reference struct {
	Name       string
	Oid        *Oid
	dest       string
	repository *Repository
}

func (ref *Reference) resolveInfo() (*Reference, error) {
	destRef := new(Reference)
	destRef.Name = ref.dest

	destpath := filepath.Join(ref.repository.Path, "info", "refs")
	f, err := os.Open(destpath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sha1, err := findRef(f, ref.dest)
	if err == errRefNotFound {
		return nil, errors.New("Could not resolve info/refs")
	}
	if err != nil {
		return nil, err
	}
	oid, err := NewOidFromString(sha1)
	if err != nil {
		return nil, err
	}
	destRef.Oid = oid
	return destRef, nil
}

var errRefNotFound = errors.New("ref not found")

// findRef parses a list of SHA1/ref pairs such as those
// found in info/refs, packed-refs, or the output of git ls-remote.
// It looks for ref and returns the corresponding SHA1 string.
func findRef(r io.Reader, ref string) (string, error) {
	refb := []byte(ref)
	scan := bufio.NewScanner(r)
	for scan.Scan() {
		line := bytes.TrimSpace(scan.Bytes())
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		// It appears that info/refs uses tabs to separate sha1s,
		// whereas packed-refs uses spaces. Be agnostic.
		ff := bytes.Fields(line)
		if len(ff) != 2 || len(ff[0]) != 40 || !bytes.Equal(refb, ff[1]) {
			continue
		}
		// Found a well-formed match.
		return string(ff[0]), nil
	}
	if err := scan.Err(); err != nil {
		return "", err
	}
	return "", errRefNotFound
}

// A typical Git repository consists of objects (path objects/ in the root directory)
// and of references to HEAD, branches, tags and such.
func (repos *Repository) LookupReference(name string) (*Reference, error) {
	// First we need to find out what's in the text file. It could be something like
	//     ref: refs/heads/master
	// or just a SHA1 such as
	//     1337a1a1b0694887722f8bd0e541bd0f6567a471
	ref := new(Reference)
	ref.repository = repos
	ref.Name = name
	f, err := ioutil.ReadFile(filepath.Join(repos.Path, name))
	if err != nil {
		if os.IsNotExist(err) {
			// Try looking it up in info/refs.
			ref.dest = name
			return ref.resolveInfo()
		}
		return nil, err
	}
	s := string(bytes.TrimSpace(f))
	if !strings.HasPrefix(s, "ref: ") {
		if len(s) != 40 {
			return nil, fmt.Errorf("malformed ref %q", s)
		}
		// let's assume this is a SHA1
		oid, err := NewOidFromString(s)
		if err != nil {
			return nil, err
		}
		ref.Oid = oid
		return ref, nil
	}
	// yes, it's "ref: something". Now let's lookup "something"
	ref.dest = strings.TrimPrefix(s, "ref: ")
	return repos.LookupReference(ref.dest)
}

// For compatibility with git2go. Return Oid from referece (same as getting .Oid directly)
func (r *Reference) Target() *Oid {
	return r.Oid
}
