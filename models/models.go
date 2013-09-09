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
	"encoding/base32"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Unknwon/gowalker/utils"
	"github.com/astaxie/beego"
	_ "github.com/coocood/mysql"
	"github.com/coocood/qbs"
)

// A PkgInfo describles a project information.
type PkgInfo struct {
	Id int64

	/*
		- Import path.
		eg.
			github.com/Unknwon/gowalker
		- Name of the project.
		eg.
			gowalker
		- Project synopsis.
		eg.
			Go Walker Server generates Go projects API documentation and Hacker View on the fly.
	*/
	Path     string `qbs:"size:150,index"`
	ProName  string `qbs:"size:50"`
	Synopsis string `qbs:"size:300"`

	/*
		- Indicates whether it's a command line tool or package.
		eg.
			True
		- Indicates whether it belongs to Go standard library.
		eg.
			True
		- Indicates whether it's developed by Go team.
		eg.
			True
	*/
	IsCmd       bool
	IsGoRepo    bool
	IsGoSubrepo bool

	/*
		- All tags of project.
		eg.
			master|||v0.6.2.0718
		- Views of projects.
		eg.
			1342
		- User viewed time(Unix-timestamp).
		eg.
			1374127619
		- Time when information last updated(UTC).
		eg.
			2013-07-16 21:09:27.48932087
	*/
	Tags       string `qbs:"size:150"`
	Views      int64  `qbs:"index"`
	ViewedTime int64
	Created    time.Time `qbs:"index"`

	/*
		- Rank is the benchmark of projects, it's based on BaseRank and views.
		eg.
			826
	*/
	Rank int64 `qbs:"index"`

	/*
		- Package (structure) version.
		eg.
			9
		- Project revision.
		eg.
			8976ce8b2848
		- Project labels.
		eg.
			$tool|
	*/
	PkgVer int
	Ptag   string `qbs:"size:50"`
	Labels string

	/*
		- Number of projects that import this project.
		eg.
			11
		- Number of projects that import this project and do not belong to same project,
			which means import own sub-projects does not count and rank.
		eg.
			3
		- Ids of projects that import this project.
		eg.
			$47|$89|$5464|$8586|$8595|$8787|$8789|$8790|$8918|$9134|$9139|
	*/
	RefNum    int
	RefProNum int
	RefPids   string

	/*
		- Addtional information.
	*/
	Vcs        string `qbs:"size:50"`
	LastUpdate time.Time
	Homepage   string `qbs:"size:100"`
	Issues     int
	Stars      int
	Forks      int
	Note       string
}

// A PkgTag descriables the project revision tag for its sub-projects,
// any project that has this record means it's passed check of Go project,
// and do not need to download whole project archive when refresh.
type PkgTag struct {
	Id   int64
	Path string `qbs:"size:150,index"`
	Ptag string `qbs:"size:50"`
}

// A PkgRock descriables the trending rank of the project.
type PkgRock struct {
	Id    int64
	Pid   int64  `qbs:"index"`
	Path  string `qbs:"size:150"`
	Rank  int64
	Delta int64 `qbs:"index"`
}

func (*PkgRock) Indexes(indexes *qbs.Indexes) {
	indexes.AddUnique("pid", "path")
}

// A PkgExam descriables the user example of the project.
type PkgExam struct {
	Id       int64
	Path     string    `qbs:"size:150,index"`
	Gist     string    `qbs:"size:150"` // Gist path.
	Examples string    // Examples.
	Created  time.Time `qbs:"index"`
}

// PkgDecl is package declaration in database acceptable form.
type PkgDecl struct {
	Id  int64
	Pid int64  `qbs:"index"`
	Tag string // Project tag.

	// Indicate how many JS should be downloaded(JsNum=total num - 1)
	JsNum       int
	IsHasExport bool

	// Top-level declarations.
	IsHasConst, IsHasVar bool

	// Internal declarations.
	//Iconsts, Ifuncs, Itypes, Ivars string

	IsHasExample bool

	Imports, TestImports string // Imports.

	IsHasFile   bool
	IsHasSubdir bool
}

func (*PkgDecl) Indexes(indexes *qbs.Indexes) {
	indexes.AddUnique("pid", "tag")
}

// PkgDoc is package documentation for multi-language usage.
type PkgDoc struct {
	Id   int64
	Path string `qbs:"size:100,index"`
	Lang string // Documentation language.
	Type string
	Doc  string // Documentataion.
}

func (*PkgDoc) Indexes(indexes *qbs.Indexes) {
	indexes.AddUnique("path", "lang", "Type")
}

// PkgFunc represents a package function.
type PkgFunc struct {
	Id       int64
	Pid      int64 `qbs:"index"` // Id of package documentation it belongs to.
	Path     string
	Name     string `qbs:"size:100,index"`
	Doc      string
	IsMaster bool
	IsOld    bool // Indicates if the function no longer exists.
}

func connDb() *qbs.Qbs {
	// 'sql.Open' only returns error when unknown driver, so it's not necessary to check in other places.
	q, _ := qbs.GetQbs()
	return q
}

