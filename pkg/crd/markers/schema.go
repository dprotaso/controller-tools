/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package markers

import (
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-tools/pkg/markers"
)

const (
	DropPropertiesName = "kubebuilder:validation:DropProperties"
)

var SchemaOperationMarkers = []*definitionWithHelp{
	must(markers.MakeDefinition(DropPropertiesName, markers.DescribesType, DropProperties{})).
		WithHelp(DropProperties{}.Help()),
	must(markers.MakeDefinition(DropPropertiesName, markers.DescribesField, DropProperties{})).
		WithHelp(DropProperties{}.Help()),
}

// +controllertools:marker:generateHelp:category="CRD validation"
// DropProperties marks that the resulting schema should omit child properties defintions.
//
// Typically this should be paired with PreserveUnknownFields
type DropProperties struct{}

func (d DropProperties) ApplyToSchema(schema *apiext.JSONSchemaProps) error {
	schema.Properties = nil
	schema.Items = nil
	schema.AdditionalProperties = nil
	schema.AdditionalItems = nil
	return nil
}

func init() {
	AllDefinitions = append(AllDefinitions, SchemaOperationMarkers...)
}
