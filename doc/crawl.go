// Copyright 2013 Unknown
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package doc

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Unknwon/gowalker/models"
	"github.com/Unknwon/gowalker/utils"
	"github.com/astaxie/beego"
)

type crawlResult struct {
	pdoc *Package
	err  error
}

// crawlDoc fetchs package from VCS
func crawlDoc(path string, etag string) (pdoc *Package, err error) {
	// I have no idea what the fuck does this mean.
	if i := strings.Index(path, "/libgo/go/"); i > 0 && utils.IsGoRepoPath(path[i+len("/libgo/go/"):]) {
		// Go Frontend source tree mirror.
		pdoc = nil
		err = errors.New("Go Frontend source tree mirror.")
	} else {
		var pdocNew *Package

		/* TODO:WORKING */

		pdocNew, err = getRepo(httpClient, path, etag)

		// For timeout logic in client.go to work, we cannot leave connections idling. This is ugly.
		httpTransport.CloseIdleConnections()

		if err != errNotModified {
			pdoc = pdocNew
		}
	}

	switch {
	case err == nil:
		if err = SaveProject(pdoc); err != nil {
			beego.Error("doc.SaveProject(", path, "):", err)
		}
	case isNotFound(err):
		// We do not need to delete standard library, so here is fine.
		if err = models.DeleteProject(path); err != nil {
			beego.Error("doc.DeleteProject(", path, "):", err)
		}
	}
	return pdoc, err
}

// getRepo downloads package data.
func getRepo(client *http.Client, importPath string, etag string) (pdoc *Package, err error) {
	const VER_PREFIX = PACKAGE_VER + "-"

	// Check version prefix.
	if strings.HasPrefix(etag, VER_PREFIX) {
		etag = etag[len(VER_PREFIX):]
	} else {
		etag = ""
	}

	switch {
	case utils.IsGoRepoPath(importPath):

		/* TODO:WORKING */

		pdoc, err = getStandardDoc(client, importPath, etag)
	default:
		return nil, errors.New("doc.getRepo(): Test Error.")
	}
	return pdoc, err
}

