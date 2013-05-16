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

// Package controllers implemented controller methods of beego.

package controllers

import (
	"bytes"
	"encoding/base32"
	godoc "go/doc"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/Unknwon/gowalker/doc"
	"github.com/Unknwon/gowalker/models"
	"github.com/Unknwon/gowalker/utils"
	"github.com/astaxie/beego"
)

var (
	recentViewedProNum = 20         // Maximum element number of recent viewed project list.
	recentViewedPros   []*recentPro // Recent viewed project list.
	langTypes          []*langType  // Languages are supported.

	tagList []string // Projects tag list.
)

// Recent viewed project.
type recentPro struct {
	Path, Synopsis    string
	IsGoRepo          bool
	Views, ViewedTime int64
}

// Language type.
type langType struct {
	Lang, Name string
}

func init() {
	// Initialized recent viewed project list.
	num, err := beego.AppConfig.Int("recentViewedProNum")
	if err == nil {
		recentViewedProNum = num
		beego.Trace("Loaded 'recentViewedProNum' -> value:", recentViewedProNum)
	} else {
		beego.Trace("Failed to load 'recentViewedProNum' -> Use default value:", recentViewedProNum)
	}

	recentViewedPros = make([]*recentPro, 0, recentViewedProNum)
	// Get recent viewed projects from database.
	proinfos, _ := models.GetRecentPros(recentViewedProNum)
	for _, p := range proinfos {
		// Only projects with import path length is less than 40 letters will be showed.
		if len(p.Path) < 40 {
			recentViewedPros = append(recentViewedPros,
				&recentPro{
					Path:       p.Path,
					Synopsis:   p.Synopsis,
					ViewedTime: p.ViewedTime,
					IsGoRepo:   p.ProName == "Go",
					Views:      p.Views,
				})
		}
	}

	// Initialized language type list.
	langs := strings.Split(beego.AppConfig.String("language"), "|")
	names := strings.Split(beego.AppConfig.String("langNames"), "|")
	langTypes = make([]*langType, 0, len(langs))
	for i, v := range langs {
		langTypes = append(langTypes, &langType{
			Lang: v,
			Name: names[i],
		})
	}

	// Initialize project tags.
	tagList = strings.Split(beego.AppConfig.String("tags"), "|")
}

func setLangVer(req *http.Request, input url.Values) (*langType, []*langType) {
	// 1. Check URL arguments.
	lang := input.Get("lang")

	// 2. Get language information from cookies.
	if len(lang) == 0 {
		ck, err := req.Cookie("lang")
		if err == nil {
			lang = ck.Value
		}
	}

	// 3. Get language information from 'Accept-Language'.
	if len(lang) == 0 {
		al := req.Header.Get("Accept-Language")
		if len(al) > 2 {
			al = al[:2] // Only compare first two letters.
			for _, v := range langTypes {
				if al == v.Lang {
					lang = al
					break
				}
			}
		}
	}

	// 4. Default language is English.
	if len(lang) == 0 {
		lang = "en"
	}

	curLang := &langType{
		Lang: lang,
	}

	restLangs := make([]*langType, 0, len(langTypes)-1)
	for _, v := range langTypes {
		if lang != v.Lang {
			restLangs = append(restLangs, v)
		} else {
			curLang.Name = v.Name
		}
	}
	return curLang, restLangs
}

type HomeController struct {
	beego.Controller
}

