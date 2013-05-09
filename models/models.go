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

// Package models implemented database access funtions.

package models

import (
	"database/sql"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/astaxie/beego"
	"github.com/coocood/qbs"
	_ "github.com/mattn/go-sqlite3"
)

const (
	DB_NAME         = "./data/gowalker.db"
	_SQLITE3_DRIVER = "sqlite3"
)

// PkgInfo is package information.
type PkgInfo struct {
	Path       string `qbs:"pk,index"` // Import path of package.
	Synopsis   string
	Views      int64     `qbs:"index"`
	Created    time.Time `qbs:"index"` // Time when information last updated.
	ViewedTime int64     // User viewed time(Unix-timestamp).
	ProName    string    // Name of the project.
	Etag       string    // Revision tag.
}

// PkgDecl is package declaration in database acceptable form.
type PkgDecl struct {
	Path      string `qbs:"pk,index"` // Import path of package.
	Doc       string // Package documentation.
	Truncated bool   // True if package documentation is incomplete.

	// Environment.
	Goos, Goarch string

	// Top-level declarations.
	Consts, Funcs, Types, Vars string

	// Internal declarations.
	Iconsts, Ifuncs, Itypes, Ivars string

	Notes            string // Source code notes.
	Files, TestFiles string // Source files.
	Dirs             string // Subdirectories

	Imports, TestImports string // Imports.
}

// PkgDoc is package documentation for multi-language usage.
type PkgDoc struct {
	Path string `qbs:"pk,index"` // Import path of package.
	Lang string // Documentation language.
	Doc  string // Documentataion.
}

func connDb() (*qbs.Qbs, error) {
	db, err := sql.Open(_SQLITE3_DRIVER, DB_NAME)
	q := qbs.New(db, qbs.NewSqlite3())
	return q, err
}

func setMg() (*qbs.Migration, error) {
	db, err := sql.Open(_SQLITE3_DRIVER, DB_NAME)
	mg := qbs.NewMigration(db, DB_NAME, qbs.NewSqlite3())
	return mg, err
}

func init() {
	// Initialize database.
	os.Mkdir("./data", os.ModePerm)

	// Connect to database.
	q, err := connDb()
	if err != nil {
		beego.Error("models.init():", err)
	}
	defer q.Db.Close()

	mg, err := setMg()
	if err != nil {
		beego.Error("models.init():", err)
	}
	defer mg.Db.Close()

	// Create data tables.
	mg.CreateTableIfNotExists(new(PkgInfo))
	mg.CreateTableIfNotExists(new(PkgDecl))
	mg.CreateTableIfNotExists(new(PkgDoc))

	beego.Trace("Initialized database ->", DB_NAME)
}

// GetProInfo returns package information from database.
func GetPkgInfo(path string) (*PkgInfo, error) {
	// Connect to database.
	q, err := connDb()
	if err != nil {
		beego.Error("models.GetPkgInfo():", err)
	}
	defer q.Db.Close()

	pinfo := new(PkgInfo)
	err = q.WhereEqual("path", path).Find(pinfo)

	return pinfo, err
}

// SaveProject save package information, declaration, documentation to database.
func SaveProject(pinfo *PkgInfo, pdecl *PkgDecl, pdoc *PkgDoc) error {
	// Connect to database.
	q, err := connDb()
	if err != nil {
		beego.Error("models.SaveProject():", err)
	}
	defer q.Db.Close()

	// Save package information.
	_, err = q.Save(pinfo)

	// When 'path' as primary key, don't need to use following code.
	/*	info := new(PkgInfo)
		err = q.WhereEqual("path", pinfo.Path).Find(info)
		if err != nil {
			_, err = q.Save(pinfo)
		} else {
			info.Synopsis = pinfo.Synopsis
			info.Created = pinfo.Created
			info.ViewedTime = pinfo.ViewedTime
			info.ProName = pinfo.ProName
			info.Etag = pinfo.Etag
			_, err = q.Save(info)
		}*/
	if err != nil {
		beego.Error("models.SaveProject(): Information:", err)
	}

	// Save package declaration
	_, err = q.Save(pdecl)
	if err != nil {
		beego.Error("models.SaveProject(): Declaration:", err)
	}

	// Save package documentation
	if len(pdoc.Doc) > 0 {
		_, err = q.Save(pdoc)
		if err != nil {
			beego.Error("models.SaveProject(): Documentation:", err)
		}
	}

	return nil
}

