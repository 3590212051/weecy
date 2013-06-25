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

// Go Walker is a web server that generates Go projects API documentation with source code on the fly.
package main

import (
	"os"
	"runtime"

	"github.com/Unknwon/gowalker/doc"
	"github.com/Unknwon/gowalker/routers"
	"github.com/astaxie/beego"
)

const (
	VERSION = "0.4.9.0625" // Application version.
)

func init() {
	// Try to have highest performance.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Set application log level.
	if beego.AppConfig.String("runmode") == "pro" {
		beego.SetLevel(beego.LevelInfo)
	}

	// ----- Initialize log file -----
	os.Mkdir("./log", os.ModePerm)
	filew := beego.NewFileWriter("log/log.log", true)
	err := filew.StartLogger()
	if err != nil {
		beego.Critical("NewFileWriter ->", err)
	}

	doc.SetGithubCredentials(beego.AppConfig.String("client_id"), beego.AppConfig.String("client_secret"))

	beego.Info("Go Walker", VERSION)
}

func main() {
	beego.AppName = "Go Walker"
	beego.Info("Go Walker", VERSION)

	// Register routers.
	beego.Router("/", &routers.HomeRouter{})
	beego.Router("/search", &routers.SearchRouter{})

	// For all unknown pages.
	beego.Router("/:all", &routers.HomeRouter{})
	beego.Run()
}