// Get implemented Get method for HomeController.
// It serves home page of Go Walker.
func (this *HomeController) Get() {
	// Print unusual User-Agent.
	ua := this.Ctx.Request.Header.Get("User-Agent")
	if len(ua) < 20 {
		beego.Trace("User-Agent:", this.Ctx.Request.Header.Get("User-Agent"))
	}

	// Set language version.
	curLang, restLangs := setLangVer(this.Ctx.Request, this.Input())

	// Save language information in cookies.
	this.Ctx.SetCookie("lang", curLang.Lang+";path=/", 0)

	// Get query field.
	q := strings.TrimSpace(this.Input().Get("q"))

	// Remove last "/".
	q = strings.TrimRight(q, "/")

	if path, ok := utils.IsBrowseURL(q); ok {
		q = path
	}

	// Get pure URL.
	reqUrl := this.Ctx.Request.RequestURI[1:]
	if i := strings.Index(reqUrl, "?"); i > -1 {
		reqUrl = reqUrl[:i]
	}

	if len(reqUrl) == 0 && len(q) > 0 {
		reqUrl = q
	}

	// Set properties
	this.Layout = "layout_" + curLang.Lang + ".html"

	// Set language properties.
	this.Data["Lang"] = curLang.Lang
	this.Data["CurLang"] = curLang.Name
	this.Data["RestLangs"] = restLangs

	// Check show home page or documentation page.
	if len(reqUrl) == 0 && len(q) == 0 {
		// Home page.
		this.TplNames = "home_" + curLang.Lang + ".html"

		// Recent projects
		this.Data["RecentPros"] = recentViewedPros
		// Get popular project list from database.
		pkgInfos, _ := models.GetPopularPros(0, 25)
		this.Data["PopPros"] = pkgInfos
		// Set standard library keyword type-ahead.
		this.Data["DataSrc"] = utils.GoRepoSet
	} else {
		// Documentation page.
		broPath := reqUrl // Browse path.

		// Check if it is standard library.
		if utils.IsGoRepoPath(broPath) {
			broPath = "code.google.com/p/go/source/browse/src/pkg/" + broPath
		}

		// Check if it is a remote path that can be used for 'go get', if not means it's a keyword.
		if !utils.IsValidRemotePath(broPath) {
			// Show search page
			this.Redirect("/search?q="+reqUrl, 302)
			return
		}

		// Check documentation of this import path, and update automatically as needed.
		pdoc, err := doc.CheckDoc(reqUrl, doc.HUMAN_REQUEST)
		if err == nil {
			// Generate documentation page.

			/* TODO */

			if pdoc != nil && generatePage(this, pdoc, broPath, curLang.Lang) {
				// Update recent project list.
				updateRecentPros(pdoc)
				// Update project views.
				pinfo := &models.PkgInfo{
					Path:        pdoc.ImportPath,
					Synopsis:    pdoc.Synopsis,
					Created:     pdoc.Created,
					ProName:     pdoc.ProjectName,
					ViewedTime:  pdoc.ViewedTime,
					Views:       pdoc.Views,
					Etag:        pdoc.Etag,
					Tags:        pdoc.Tags,
					ImportedNum: pdoc.ImportedNum,
					ImportPid:   pdoc.ImportPid,
				}
				models.AddViews(pinfo)
				return
			}
		} else {
			beego.Error("HomeController.Get():", err)
		}

		// Show search page
		this.Redirect("/search?q="+reqUrl, 302)
		return
	}
}

