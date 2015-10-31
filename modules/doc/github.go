// Copyright 2015 Unknwon
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
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/Unknwon/com"

	"github.com/Unknwon/gowalker/models"
	"github.com/Unknwon/gowalker/modules/base"
	"github.com/Unknwon/gowalker/modules/setting"
)

var (
	githubRawHeader       = http.Header{"Accept": {"application/vnd.github-blob.raw"}}
	githubRevisionPattern = regexp.MustCompile(`data-clipboard-text="[a-z0-9A-Z]+`)
	githubPattern         = regexp.MustCompile(`^github\.com/(?P<owner>[a-z0-9A-Z_.\-]+)/(?P<repo>[a-z0-9A-Z_.\-]+)(?P<dir>/[a-z0-9A-Z_.\-/]*)?$`)
)

func getGithubRevision(importPath, tag string) (string, error) {
	data, err := com.HttpGetBytes(Client, fmt.Sprintf("https://%s/commits/"+tag, importPath), nil)
	if err != nil {
		return "", fmt.Errorf("fetch revision page: %v", err)
	}

	i := bytes.Index(data, []byte(`btn-outline`))
	if i == -1 {
		return "", errors.New("find revision locater: not found")
	}
	data = data[i+1:]
	m := githubRevisionPattern.FindSubmatch(data)
	if m == nil {
		return "", fmt.Errorf("find revision: not found")
	}
	return strings.TrimPrefix(string(m[0]), `data-clipboard-text="`), nil
}

type RepoInfo struct {
	DefaultBranch string `json:"default_branch"`
}

func getGithubDoc(match map[string]string, etag string) (*Package, error) {
	match["cred"] = setting.GitHubCredentials
	if len(match["tag"]) == 0 {
		repoInfo := new(RepoInfo)
		if err := com.HttpGetJSON(Client, com.Expand("https://api.github.com/repos/{owner}/{repo}?{cred}", match), repoInfo); err != nil {
			return nil, fmt.Errorf("get repo default branch: %v", err)
		}

		match["tag"] = repoInfo.DefaultBranch
	}

	// Check revision.
	commit, err := getGithubRevision(com.Expand("github.com/{owner}/{repo}", match), match["tag"])
	if err != nil {
		return nil, fmt.Errorf("get revision: %v", err)
	}
	if commit == etag {
		return nil, ErrPackageNotModified
	}

	// Get files.
	var tree struct {
		Tree []struct {
			Url  string
			Path string
			Type string
		}
		Url string
	}

	if err := com.HttpGetJSON(Client,
		com.Expand("https://api.github.com/repos/{owner}/{repo}/git/trees/{tag}?recursive=1&{cred}", match), &tree); err != nil {
		return nil, fmt.Errorf("get tree: %v", err)
	}

	// Because Github API URLs are case-insensitive, we need to check that the
	// userRepo returned from Github matches the one that we are requesting.
	if !strings.HasPrefix(tree.Url, com.Expand("https://api.github.com/repos/{owner}/{repo}/", match)) {
		return nil, errors.New("GitHub import path has incorrect case")
	}

	// Get source file data and subdirectories.
	dirPrefix := match["dir"]
	if dirPrefix != "" {
		dirPrefix = dirPrefix[1:] + "/"
	}
	dirLevel := len(strings.Split(dirPrefix, "/"))
	dirLength := len(dirPrefix)
	dirMap := make(map[string]bool)
	files := make([]com.RawFile, 0, 10)

	for _, node := range tree.Tree {
		// Skip directories and files in wrong directories, get them later.
		if node.Type != "blob" || !strings.HasPrefix(node.Path, dirPrefix) {
			continue
		}

		// Get files and check if directories have acceptable files.
		if d, f := path.Split(node.Path); base.IsDocFile(f) {
			// Check if file is in the directory that is corresponding to import path.
			if d == dirPrefix {
				files = append(files, &Source{
					SrcName:   f,
					BrowseUrl: com.Expand("github.com/{owner}/{repo}/blob/{tag}/{0}", match, node.Path),
					RawSrcUrl: com.Expand("https://raw.github.com/{owner}/{repo}/{tag}/{0}?{1}", match, node.Path, setting.GitHubCredentials),
				})
				continue
			}

			// Otherwise, check if it's a direct sub-directory of import path.
			if len(strings.Split(d, "/"))-dirLevel == 1 {
				dirMap[d[dirLength:len(d)-1]] = true
				continue
			}
		}
	}

	dirs := base.MapToSortedStrings(dirMap)

	if len(files) == 0 && len(dirs) == 0 {
		return nil, ErrPackageNoGoFile
	} else if err := com.FetchFiles(Client, files, githubRawHeader); err != nil {
		return nil, fmt.Errorf("fetch files: %v", err)
	}

	// Start generating data.
	// IsGoSubrepo check has been placed to crawl.getDynamic.
	w := &Walker{
		LineFmt: "#L%d",
		Pdoc: &Package{
			PkgInfo: &models.PkgInfo{
				ImportPath:  match["importPath"],
				ProjectPath: com.Expand("github.com/{owner}/{repo}", match),
				ViewDirPath: com.Expand("github.com/{owner}/{repo}/tree/{tag}/{importPath}", match),
				Etag:        commit,
				Subdirs:     strings.Join(dirs, "|"),
			},
		},
	}

	srcs := make([]*Source, 0, len(files))
	srcMap := make(map[string]*Source)
	for _, f := range files {
		s, _ := f.(*Source)
		srcs = append(srcs, s)

		if !strings.HasSuffix(f.Name(), "_test.go") {
			srcMap[f.Name()] = s
		}
	}

	pdoc, err := w.Build(&WalkRes{
		WalkDepth: WD_All,
		WalkType:  WT_Memory,
		WalkMode:  WM_All,
		Srcs:      srcs,
	})
	if err != nil {
		return nil, fmt.Errorf("error walking package: %v", err)
	}

	// Get stars.
	var repoTree struct {
		Stars int64 `json:"watchers"`
	}
	if err := com.HttpGetJSON(Client,
		com.Expand("https://api.github.com/repos/{owner}/{repo}?{cred}", match), &repoTree); err != nil {
		return nil, fmt.Errorf("get repoTree: %v", err)
	}
	pdoc.Stars = repoTree.Stars

	return pdoc, nil
}