// DeleteProject deletes everything about the path in database.
func DeleteProject(path string) error {
	// Check path length to reduce connect times. (except launchpad.net)
	if path[0] != 'l' && len(strings.Split(path, "/")) <= 2 {
		return errors.New("models.DeleteProject(): Short path as not needed.")
	}

	// Connect to database.
	q, err := connDb()
	if err != nil {
		beego.Error("models.SaveProject():", err)
	}
	defer q.Db.Close()

	// Delete package information.
	info := &PkgInfo{Path: path}
	_, err = q.Delete(info)
	if err != nil {
		beego.Error("models.DeleteProject(): Information:", err)
	}

	// Delete package declaration
	pdecl := &PkgDecl{Path: path}
	_, err = q.Delete(pdecl)
	if err != nil {
		beego.Error("models.DeleteProject(): Declaration:", err)
	}

	// Delete package documentation
	pdoc := &PkgDoc{Path: path}
	_, err = q.Delete(pdoc)
	if err != nil {
		beego.Error("models.DeleteProject(): Documentation:", err)
	}

	beego.Error("models.DeleteProject(", path, ")")
	return nil
}

// LoadProject gets package declaration from database.
func LoadProject(path string) (*PkgDecl, error) {
	// Check path length to reduce connect times.
	if len(path) == 0 {
		return nil, errors.New("models.LoadProject(): Empty path as not found.")
	}

	// Connect to database.
	q, err := connDb()
	if err != nil {
		beego.Error("models.LoadProject():", err)
	}
	defer q.Db.Close()

	pdecl := &PkgDecl{Path: path}
	err = q.WhereEqual("path", path).Find(pdecl)
	return pdecl, err
}

// GetRecentPros gets recent viewed projects from database
func GetRecentPros(num int) ([]*PkgInfo, error) {
	// Connect to database.
	q, err := connDb()
	if err != nil {
		beego.Error("models.GetRecentPros():", err)
	}
	defer q.Db.Close()

	var pkgInfos []*PkgInfo
	err = q.Where("views > ?", 0).Limit(num).OrderByDesc("viewed_time").FindAll(&pkgInfos)
	return pkgInfos, err
}

// AddViews add views in database by 1 each time
func AddViews(pinfo *PkgInfo) error {
	// Connect to database.
	q, err := connDb()
	if err != nil {
		beego.Error("models.AddViews():", err)
	}
	defer q.Db.Close()

	pinfo.Views++
	_, err = q.Save(pinfo)
	return err
}

// GetPopularPros gets most viewed projects from database
func GetPopularPros() ([]*PkgInfo, error) {
	// Connect to database.
	q, err := connDb()
	if err != nil {
		beego.Error("models.GetPopularPros():", err)
	}
	defer q.Db.Close()

	var pkgInfos []*PkgInfo
	err = q.Where("views > ?", 0).Limit(25).OrderByDesc("views").FindAll(&pkgInfos)
	return pkgInfos, err
}

// GetGoRepo gets go standard library
func GetGoRepo() ([]*PkgInfo, error) {
	// Connect to database.
	q, err := connDb()
	if err != nil {
		beego.Error("models.GetGoRepo():", err)
	}
	defer q.Db.Close()

	var pkgInfos []*PkgInfo
	condition := qbs.NewCondition("pro_name = ?", "Go").And("views > ?", 0)
	err = q.Condition(condition).OrderBy("path").FindAll(&pkgInfos)
	return pkgInfos, err
}

// SearchDoc gets packages that contain keyword
func SearchDoc(key string) ([]*PkgInfo, error) {
	// Connect to database.
	q, err := connDb()
	if err != nil {
		beego.Error("models.SearchDoc():", err)
	}
	defer q.Db.Close()

	var pkgInfos []*PkgInfo
	condition := qbs.NewCondition("path like ?", "%"+key+"%").And("views > ?", 0)
	err = q.Condition(condition).OrderBy("path").FindAll(&pkgInfos)
	return pkgInfos, err
}

// GetAllPkgs gets all packages in database
func GetAllPkgs() ([]*PkgInfo, error) {
	// Connect to database.
	q, err := connDb()
	if err != nil {
		beego.Error("models.GetAllPkgs():", err)
	}
	defer q.Db.Close()

	var pkgInfos []*PkgInfo
	err = q.Where("views > ?", 0).OrderByDesc("pro_name").OrderBy("views").FindAll(&pkgInfos)
	return pkgInfos, err
}