// generatePage genarates documentation page for project.
// it returns false when its a invaild(empty) project.
func generatePage(this *HomeController, pdoc *doc.Package, q string, lang string) bool {
	// Load project data from database.
	pdecl, err := models.LoadProject(pdoc.ImportPath)
	if err != nil {
		beego.Error("HomeController.generatePage():", err)
		return false
	}

	// Set properties.
	this.TplNames = "docs_" + lang + ".html"

	// Refresh (within 10 seconds).
	this.Data["IsRefresh"] = pdoc.Created.Add(10 * time.Second).UTC().After(time.Now().UTC())

	// Get VCS name, project name, project home page, and Upper level project URL.
	this.Data["VCS"], this.Data["ProName"], this.Data["ProPath"], this.Data["ProDocPath"] = getVCSInfo(q, pdoc)

	if utils.IsGoRepoPath(pdoc.ImportPath) {
		this.Data["IsGoRepo"] = true
	}

	this.Data["Views"] = pdoc.Views + 1

	// Tags.
	this.Data["Tags"] = getTags(pdoc.Tags, lang)

	// Introduction.
	this.Data["ImportPath"] = pdoc.ImportPath
	byts, _ := base32.StdEncoding.DecodeString(pdecl.Doc)
	this.Data["PkgFullIntro"] = string(byts)

	var buf bytes.Buffer
	// Convert data format.
	err = ConvertDataFormat(pdoc, pdecl)
	if err != nil {
		beego.Error("HomeController.generatePage(): ConvertDataFormat", err)
		return false
	}

	links := make([]*utils.Link, 0, len(pdoc.Types)+len(pdoc.Imports)+len(pdoc.Funcs)+10)
	// Get all types, functions and import packages
	for _, t := range pdoc.Types {
		links = append(links, &utils.Link{
			Name:    t.Name,
			Comment: template.HTMLEscapeString(t.Doc),
		})
	}

	for _, f := range pdoc.Funcs {
		links = append(links, &utils.Link{
			Name:    f.Name,
			Comment: template.HTMLEscapeString(f.Doc),
		})
	}

	for _, t := range pdoc.Types {
		for _, f := range t.Funcs {
			links = append(links, &utils.Link{
				Name:    f.Name,
				Comment: template.HTMLEscapeString(f.Doc),
			})
		}
	}

	for _, v := range pdoc.Imports {
		links = append(links, &utils.Link{
			Name: path.Base(v) + ".",
			Path: v,
		})
	}

	// Index.
	this.Data["IsHasConst"] = len(pdoc.Consts) > 0
	this.Data["IsHasVar"] = len(pdoc.Vars) > 0
	this.Data["Funcs"] = pdoc.Funcs
	for i, f := range pdoc.Funcs {
		buf.Reset()
		godoc.ToHTML(&buf, f.Doc, nil)
		f.Doc = buf.String()
		buf.Reset()
		utils.FormatCode(&buf, &f.Decl, links)
		f.FmtDecl = buf.String()
		buf.Reset()
		utils.FormatCode(&buf, &f.Code, links)
		f.Code = buf.String()
		pdoc.Funcs[i] = f
	}
	this.Data["Types"] = pdoc.Types
	for i, t := range pdoc.Types {
		for j, f := range t.Funcs {
			buf.Reset()
			godoc.ToHTML(&buf, f.Doc, nil)
			f.Doc = buf.String()
			buf.Reset()
			utils.FormatCode(&buf, &f.Decl, links)
			f.FmtDecl = buf.String()
			buf.Reset()
			utils.FormatCode(&buf, &f.Code, links)
			f.Code = buf.String()
			t.Funcs[j] = f
		}
		for j, m := range t.Methods {
			buf.Reset()
			godoc.ToHTML(&buf, m.Doc, nil)
			m.Doc = buf.String()
			buf.Reset()
			utils.FormatCode(&buf, &m.Decl, links)
			m.FmtDecl = buf.String()
			buf.Reset()
			utils.FormatCode(&buf, &m.Code, links)
			m.Code = buf.String()
			t.Methods[j] = m
		}
		buf.Reset()
		godoc.ToHTML(&buf, t.Doc, nil)
		t.Doc = buf.String()
		buf.Reset()
		utils.FormatCode(&buf, &t.Decl, links)
		t.FmtDecl = buf.String()
		pdoc.Types[i] = t
	}

	// Constants.
	this.Data["Consts"] = pdoc.Consts
	for i, v := range pdoc.Consts {
		buf.Reset()
		utils.FormatCode(&buf, &v.Decl, links)
		v.FmtDecl = buf.String()
		pdoc.Consts[i] = v
	}

	// Variables.
	this.Data["Vars"] = pdoc.Vars
	for i, v := range pdoc.Vars {
		buf.Reset()
		utils.FormatCode(&buf, &v.Decl, links)
		v.FmtDecl = buf.String()
		pdoc.Vars[i] = v
	}

	// Dirs.
	this.Data["IsHasSubdirs"] = len(pdoc.Dirs) > 0
	pinfos := make([]*models.PkgInfo, 0, len(pdoc.Dirs))
	for _, v := range pdoc.Dirs {
		v = pdoc.ImportPath + "/" + v
		if pinfo, err := models.GetPkgInfo(v); err == nil {
			pinfos = append(pinfos, pinfo)
		} else {
			pinfos = append(pinfos, &models.PkgInfo{Path: v})
		}
	}
	this.Data["Subdirs"] = pinfos

	this.Data["Files"] = pdoc.Files
	this.Data["ImportPkgs"] = pdecl.Imports
	this.Data["ImportPkgNum"] = len(pdoc.Imports) - 1
	this.Data["IsImported"] = pdoc.ImportedNum > 0
	this.Data["ImportPid"] = pdoc.ImportPid
	this.Data["ImportedNum"] = pdoc.ImportedNum
	this.Data["UtcTime"] = pdoc.Created
	this.Data["GOOS"] = pdecl.Goos
	this.Data["GOARCH"] = pdecl.Goarch
	return true
}

