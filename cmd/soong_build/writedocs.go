// Copyright 2017 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"android/soong/android"
	"bytes"
	"html/template"
	"io/ioutil"
	"sort"

	"github.com/google/blueprint/bootstrap"
	"github.com/google/blueprint/bootstrap/bpdoc"
)

type moduleTypeTemplateData struct {
	Name       string
	Synopsis   string
	Properties []bpdoc.Property
}

// The properties in this map are displayed first, according to their rank
var propertyRank = map[string]int{
	"name":             0,
	"srcs":             1,
	"defautls":         2,
	"host_supported":   3,
	"device_supported": 4,
}

// For each module type, extract its documentation and convert it to the template data.
func moduleTypeDocs2Templates(ctx *android.Context) ([]moduleTypeTemplateData, error) {
	result := make([]moduleTypeTemplateData, 0)
	moduleTypeList, err := bootstrap.ModuleTypeDocs(ctx.Context)
	if err != nil {
		return nil, err
	}

	// Flatten properties list and arrange it by putting the ones with rank first, followed by
	// the rest of the properties in alphabetic order
	for _, m := range moduleTypeList {
		item := moduleTypeTemplateData{
			Name:       m.Name,
			Synopsis:   m.Text,
			Properties: make([]bpdoc.Property, 0),
		}
		for _, propStruct := range m.PropertyStructs {
			item.Properties = append(item.Properties, propStruct.Properties...)
		}
		sort.Slice(item.Properties, func(i, j int) bool {
			if rankI, ok := propertyRank[item.Properties[i].Name]; ok {
				if rankJ, ok := propertyRank[item.Properties[j].Name]; ok {
					return rankI < rankJ
				} else {
					return true
				}
			}
			if _, ok := propertyRank[item.Properties[j].Name]; ok {
				return false
			}
			return item.Properties[i].Name < item.Properties[j].Name
		})
		result = append(result, item)
	}
	return result, err
}

func writeDocs(ctx *android.Context, filename string) error {
	buf := &bytes.Buffer{}

	// We need a module name getter/setter function because I couldn't
	// find a way to keep it in a variable defined within the template.
	currentModuleName := ""
	data, err := moduleTypeDocs2Templates(ctx)
	if err != nil {
		return err
	}
	tmpl, err := template.New("file").Funcs(map[string]interface{}{
		"setModule": func(moduleName string) string {
			currentModuleName = moduleName
			return ""
		},
		"getModule": func() string {
			return currentModuleName
		},
	}).Parse(fileTemplate)
	if err == nil {

		err = tmpl.Execute(buf, data)
	}
	if err == nil {
		err = ioutil.WriteFile(filename, buf.Bytes(), 0666)
	}
	return err
}

const (
	fileTemplate = `
<html>
<head>
<title>Build Docs</title>
<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.2.1/css/bootstrap.min.css">
<style>
.accordion,.simple{margin-left:1.5em;text-indent:-1.5em;margin-top:.25em}
.collapsible{border-width:0 0 0 1;margin-left:.25em;padding-left:.25em;border-style:solid;border-color:grey;display:none;}
span.fixed{display: block; float: left; clear: left; width: 1em;}
ul {
	list-style-type: none;
  margin: 0;
  padding: 0;
  width: 30ch;
  background-color: #f1f1f1;
  position: fixed;
  height: 100%;
  overflow: auto;
}
li a {
  display: block;
  color: #000;
  padding: 8px 16px;
  text-decoration: none;
}

li a.active {
  background-color: #4CAF50;
  color: white;
}

li a:hover:not(.active) {
  background-color: #555;
  color: white;
}repo 
</style>
</head>
<body>
{{- /* Fixed sidebar with module types */ -}}
<ul>
<li><h3>Module Types:</h3></li>
{{range $moduleType := .}}<li><a href="#{{$moduleType.Name}}">{{$moduleType.Name}}</a></li>
{{end -}}
</ul>
{{/* Main panel with H1 section per module type */}}
<div style="margin-left:30ch;padding:1px 16px;">
{{range $imodule, $moduleType := .}}
  {{setModule $moduleType.Name}}
  <h1 id="{{$moduleType.Name}}">Module {{$moduleType.Name}}</h1>
  {{if $moduleType.Synopsis }}{{$moduleType.Synopsis}}{{else}}<i>Missing synopsis</i>{{end}}
  {{- /* Comma-separated list of module attributes' links module attributes */ -}}
	<div class="breadcrumb">
    {{range $i,$prop := $moduleType.Properties }}
				{{ if gt $i 0 }},&nbsp;{{end -}}
				<a href=#{{getModule}}.{{$prop.Name}}>{{$prop.Name}}</a>
		{{- end -}}
  </div>
	{{- /* Property description */ -}}
	{{- template "properties" $moduleType.Properties -}}
{{- end -}}

{{define "properties" -}}
  {{range .}}
    {{if .Properties -}}
      <div class="accordion"  id="{{getModule}}.{{.Name}}">
        <span class="fixed">&#x2295</span><b>{{.Name}}</b>
        {{- range .OtherNames -}}, {{.}}{{- end -}}
      </div>
      <div class="collapsible">
        {{- .Text}} {{range .OtherTexts}}{{.}}{{end}}
        {{template "properties" .Properties -}}
      </div>
    {{- else -}}
      <div class="simple" id="{{getModule}}.{{.Name}}">
        <span class="fixed">&nbsp;</span><b>{{.Name}} {{range .OtherNames}}, {{.}}{{end -}}</b>
        {{- if .Text -}}{{.Text}}{{- end -}}
        {{- with .OtherTexts -}}{{.}}{{- end -}}<i>{{.Type}}</i>{{- if .Default -}}<i>Default: {{.Default}}</i>{{- end -}}
      </div>
    {{- end}}
  {{- end -}}
{{- end -}}

</div>
<script>
  accordions = document.getElementsByClassName('accordion');
  for (i=0; i < accordions.length; ++i) {
    accordions[i].addEventListener("click", function() {
      var panel = this.nextElementSibling;
      var child = this.firstElementChild;
      if (panel.style.display === "block") {
          panel.style.display = "none";
          child.textContent = '\u2295';
      } else {
          panel.style.display = "block";
          child.textContent = '\u2296';
      }
    });
  }
</script>
</body>
`
)