func setMg() (*qbs.Migration, error) {
	mg, err := qbs.GetMigration()
	return mg, err
}

// InitDb initializes the database.
func InitDb() {
	dbName := utils.Cfg.MustValue("db", "name")
	dbPwd := utils.Cfg.MustValue("db", "pwd_"+runtime.GOOS)

	// Register database.
	qbs.Register("mysql", fmt.Sprintf("%v:%v@%v/%v?charset=utf8&parseTime=true",
		utils.Cfg.MustValue("db", "user"), dbPwd,
		utils.Cfg.MustValue("db", "host"), dbName),
		dbName, qbs.NewMysql())

	// Connect to database.
	q := connDb()
	defer q.Close()

	mg, err := setMg()
	if err != nil {
		panic("models.init -> " + err.Error())
	}
	defer mg.Close()

	// Create data tables.
	mg.CreateTableIfNotExists(new(PkgInfo))
	mg.CreateTableIfNotExists(new(PkgTag))
	mg.CreateTableIfNotExists(new(PkgRock))
	mg.CreateTableIfNotExists(new(PkgExam))

	beego.Trace("Initialized database ->", dbName)
}

func initOld() {
	// mg.CreateTableIfNotExists(new(PkgDecl))
	// mg.CreateTableIfNotExists(new(PkgDoc))
	// mg.CreateTableIfNotExists(new(PkgFunc))
}

// GetGoRepo returns packages in go standard library.
func GetGoRepo() ([]*PkgInfo, error) {
	// Connect to database.
	q := connDb()
	defer q.Close()

	var pkgInfos []*PkgInfo
	condition := qbs.NewCondition("pro_name = ?", "Go")
	err := q.OmitFields("ProName", "IsCmd", "Tags", "Views", "ViewedTime", "Created",
		"Etag", "Labels", "ImportedNum", "ImportPid", "Note").
		Condition(condition).OrderBy("path").FindAll(&pkgInfos)
	infos := make([]*PkgInfo, 0, 30)
	for _, v := range pkgInfos {
		if strings.Index(v.Path, ".") == -1 {
			infos = append(infos, v)
		}
	}
	return infos, err
}

// SearchRawDoc returns results for raw page,
// which are package that import path and synopsis contains keyword.
func SearchRawDoc(key string, isMatchSub bool) (pkgInfos []*PkgInfo, err error) {
	// Connect to database.
	q := connDb()
	defer q.Close()

	// TODO: need to use q.OmitFields to speed up.
	// Check if need to match sub-packages.
	if isMatchSub {
		condition := qbs.NewCondition("pro_name != ?", "Go")
		condition2 := qbs.NewCondition("path like ?", "%"+key+"%").Or("synopsis like ?", "%"+key+"%")
		err = q.Condition(condition).Condition(condition2).Limit(50).OrderByDesc("views").FindAll(&pkgInfos)
		return pkgInfos, err
	}

	condition := qbs.NewCondition("pro_name like ?", "%"+key+"%").Or("synopsis like ?", "%"+key+"%")
	err = q.Condition(condition).Limit(50).OrderByDesc("views").FindAll(&pkgInfos)
	return pkgInfos, err
}

// GetPkgExams returns user examples.
func GetPkgExams(path string) ([]*PkgExam, error) {
	// Connect to database.
	q := connDb()
	defer q.Close()

	var pkgExams []*PkgExam
	err := q.WhereEqual("path", path).FindAll(&pkgExams)
	return pkgExams, err
}

// GetAllExams returns all user examples.
func GetAllExams() ([]*PkgExam, error) {
	// Connect to database.
	q := connDb()
	defer q.Close()

	var pkgExams []*PkgExam
	err := q.OmitFields("Examples", "Created").OrderBy("path").FindAll(&pkgExams)
	return pkgExams, err
}

// GetLabelsPageInfo returns all data that used for labels page.
// One function is for reducing database connect times.
func GetLabelsPageInfo() (WFPros, ORMPros, DBDPros, GUIPros, NETPros, TOOLPros []*PkgInfo, err error) {
	// Connect to database.
	q := connDb()
	defer q.Close()

	condition := qbs.NewCondition("labels like ?", "%$wf|%")
	err = q.Limit(10).Condition(condition).OrderByDesc("views").FindAll(&WFPros)
	condition = qbs.NewCondition("labels like ?", "%$orm|%")
	err = q.Limit(10).Condition(condition).OrderByDesc("views").FindAll(&ORMPros)
	condition = qbs.NewCondition("labels like ?", "%$dbd|%")
	err = q.Limit(10).Condition(condition).OrderByDesc("views").FindAll(&DBDPros)
	condition = qbs.NewCondition("labels like ?", "%$gui|%")
	err = q.Limit(10).Condition(condition).OrderByDesc("views").FindAll(&GUIPros)
	condition = qbs.NewCondition("labels like ?", "%$net|%")
	err = q.Limit(10).Condition(condition).OrderByDesc("views").FindAll(&NETPros)
	condition = qbs.NewCondition("labels like ?", "%$tool|%")
	err = q.Limit(10).Condition(condition).OrderByDesc("views").FindAll(&TOOLPros)
	return WFPros, ORMPros, DBDPros, GUIPros, NETPros, TOOLPros, nil
}