// getVCSInfo returns VCS name, project name, project home page, and Upper level project URL.
func getVCSInfo(q string, pdoc *doc.Package) (vcs, proName, proPath, pkgDocPath string) {
	// Get project name.
	lastIndex := strings.LastIndex(q, "/")
	proName = q[lastIndex+1:]
	if i := strings.Index(proName, "?"); i > -1 {
		proName = proName[:i]
	}

	// Project VCS home page.
	switch {
	case q[0] == 'c': // code.google.com
		vcs = "Google Code"
		if strings.Index(q, "source/") == -1 {
			proPath = strings.Replace(q, "/"+pdoc.ProjectName, "/"+pdoc.ProjectName+"/source/browse", 1)
		} else {
			proPath = q
		}
	case q[0] == 'g': // github.com
		vcs = "Github"
		if proName != pdoc.ProjectName {
			// Not root.
			proPath = strings.Replace(q, "/"+proName, "/tree/master/"+proName, 1)
		} else {
			proPath = q + "/tree/master"
		}
	case q[0] == 'b': // bitbucket.org
		vcs = "BitBucket"
		if proName != pdoc.ProjectName {
			// Not root.
			proPath = strings.Replace(q, "/"+pdoc.ProjectName, "/"+pdoc.ProjectName+"/src/default", 1)
		} else {
			proPath = q + "/src/default"
		}
	case q[0] == 'l': // launchpad.net
		vcs = "Launchpad"
		proPath = "bazaar." + strings.Replace(q, "/"+pdoc.ProjectName, "/+branch/"+pdoc.ProjectName+"/view/head:/", 1)
	}

	pkgDocPath = q[:lastIndex]

	return vcs, proName, proPath, pkgDocPath
}

func getTags(rawTag, lang string) []string {
	// Get tag labels.
	tagLabels := strings.Split(beego.AppConfig.String("tagsLabels"), "|")

	tags := strings.Split(rawTag, "|")
	tags = tags[:len(tags)-1] // The last element is always empty.
	// Remove first character '$' in every tag.
	for i := range tags {
		tags[i] = tags[i][1:]
		// Reassign tag label name.
		for j, v := range tagList {
			if tags[i] == v {
				tags[i] = tagLabels[j]
				break
			}
		}
	}
	return tags
}