// SaveProject saves project information to database.
func SaveProject(pdoc *Package) error {

	t := time.Now().UTC().String()
	// Save package information.
	pinfo := &models.PkgInfo{
		Path:       pdoc.ImportPath,
		Synopsis:   pdoc.Synopsis,
		Created:    time.Now().UTC(),
		ViewedTime: t[:19],
		ProName:    pdoc.ProjectName,
	}

	// Save package declaration.
	pdecl := &models.PkgDecl{
		Path:      pdoc.ImportPath,
		Doc:       pdoc.Doc,
		Truncated: pdoc.Truncated,
		Goos:      pdoc.GOOS,
		Goarch:    pdoc.GOARCH,
	}
	var buf bytes.Buffer

	// Consts.
	addValues(&buf, &pdecl.Consts, pdoc.Consts)
	buf.Reset()
	addValues(&buf, &pdecl.Iconsts, pdoc.Iconsts)

	// Variables.
	buf.Reset()
	addValues(&buf, &pdecl.Vars, pdoc.Vars)
	buf.Reset()
	addValues(&buf, &pdecl.Ivars, pdoc.Ivars)

	// Functions.
	buf.Reset()
	addFuncs(&buf, &pdecl.Funcs, pdoc.Funcs)
	buf.Reset()
	addFuncs(&buf, &pdecl.Ifuncs, pdoc.Ifuncs)

	// Types.
	buf.Reset()
	for _, v := range pdoc.Types {
		buf.WriteString(v.Name)
		buf.WriteString("&T#")
		buf.WriteString(v.Doc)
		buf.WriteString("&T#")
		buf.WriteString(v.Decl)
		buf.WriteString("&T#")
		buf.WriteString(v.URL)
		buf.WriteString("&$#")
		// Functions.
		for _, m := range v.Funcs {
			buf.WriteString(m.Name)
			buf.WriteString("&F#")
			buf.WriteString(m.Doc)
			buf.WriteString("&F#")
			buf.WriteString(m.Decl)
			buf.WriteString("&F#")
			buf.WriteString(m.URL)
			buf.WriteString("&F#")
			buf.WriteString(m.Code)
			buf.WriteString("&M#")
		}
		buf.WriteString("&$#")
		for _, m := range v.IFuncs {
			buf.WriteString(m.Name)
			buf.WriteString("&F#")
			buf.WriteString(m.Doc)
			buf.WriteString("&F#")
			buf.WriteString(m.Decl)
			buf.WriteString("&F#")
			buf.WriteString(m.URL)
			buf.WriteString("&F#")
			buf.WriteString(m.Code)
			buf.WriteString("&M#")
		}
		buf.WriteString("&$#")

		// Methods.
		for _, m := range v.Methods {
			buf.WriteString(m.Name)
			buf.WriteString("&F#")
			buf.WriteString(m.Doc)
			buf.WriteString("&F#")
			buf.WriteString(m.Decl)
			buf.WriteString("&F#")
			buf.WriteString(m.URL)
			buf.WriteString("&F#")
			buf.WriteString(m.Code)
			buf.WriteString("&M#")
		}
		buf.WriteString("&$#")
		for _, m := range v.IMethods {
			buf.WriteString(m.Name)
			buf.WriteString("&F#")
			buf.WriteString(m.Doc)
			buf.WriteString("&F#")
			buf.WriteString(m.Decl)
			buf.WriteString("&F#")
			buf.WriteString(m.URL)
			buf.WriteString("&F#")
			buf.WriteString(m.Code)
			buf.WriteString("&M#")
		}
		buf.WriteString("&##")
	}
	pdecl.Types = buf.String()

	// Notes.
	buf.Reset()
	for _, s := range pdoc.Notes {
		buf.WriteString(s)
		buf.WriteString("&$#")
	}
	pdecl.Notes = buf.String()

	// Dirs.
	buf.Reset()
	for _, s := range pdoc.Dirs {
		buf.WriteString(s)
		buf.WriteString("&$#")
	}
	pdecl.Dirs = buf.String()

	// Imports.
	buf.Reset()
	for _, s := range pdoc.Imports {
		buf.WriteString(s)
		buf.WriteString("&$#")
	}
	pdecl.Imports = buf.String()

	buf.Reset()
	for _, s := range pdoc.TestImports {
		buf.WriteString(s)
		buf.WriteString("&$#")
	}
	pdecl.TestImports = buf.String()

	// Files.
	buf.Reset()
	for _, s := range pdoc.Files {
		buf.WriteString(s)
		buf.WriteString("&$#")
	}
	pdecl.Files = buf.String()

	buf.Reset()
	for _, s := range pdoc.TestFiles {
		buf.WriteString(s)
		buf.WriteString("&$#")
	}
	pdecl.TestFiles = buf.String()

	// Save package documentation.
	doc := &models.PkgDoc{
		Path: pdoc.ImportPath,
		Lang: "zh",
	}

	// Documentataion.
	buf.Reset()
	for k, v := range pdoc.TDoc {
		buf.WriteString(k)
		buf.WriteByte(':')
		buf.WriteString(v)
		buf.WriteString("&$#")
	}

	for k, v := range pdoc.IDoc {
		buf.WriteString(k)
		buf.WriteByte(':')
		buf.WriteString(v)
		buf.WriteString("&$#")
	}
	doc.Doc = buf.String()

	err := models.SaveProject(pinfo, pdecl, doc)
	return err
}

func addValues(buf *bytes.Buffer, pvals *string, vals []*Value) {
	for _, v := range vals {
		buf.WriteString(v.Name)
		buf.WriteString("&V#")
		buf.WriteString(v.Doc)
		buf.WriteString("&V#")
		buf.WriteString(v.Decl)
		buf.WriteString("&V#")
		buf.WriteString(v.URL)
		buf.WriteString("&$#")
	}
	*pvals = buf.String()
}

func addFuncs(buf *bytes.Buffer, pfuncs *string, funcs []*Func) {
	for _, v := range funcs {
		buf.WriteString(v.Name)
		buf.WriteString("&F#")
		buf.WriteString(v.Doc)
		buf.WriteString("&F#")
		buf.WriteString(v.Decl)
		buf.WriteString("&F#")
		buf.WriteString(v.URL)
		buf.WriteString("&F#")
		buf.WriteString(v.Code)
		buf.WriteString("&$#")
	}
	*pfuncs = buf.String()
}
