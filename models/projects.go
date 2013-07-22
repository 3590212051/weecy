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

package models

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Unknwon/gowalker/utils"
	"github.com/astaxie/beego"
	"github.com/coocood/qbs"
)

/*
	GetPopulars returns <num>
		1. Recent viewed
		2. Top rank
		3. Top viewed
		4. Rock this week
	projects and recent updated examples.
*/
func GetPopulars(proNum, exNum int) (error, []*PkgExam,
	[]*PkgInfo, []*PkgInfo, []*PkgInfo, []*PkgInfo) {
	// Connect to database.
	q := connDb()
	defer q.Close()

	var ruExs []*PkgExam
	// q.Limit(examNum).OrderByDesc("created").FindAll(&popExams)

	var rvPros, trPros, tvPros, rtwPros []*PkgInfo
	err := q.Limit(proNum).OrderByDesc("viewed_time").FindAll(&rvPros)
	if err != nil {
		return err, nil, nil, nil, nil, nil
	}
	err = q.Limit(proNum).OrderByDesc("rank").FindAll(&trPros)
	if err != nil {
		return err, nil, nil, nil, nil, nil
	}
	err = q.Limit(proNum).OrderByDesc("views").FindAll(&tvPros)
	if err != nil {
		return err, nil, nil, nil, nil, nil
	}
	return nil, ruExs, rvPros, trPros, tvPros, rtwPros
}

// SaveProject saves package information, declaration and functions;
// update import information.
func SaveProject(pinfo *PkgInfo, pdecl *PkgDecl, pfuncs []*PkgFunc, imports []string) error {
	q := connDb()
	defer q.Close()

	// Save package information.
	info := new(PkgInfo)
	err := q.WhereEqual("path", pinfo.Path).Find(info)
	if err == nil {
		pinfo.Id = info.Id
	}

	_, err = q.Save(pinfo)
	if err != nil {
		beego.Error("models.SaveProject(", pinfo.Path, ") -> Information:", err)
	}

	// Save package declaration.
	decl := new(PkgDecl)
	if pdecl != nil {
		cond := qbs.NewCondition("pid = ?", pinfo.Path).And("tag = ?", pdecl.Tag)
		err = q.Condition(cond).Find(decl)
		if err == nil {
			pdecl.Id = decl.Id
		}

		_, err = q.Save(pdecl)
		if err != nil {
			beego.Error("models.SaveProject(", pinfo.Path, ") -> Declaration:", err)
		}
	}

	// ------------------------------
	// Save package functions.
	// ------------------------------

	if pfuncs != nil {
		// Old package need to clean old data.
		if decl.Id > 0 {
			// Update all old functions' 'IsOle' to be true.
			type pkgFunc struct {
				IsOld bool
			}
			pfunc := new(pkgFunc)
			pfunc.IsOld = true
			_, err = q.WhereEqual("pid", pdecl.Id).Update(pfunc)
		}

		// Save new ones.
		for _, pf := range pfuncs {
			f := new(PkgFunc)
			cond := qbs.NewCondition("pid = ?", pdecl.Id).And("name = ?", pf.Name)
			err = q.Condition(cond).Find(f)
			if err == nil {
				pf.Id = f.Id
			}

			pf.Pid = pdecl.Id
			pf.Path = pinfo.Path
			_, err = q.Save(pf)
			if err != nil {
				beego.Error("models.SaveProject(", pinfo.Path, ") -> Update function(", pf.Name, "):", err)
			}
		}

		if decl.Id > 0 {
			// Delete old ones if exist.
			cond := qbs.NewCondition("pid = ?", pdecl.Id).And("is_old = ?", true)
			_, err = q.Condition(cond).Delete(new(PkgFunc))
			if err != nil {
				beego.Error("models.SaveProject(", pinfo.Path, ") -> Delete functions:", err)
			}
		}
	}

	// ------------- END ------------

	// Don't need to check standard library.
	if imports != nil && !utils.IsGoRepoPath(pinfo.Path) {
		// Update import information.
		for _, v := range imports {
			if !utils.IsGoRepoPath(v) {
				// Only count non-standard library.
				updateImportInfo(q, v, int(pinfo.Id), true)
			}
		}
	}
	return nil
}