// ConvertDataFormat converts data from database acceptable format to useable format.
func ConvertDataFormat(pdoc *doc.Package, pdecl *models.PkgDecl) error {
	// Consts
	pdoc.Consts = make([]*doc.Value, 0, 5)
	for _, v := range strings.Split(pdecl.Consts, "&$#") {
		val := new(doc.Value)
		for j, s := range strings.Split(v, "&V#") {
			switch j {
			case 0: // Name
				val.Name = s
			case 1: // Doc
				val.Doc = s
			case 2: // Decl
				val.Decl = template.HTMLEscapeString(s)
			case 3: // URL
				val.URL = s
			}
		}
		pdoc.Consts = append(pdoc.Consts, val)
	}
	pdoc.Consts = pdoc.Consts[:len(pdoc.Consts)-1]

	// Variables
	pdoc.Vars = make([]*doc.Value, 0, 5)
	for _, v := range strings.Split(pdecl.Vars, "&$#") {
		val := new(doc.Value)
		for j, s := range strings.Split(v, "&V#") {
			switch j {
			case 0: // Name
				val.Name = s
			case 1: // Doc
				val.Doc = s
			case 2: // Decl
				val.Decl = s
			case 3: // URL
				val.URL = s
			}
		}
		pdoc.Vars = append(pdoc.Vars, val)
	}
	pdoc.Vars = pdoc.Vars[:len(pdoc.Vars)-1]

	// Functions
	pdoc.Funcs = make([]*doc.Func, 0, 10)
	for _, v := range strings.Split(pdecl.Funcs, "&$#") {
		val := new(doc.Func)
		for j, s := range strings.Split(v, "&F#") {
			switch j {
			case 0: // Name
				val.Name = s
			case 1: // Doc
				val.Doc = s
			case 2: // Decl
				val.Decl = s
			case 3: // URL
				val.URL = s
			case 4: // Code
				val.Code = *codeDecode(&s)
			}
		}
		pdoc.Funcs = append(pdoc.Funcs, val)
	}
	pdoc.Funcs = pdoc.Funcs[:len(pdoc.Funcs)-1]

	// Types
	pdoc.Types = make([]*doc.Type, 0, 10)
	for _, v := range strings.Split(pdecl.Types, "&##") {
		val := new(doc.Type)
		for j, s := range strings.Split(v, "&$#") {
			switch j {
			case 0: // Type
				for y, s2 := range strings.Split(s, "&T#") {
					switch y {
					case 0: // Name
						val.Name = s2
					case 1: // Doc
						val.Doc = s2
					case 2: // Decl
						val.Decl = s2
					case 3: // URL
						val.URL = s2
					}
				}
			case 1: // Functions
				val.Funcs = make([]*doc.Func, 0, 2)
				for _, v2 := range strings.Split(s, "&M#") {
					val2 := new(doc.Func)
					for y, s2 := range strings.Split(v2, "&F#") {
						switch y {
						case 0: // Name
							val2.Name = s2
						case 1: // Doc
							val2.Doc = s2
						case 2: // Decl
							val2.Decl = s2
						case 3: // URL
							val2.URL = s2
						case 4: // Code
							val2.Code = *codeDecode(&s2)
						}
					}
					val.Funcs = append(val.Funcs, val2)
				}
				val.Funcs = val.Funcs[:len(val.Funcs)-1]
			case 3: // Methods.
				val.Methods = make([]*doc.Func, 0, 5)
				for _, v2 := range strings.Split(s, "&M#") {
					val2 := new(doc.Func)
					for y, s2 := range strings.Split(v2, "&F#") {
						switch y {
						case 0: // Name
							val2.Name = s2
						case 1: // Doc
							val2.Doc = s2
						case 2: // Decl
							val2.Decl = s2
						case 3: // URL
							val2.URL = s2
						case 4: // Code
							val2.Code = *codeDecode(&s2)
						}
					}
					val.Methods = append(val.Methods, val2)
				}
				val.Methods = val.Methods[:len(val.Methods)-1]
			}
		}
		pdoc.Types = append(pdoc.Types, val)
	}
	pdoc.Types = pdoc.Types[:len(pdoc.Types)-1]

	// Dirs.
	pdoc.Dirs = strings.Split(pdecl.Dirs, "|")
	pdoc.Dirs = pdoc.Dirs[:len(pdoc.Dirs)-1]

	// Imports.
	pdoc.Imports = strings.Split(pdecl.Imports, "|")

	// Files.
	pdoc.Files = strings.Split(pdecl.Files, "|")
	return nil
}

func codeDecode(code *string) *string {
	str := new(string)
	byts, _ := base32.StdEncoding.DecodeString(*code)
	*str = string(byts)
	return str
}

func updateRecentPros(pdoc *doc.Package) {
	// Only projects with import path length is less than 40 letters will be showed.
	if len(pdoc.ImportPath) < 40 {
		index := -1
		listLen := len(recentViewedPros)
		curPro := &recentPro{
			Path:       pdoc.ImportPath,
			Synopsis:   pdoc.Synopsis,
			ViewedTime: time.Now().UTC().Unix(),
			IsGoRepo:   pdoc.ProjectName == "Go",
			Views:      pdoc.Views,
		}

		pdoc.ViewedTime = curPro.ViewedTime

		// Check if in the list
		for i, s := range recentViewedPros {
			if s.Path == curPro.Path {
				index = i
				break
			}
		}

		s := make([]*recentPro, 0, recentViewedProNum)
		s = append(s, curPro)
		switch {
		case index == -1 && listLen < recentViewedProNum:
			// Not found and list is not full
			s = append(s, recentViewedPros...)
		case index == -1 && listLen >= recentViewedProNum:
			// Not found but list is full
			s = append(s, recentViewedPros[:recentViewedProNum-1]...)
		case index > -1:
			// Found
			s = append(s, recentViewedPros[:index]...)
			s = append(s, recentViewedPros[index+1:]...)
		}
		recentViewedPros = s
	}
}
