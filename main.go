package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type vendorcomponent struct {
	Path     string `json:"path"`
	Revision string `json:"revision"`
}

type vendor struct {
	Package []vendorcomponent `json:"package"`
}

type component struct {
	name       string
	version    string
	copyRight  string
	licenseTag string
}

var golangComponents []component
var npmComponents []component

const golangLicense = `
Copyright (c) 2009 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
`

func main() {
	b, err := ioutil.ReadFile("vendor/vendor.json")
	if err != nil {
		log.Fatal(err)
	}
	var v vendor
	err = json.Unmarshal(b, &v)
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range v.Package {
		if c.Revision == "" {
			continue
		}
		p := path.Join("vendor", c.Path)
		if strings.HasPrefix(p, "golang.org") {
			golangComponents = append(golangComponents, component{c.Path, c.Revision, golangLicense, licenseBSD3})
			continue
		}
		for p != "vendor" {
			licFile := findLicenseFile(p)
			if licFile == "" {
				p = path.Dir(p)
				continue
			}
			licType := getLicenseType(licFile)
			switch licType {
			case licenseApache2, licenseMPL2, licenseEPL1:
				copyRight := getCopyright(p)
				if copyRight == "" {
					copyRight = "Copyright not specified."
				}
				golangComponents = append(golangComponents, component{c.Path, c.Revision, copyRight, licType})
			default:
				b, err = ioutil.ReadFile(licFile)
				if err != nil {
					log.Fatal(err)
				}
				golangComponents = append(golangComponents, component{c.Path, c.Revision, string(b), licType})
			}
			break
		}
	}
	if err = os.Chdir("browser"); err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command("npm", "list", "--prod", "--parseable")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)

	}
	b, err = ioutil.ReadAll(stdout)
	if err != nil {
		log.Fatal(err)
	}
	str := string(b)
	nodeModules := strings.Split(str, "\n")

	for _, p := range nodeModules[1:] {
		if p == "" {
			continue
		}
		licFile := findLicenseFile(p)
		if licFile == "" {
			readme := getReadme(p)
			if readme == "" {
				npmpackage := getNPMPackage(path.Join(p, "package.json"))
				copyRight := "Copyright not specified."
				author := npmpackage.getAuthor()
				if author != "" {
					copyRight = copyRight + " Module author: " + author
				}
				npmComponents = append(npmComponents, component{npmpackage.Name, npmpackage.Version, copyRight, npmpackage.getLicenseType()})
				continue
			}
			licType := getLicenseType(readme)
			b, err := ioutil.ReadFile(readme)
			if err != nil {
				log.Fatal(err)
			}
			str := string(b)
			i := strings.Index(str, "opyright")
			if i == -1 {
				npmpackage := getNPMPackage(path.Join(p, "package.json"))
				licType = npmpackage.getLicenseType()
				if licType == "" {
					if strings.Contains(str, "MIT") {
						licType = "MIT"
					}
				}
				copyRight := "Copyright not specified."
				author := npmpackage.getAuthor()
				if author != "" {
					copyRight = copyRight + " Module author: " + author
				}
				npmComponents = append(npmComponents, component{npmpackage.Name, npmpackage.Version, copyRight, licType})
				continue
			}
			i--
			copyRight := str[i:]
			npmpackage := getNPMPackage(path.Join(p, "package.json"))
			npmComponents = append(npmComponents, component{npmpackage.Name, npmpackage.Version, copyRight, licType})
			continue

		}
		licType := getLicenseType(licFile)
		switch licType {
		case licenseApache2, licenseMPL2, licenseEPL1, licenseCreativeCommons:
			copyRight := getCopyright(p)
			if copyRight == "" {
				npmpackage := getNPMPackage(path.Join(p, "package.json"))
				copyRight := "Copyright not specified."
				author := npmpackage.getAuthor()
				if author != "" {
					copyRight = copyRight + " Module author: " + author
				}
			}
			npmpackage := getNPMPackage(path.Join(p, "package.json"))
			npmComponents = append(npmComponents, component{npmpackage.Name, npmpackage.Version, copyRight, licType})
		default:
			b, err = ioutil.ReadFile(licFile)
			if err != nil {
				log.Fatal(err)
			}
			npmpackage := getNPMPackage(path.Join(p, "package.json"))
			npmComponents = append(npmComponents, component{npmpackage.Name, npmpackage.Version, string(b), licType})
		}
	}
	fmt.Println(`open_source_disclosures_minio 0.9.0

=============== TABLE OF CONTENTS =============================


The following is a listing of the open source components detailed in
this document. This list is provided for your convenience; please read
further if you wish to review the copyright notice(s) and the full text
of the license associated with each component.
`)
	for i, t := range licenseList {
		fmt.Printf("\nSECTION %d: %s License\n\n", i+1, t)
		for _, c := range golangComponents {
			if t != c.licenseTag {
				continue
			}
			fmt.Printf("  >>> %s %s\n", c.name, c.version)
		}
		for _, c := range npmComponents {
			if t != c.licenseTag {
				continue
			}
			fmt.Printf("  >>> npm:%s %s\n", c.name, c.version)
		}
	}
	fmt.Println("\nAPPENDIX: Standard License Files and Templates\n")
	for _, t := range licenseList {
		fmt.Printf("  >>> %s License\n", t)
	}
	for i, t := range licenseList {
		fmt.Printf("\n--------------- SECTION %d: %s License ----------\n\n", i+1, t)
		for _, c := range golangComponents {
			if t != c.licenseTag {
				continue
			}
			fmt.Printf("\n>>> Go package: %s %s\n\n", c.name, c.version)
			fmt.Printf("%s", c.copyRight)
			fmt.Printf("\nLicense Type: %s\n", c.licenseTag)
		}
		for _, c := range npmComponents {
			if t != c.licenseTag {
				continue
			}
			fmt.Printf("\n>>> NPM package: %s %s\n\n", c.name, c.version)
			fmt.Printf("%s", c.copyRight)
			fmt.Printf("\nLicense Type: %s\n", c.licenseTag)
		}
	}
	fmt.Println("\n=============== APPENDIX: License Files and Templates ==============\n\n")
	for i, t := range licenseList {
		fmt.Printf("\n--------------- APPENDIX %d: %s License (Template) -----------\n\n", i+1, t)
		fmt.Println(licenseMap[t])
	}
}