// UpdateLabelInfo updates project label information, returns false if the project does not exist.
func UpdateLabelInfo(path string, label string, add bool) bool {
	// Connect to database.
	q := connDb()
	defer q.Close()

	info := new(PkgInfo)
	err := q.WhereEqual("path", path).Find(info)
	if err != nil {
		return false
	}

	i := strings.Index(info.Labels, "$"+label+"|")
	switch {
	case i == -1 && add: // Add operation and does not contain.
		info.Labels += "$" + label + "|"
		_, err = q.Save(info)
		if err != nil {
			beego.Error("models.UpdateLabelInfo -> add:", path, err)
		}
	case i > -1 && !add: // Delete opetation and contains.
		info.Labels = strings.Replace(info.Labels, "$"+label+"|", "", 1)
		_, err = q.Save(info)
		if err != nil {
			beego.Error("models.UpdateLabelInfo -> delete:", path, err)
		}
	}

	return true
}

var buildPicPattern = regexp.MustCompile(`\[+!+\[+([a-zA-Z ]*)+\]+\(+[a-zA-z]+://[^\s]*`)

// SavePkgExam saves user examples.
func SavePkgExam(gist *PkgExam) error {
	q := connDb()
	defer q.Close()

	// Check if corresponding package exists.
	pinfo := new(PkgInfo)
	err := q.WhereEqual("path", gist.Path).Find(pinfo)
	if err != nil {
		return errors.New(
			fmt.Sprintf("models.SavePkgExam( %s ) -> Package does not exist", gist.Path))
	}

	pexam := new(PkgExam)
	cond := qbs.NewCondition("path = ?", gist.Path).And("gist = ?", gist.Gist)
	err = q.Condition(cond).Find(pexam)
	if err == nil {
		// Check if refresh too frequently(within in 5 minutes).
		if pexam.Created.Add(5 * time.Minute).UTC().After(time.Now().UTC()) {
			return errors.New(
				fmt.Sprintf("models.SavePkgExam( %s ) -> Refresh too frequently(within in 5 minutes)", gist.Path))
		}
		gist.Id = pexam.Id
	}
	gist.Created = time.Now().UTC()

	_, err = q.Save(gist)
	if err != nil {
		return errors.New(
			fmt.Sprintf("models.SavePkgExam( %s ) -> %s", gist.Path, err))
	}

	// Delete 'PkgDecl' in order to generate new page.
	cond = qbs.NewCondition("pid = ?", pinfo.Id).And("tag = ?", "")
	q.Condition(cond).Delete(new(PkgDecl))

	return nil
}

// SavePkgDoc saves readered readme.md file data.
func SavePkgDoc(path, lang string, docBys []byte) {
	q := connDb()
	defer q.Close()

	// Reader readme.
	doc := string(docBys)
	if len(doc) == 0 {
		return
	}

	if doc[0] == '\n' {
		doc = doc[1:]
	}

	pdoc := new(PkgDoc)
	cond := qbs.NewCondition("path = ?", path).And("lang = ?", lang).And("type = ?", "rm")
	q.Condition(cond).Find(pdoc)
	pdoc.Path = path
	pdoc.Lang = lang
	pdoc.Type = "rm"
	pdoc.Doc = base32.StdEncoding.EncodeToString([]byte(doc))
	_, err := q.Save(pdoc)
	if err != nil {
		beego.Error("models.SavePkgDoc -> readme:", err)
	}
}

// LoadPkgDoc loads project introduction documentation.
func LoadPkgDoc(path, lang, docType string) (doc string) {
	q := connDb()
	defer q.Close()

	pdoc := new(PkgDoc)
	cond := qbs.NewCondition("path = ?", path).And("lang = ?", lang).And("type = ?", docType)
	err := q.Condition(cond).Find(pdoc)
	if err == nil {
		return pdoc.Doc
	}

	cond = qbs.NewCondition("path = ?", path).And("lang = ?", "en").And("type = ?", docType)
	err = q.Condition(cond).Find(pdoc)
	if err == nil {
		return pdoc.Doc
	}
	return doc
}

// GetIndexStats returns index page statistic information.
func GetIndexStats() (int64, int64, int64) {
	q := connDb()
	defer q.Close()

	return q.Count(new(PkgInfo)), q.Count(new(PkgDecl)), q.Count(new(PkgFunc))
}

// SearchFunc returns functions that name contains keyword.
func SearchFunc(key string) []*PkgFunc {
	q := connDb()
	defer q.Close()

	var pfuncs []*PkgFunc
	cond := qbs.NewCondition("is_master = ?", true).And("name like ?", "%"+key+"%")
	q.OmitFields("Pid", "IsMaster", "IsOld").Limit(200).Condition(cond).FindAll(&pfuncs)
	return pfuncs
}
