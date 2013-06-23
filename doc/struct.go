// Copyright 2011 Gary Burd
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
	"go/doc"
	"go/token"
	"os"
	"time"
)

// Value represents constants and variable
type Value struct {
	Name          string // Value name.
	Doc           string
	Decl, FmtDecl string // Normal and formatted form of declaration.
	URL           string // VCS URL.
}

// Func represents functions
type Func struct {
	Name          string // Function name.
	Doc           string
	Decl, FmtDecl string // Normal and formatted form of declaration.
	URL           string // VCS URL.
	Code          string // Source code.
	IsHasExam     bool
	Exam          *Example
}

// Type represents structs and interfaces
type Type struct {
	Name          string // Type name.
	Doc           string
	Decl, FmtDecl string  // Normal and formatted form of declaration.
	URL           string  // VCS URL.
	Funcs         []*Func // Exported functions that return this type.
	IFuncs        []*Func // Internal functions that return this type.
	Methods       []*Func // Exported methods.
	IMethods      []*Func // Internal methods.
	IsHasExam     bool
	Exam          *Example
}

// Example represents function examples.
type Example struct {
	Name string
	Doc  string
	Code string
	//Play   string
	Output      string
	IsHasOutput bool
}

// PACKAGE_VER is modified when previously stored packages are invalid.
const PACKAGE_VER = "1"

// Package represents full information and documentation for a package.
type Package struct {
	// Package import path.
	ImportPath string

	// Tags of project.
	Tag  string
	Tags []string

	// Synopsis and full documentation for package.
	Synopsis string
	Doc      string

	// Indicates whether it's a main package or not.
	IsCmd bool

	// Name of the project.
	ProjectName string

	// Time when information last updated.
	Created time.Time

	Views, ViewedTime int64 // User viewed time(Unix-timestamp).

	Etag, Labels string // Revision tag and project labels.

	// Top-level declarations.
	Consts []*Value
	Funcs  []*Func
	Types  []*Type
	Vars   []*Value

	// Internal declarations.
	Iconsts []*Value
	Ifuncs  []*Func
	Itypes  []*Type
	Ivars   []*Value

	Examples         []*Example // Function or method example.
	Notes            []string   // Source code notes.
	Files, TestFiles []string   // Source files.
	Dirs             []string   // Subdirectories

	Imports, TestImports []string // Imports.

	ImportedNum int    // Number of packages that imports this project.
	ImportPid   string // Packages id of packages that imports this project.
}

// source is source code file.
type source struct {
	name      string
	browseURL string
	rawURL    string
	data      []byte
}

func (s *source) Name() string       { return s.name }
func (s *source) Size() int64        { return int64(len(s.data)) }
func (s *source) Mode() os.FileMode  { return 0 }
func (s *source) ModTime() time.Time { return time.Time{} }
func (s *source) IsDir() bool        { return false }
func (s *source) Sys() interface{}   { return nil }

// walker holds the state used when building the documentation.
type walker struct {
	lineFmt  string
	pdoc     *Package
	examples []*doc.Example      // Function or method example.
	srcLines map[string][]string // Source files with line arrays.
	srcs     map[string]*source  // Source files.
	fset     *token.FileSet
	buf      []byte // scratch space for printNode method.
}