func getLicenseType(licFile string) string {
	b, err := ioutil.ReadFile(licFile)
	if err != nil {
		log.Fatal(err)
	}
	str := string(b)
	str = strings.Replace(str, "\n", "", -1)
	str = strings.Replace(str, "\r", "", -1)
	str = strings.Replace(str, " ", "", -1)
	switch {
	case strings.Contains(str, strings.Replace(`Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions`, " ", "", -1)):
		return licenseMIT
	case strings.Contains(str, "http://www.apache.org/licenses/LICENSE-2.0"):
		return licenseApache2
	case strings.Contains(str, "ApacheLicense") && strings.Contains(str, "2.0"):
		return licenseApache2
	case strings.Contains(str, "MozillaPublicLicense") && strings.Contains(str, "2.0"):
		return licenseMPL2
	case strings.Contains(str, "EclipsePublicLicense"):
		return licenseEPL1
	case strings.Contains(str, "CreativeCommonsPublicLicenses"):
		return licenseCreativeCommons
	case strings.Contains(str, strings.Replace(`the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission`, " ", "", -1)):
		return licenseBSD3
	case strings.Contains(str, strings.Replace(`Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution`, " ", "", -1)):
		return licenseBSD2
	case strings.Contains(str, strings.Replace(`distribute this software for any purpose with or without fee is hereby granted, provided that the above copyright notice and this permission notice appear in all copies`, " ", "", -1)):
		return licenseISC
	case strings.Contains(str, "ISC"):
		return licenseISC
	case strings.Contains(str, "MIT"):
		return licenseMIT
	default:
		return "Unknown"
	}
}

func findLicenseFile(p string) string {
	if _, err := os.Stat(path.Join(p, "LICENSE.MIT")); err == nil {
		return path.Join(p, "LICENSE.MIT")
	}
	fis, err := ioutil.ReadDir(p)
	if err != nil {
		log.Fatal(err)
	}
	for _, fi := range fis {
		if strings.Contains(fi.Name(), "LICENSE") ||
			strings.Contains(fi.Name(), "LICENCE") ||
			strings.Contains(fi.Name(), "license") {
			return path.Join(p, fi.Name())
		}
	}
	return ""
}

func getCopyright(p string) (copyRight string) {
	noticeFile := path.Join(p, "NOTICE")
	b, err := ioutil.ReadFile(noticeFile)
	if err == nil {
		return string(b)
	}
	distFile := path.Join(p, "DISTRIBUTION")
	b, err = ioutil.ReadFile(distFile)
	if err == nil {
		return string(b)
	}
	filepath.Walk(p, func(file string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(file, ".go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if len(f.Comments) > 0 {
			copyRight = f.Comments[0].Text()
		}
		if !strings.Contains(copyRight, "icense") {
			copyRight = ""
		}
		return errors.New("done")
	})
	return copyRight
}

func getReadme(p string) (file string) {
	for _, readme := range []string{"README.md", "README.MD", "README", "Readme.md", "readme.md"} {
		file = path.Join(p, readme)
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}
	return ""
}
