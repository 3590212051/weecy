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

package controllers

import (
	"github.com/astaxie/beego"
)

type AboutController struct {
	beego.Controller
}

// Get implemented Get method for AboutController.
// It serves about page of Go Walker.
func (this *AboutController) Get() {
	// Check language version by different ways.
	lang := checkLangVer(this.Ctx.Request, this.Input().Get("lang"))

	// Get language version.
	curLang, restLangs := getLangVer(
		this.Ctx.Request.Header.Get("Accept-Language"), lang)

	// Save language information in cookies.
	this.Ctx.SetCookie("lang", curLang.Lang+";path=/", 0)

	// Set properties
	this.Layout = "layout_" + curLang.Lang + ".html"
	this.TplNames = "about_" + curLang.Lang + ".html"

	// Set language properties.
	this.Data["Lang"] = curLang.Lang
	this.Data["CurLang"] = curLang.Name
	this.Data["RestLangs"] = restLangs
}
