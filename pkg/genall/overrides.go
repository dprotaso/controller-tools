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

package genall

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-tools/pkg/markers"
)

type TypeOverrides map[string]TypeOverride

type TypeOverride struct {
	Schema            apiext.JSONSchemaProps
	AdditionalMarkers markers.MarkerValues
	FieldMask         sets.String
	FieldOverrides    map[string]FieldOverride
}

type FieldOverride struct {
	Schema            apiext.JSONSchemaProps
	AdditionalMarkers markers.MarkerValues
}

type yamlTypeOverride struct {
	Schema            apiext.JSONSchemaProps       `yaml:",inline"`
	AdditionalMarkers []string                     `yaml:"additionalMarkers"`
	FieldMask         []string                     `yaml:"fieldMask"`
	FieldOverrides    map[string]yamlFieldOverride `yaml:"fieldOverrides"`
}

type yamlFieldOverride struct {
	Schema            apiext.JSONSchemaProps `yaml:",inline"`
	AdditionalMarkers []string               `yaml:"additionalMarkers"`
}

func loadOverrides(runtime *Runtime, configPath string) (map[string]TypeOverride, error) {
	yamlBytes, err := os.ReadFile(configPath)

	if err != nil {
		return nil, fmt.Errorf("failed to read type override config at path %q: %w", configPath, err)
	}
	d := yaml.NewDecoder(bytes.NewReader(yamlBytes))
	d.KnownFields(true)

	yConfig := map[string]yamlTypeOverride{}
	if err := d.Decode(&yConfig); err != nil {
		return nil, fmt.Errorf("failed to parse type override config: %w", err)
	}

	registry := runtime.Collector.Registry

	c := make(TypeOverrides, len(yConfig))
	for typeName, yType := range yConfig {
		t := TypeOverride{
			Schema:            yType.Schema,
			FieldMask:         sets.NewString(yType.FieldMask...),
			FieldOverrides:    make(map[string]FieldOverride, len(yType.FieldOverrides)),
			AdditionalMarkers: make(markers.MarkerValues, len(yType.AdditionalMarkers)),
		}

		if err := parseMarkers(registry, markers.DescribesType, yType.AdditionalMarkers, t.AdditionalMarkers); err != nil {
			return nil, fmt.Errorf("failed to parse additional marker for type %q: %w", typeName, err)
		}

		for fieldName, yField := range yType.FieldOverrides {
			f := FieldOverride{
				Schema:            yField.Schema,
				AdditionalMarkers: make(markers.MarkerValues, len(yField.AdditionalMarkers)),
			}
			if err := parseMarkers(registry, markers.DescribesField, yField.AdditionalMarkers, f.AdditionalMarkers); err != nil {
				key := fmt.Sprintf("%s.%s", typeName, fieldName)
				return nil, fmt.Errorf("failed to parse additional marker for type field %q: %w", key, err)
			}

			t.FieldOverrides[fieldName] = f
		}
		c[typeName] = t
	}
	return c, nil
}

func parseMarkers(registry *markers.Registry, targetType markers.TargetType, additionalMarkers []string, target markers.MarkerValues) error {
	for _, rawText := range additionalMarkers {
		def := registry.Lookup("+"+rawText, targetType)
		if def == nil {
			continue
		}
		val, err := def.Parse(rawText)
		if err != nil {
			return fmt.Errorf("failed to parse marker %q: %w", rawText, err)
		}
		target[def.Name] = append(target[def.Name], val)
	}
	return nil
}
