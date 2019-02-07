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

func writeDocs(ctx *android.Context, filename string) error {
	moduleTypeList, err := bootstrap.ModuleTypeDocs(ctx.Context)
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}

	// We need a module name getter/setter function because I couldn't
	// find a way to keep it in a variable defined within the template.
	currentModuleName := ""
	moduleProperties := make(map[string][]bpdoc.Property)
	tmpl, err := template.New("file").Funcs(map[string]interface{}{
		"setModule": func(moduleName string) string {
			currentModuleName = moduleName
			return ""
		},
		"getModule": func() string {
			return currentModuleName
		},
		"getModuleProperties": func() []bpdoc.Property {
			if props, ok := moduleProperties[currentModuleName]; ok {
				return props
			}
			return []bpdoc.Property{}
		},
	}).Parse(fileTemplate)
	if err != nil {
		return err
	}

	var specialAttributesIndices = map[string]int{
		"name":             0,
		"srcs":             1,
		"defautls":         2,
		"host_supported":   3,
		"device_supported": 4,
	}

	// We don't care about PropertyStruct groups. Flatten properties list and arrange it
	// by putting "important" ones first, followed by the rest in alphabetic order
	for _, module := range moduleTypeList {
		specialProperties := make([]bpdoc.Property, len(specialAttributesIndices), len(specialAttributesIndices))
		properties := make([]bpdoc.Property, 0, 30)
		specialPropertiesCount := 0
		for _, propStruct := range module.PropertyStructs {
			for _, property := range propStruct.Properties {
				if index, ok := specialAttributesIndices[property.Name]; ok {
					specialProperties[index] = property
					specialPropertiesCount++
				} else {
					properties = append(properties, property)
				}
			}
		}
		sort.Slice(properties, func(i, j int) bool {
			return properties[i].Name < properties[j].Name
		})
		sortedProperties := make([]bpdoc.Property, specialPropertiesCount+len(properties))
		i := 0
		for _, prop := range specialProperties {
			if prop.Name != "" {
				sortedProperties[i] = prop
				i++
			}
		}
		copy(sortedProperties[i:], properties)
		moduleProperties[module.Name] = sortedProperties
	}

	err = tmpl.Execute(buf, moduleTypeList)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filename, buf.Bytes(), 0666)
	if err != nil {
		return err
	}

	return nil
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
}
</style>
</head>
<body>
{{- /* Fixed sidebar with module names */ -}}
<ul>
<li><h3>Modules:</h3></li>
{{range $module := .}}<li><a href="#{{$module.Name}}">{{$module.Name}}</a></li>
{{end -}}
</ul>
{{/* Main panel with H1 section per module*/}}
<div style="margin-left:30ch;padding:1px 16px;">
{{range $imodule, $module := .}}
  {{setModule $module.Name}}
  <h1 id="{{$module.Name}}">Module {{$module.Name}}</h1>
  {{if .Text }}{{.Text}}{{else}}<i>Missing synopsis</i>{{end}}
  {{- /* Comma-separated list of module attributes' links module attributes */ -}}
	<div class="breadcrumb">
    {{range $i,$prop := getModuleProperties }}
				{{ if gt $i 0 }},&nbsp;{{end -}}
				<a href=#{{getModule}}.{{$prop.Name}}>{{$prop.Name}}</a>
		{{- end -}}
  </div>

	{{- /* Property description */ -}}
	{{- template "properties" getModuleProperties -}}{{- end -}}

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