// LoadProject returns package declaration.
func LoadProject(path, tag string) (*PkgDecl, error) {
	// Check path length to reduce connect times.
	if len(path) == 0 {
		return nil, errors.New("models.LoadProject( " + path + " ) -> Empty path as not found.")
	}

	// Connect to database.
	q := connDb()
	defer q.Close()

	pdecl := new(PkgDecl)
	cond := qbs.NewCondition("path = ?", path).And("tag = ?", tag)
	err := q.Condition(cond).Find(pdecl)
	return pdecl, err
}

// DeleteProject deletes everything about the path in database, and update import information.
func DeleteProject(path string) error {
	// Check path length to reduce connect times. (except launchpad.net)
	if path[0] != 'l' && len(strings.Split(path, "/")) <= 2 {
		beego.Error("models.DeleteProject(", path, ") -> Short path as not needed")
		return nil
	}

	// Connect to database.
	q := connDb()
	defer q.Close()

	var i1, i2, i3, i4 int64
	// Delete package information.
	info := new(PkgInfo)
	err := q.WhereEqual("path", path).Find(info)
	if err == nil {
		i1, err = q.WhereEqual("path", path).Delete(info)
		if err != nil {
			beego.Error("models.DeleteProject(", path, ") -> Information:", err)
		}
	}

	// TODO: return once and update one by one.
	// Delete package declaration.
	for {
		pdecl := new(PkgDecl)
		err = q.WhereEqual("path", path).Find(pdecl)
		if err != nil {
			// Not found, finish delete.
			break
		}

		i2, err = q.Delete(pdecl)
		if err != nil {
			beego.Error("models.DeleteProject(", path, ") -> Declaration:", err)
		} else if info.Id > 0 && !utils.IsGoRepoPath(path) {
			// Don't need to check standard library.
			// Update import information.
			imports := strings.Split(pdecl.Imports, "|")
			imports = imports[:len(imports)-1]
			for _, v := range imports {
				if !utils.IsGoRepoPath(v) {
					// Only count non-standard library.
					updateImportInfo(q, v, int(info.Id), false)
				}
			}
		}
	}

	// Delete package documentation.
	pdoc := new(PkgDoc)
	i3, err = q.WhereEqual("path", path).Delete(pdoc)
	if err != nil {
		beego.Error("models.DeleteProject(", path, ") -> Documentation:", err)
	}

	// Delete package examples.
	pexam := new(PkgExam)
	i4, err = q.WhereEqual("path", path).Delete(pexam)
	if err != nil {
		beego.Error("models.DeleteProject(", path, ") -> Example:", err)
	}

	if i1+i2+i3+i4 > 0 {
		beego.Info("models.DeleteProject(", path, i1, i2, i3, i4, ")")
	}

	return nil
}

func updateImportInfo(q *qbs.Qbs, path string, pid int, add bool) {
	// Save package information.
	info := new(PkgInfo)
	err := q.WhereEqual("path", path).Find(info)
	if err == nil {
		// Check if pid exists in this project.
		i := strings.Index(info.ImportPid, "$"+strconv.Itoa(pid)+"|")
		switch {
		case i == -1 && add: // Add operation and does not contain.
			info.ImportPid += "$" + strconv.Itoa(pid) + "|"
			info.ImportedNum = len(strings.Split(info.ImportPid, "|")) - 1
			_, err = q.Save(info)
			if err != nil {
				beego.Error("models.updateImportInfo -> add:", path, err)
			}
		case i > -1 && !add: // Delete operation and contains.
			info.ImportPid = strings.Replace(info.ImportPid, "$"+strconv.Itoa(pid)+"|", "", 1)
			info.ImportedNum = len(strings.Split(info.ImportPid, "|")) - 1
			_, err = q.Save(info)
			if err != nil {
				beego.Error("models.updateImportInfo -> delete:", path, err)
			}
		}
	}

	// Error means this project does not exist, simply skip.
}

// FlushCacheProjects saves cache data to database.
func FlushCacheProjects(pinfos []*PkgInfo) {
	q := connDb()
	defer q.Close()

	for _, p := range pinfos {
		info := new(PkgInfo)
		err := q.WhereEqual("path", p.Path).Find(info)
		if err == nil {
			// Shoule always be nil, just in case not exist.
			p.Id = info.Id
			// Limit 10 views each period.
			if p.Views-info.Views > 10 {
				p.Views = info.Views + 10
			}
		}
		_, err = q.Save(p)
		if err != nil {
			beego.Error("models.FlushCacheProjects(", p.Path, ") ->", err)
		}
	}
}
