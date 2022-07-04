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

type Overrides map[string]*Override

type Override struct {
	// Schema specifices a partial patch that should be applied to the target
	// type or field
	Schema apiext.JSONSchemaProps

	// AdditionalMarkers specifies the extra markers that should apply
	// to this type or field
	AdditionalMarkers markers.MarkerValues

	// FieldMask signals fields that should be included for struct types
	// An empty set implies all fields should remain
	FieldMask sets.String

	// FieldOverrides applies to fields in structs
	FieldOverrides Overrides

	// ItemOverride applies to items in arrays or map values
	ItemOverride *Override
}

type yamlOverride struct {
	Schema            apiext.JSONSchemaProps  `yaml:",inline"`
	AdditionalMarkers []string                `yaml:"additionalMarkers"`
	FieldMask         []string                `yaml:"fieldMask"`
	FieldOverrides    map[string]yamlOverride `yaml:"fieldOverrides"`
	ItemOverride      *yamlOverride           `yaml:"itemOverride"`
	TypeOverride      *yamlOverride           `yaml:"typeOverride"`
}

func loadOverrides(runtime *Runtime, configPath string) (Overrides, error) {
	yamlBytes, err := os.ReadFile(configPath)

	if err != nil {
		return nil, fmt.Errorf("failed to read type override config at path %q: %w", configPath, err)
	}
	d := yaml.NewDecoder(bytes.NewReader(yamlBytes))
	d.KnownFields(true)

	yConfig := map[string]yamlOverride{}
	if err := d.Decode(&yConfig); err != nil {
		return nil, fmt.Errorf("failed to parse type override config: %w", err)
	}

	registry := runtime.Collector.Registry

	c := make(Overrides, len(yConfig))
	for typeName, yType := range yConfig {
		t, err := convertOverride(registry, typeName, yType)
		if err != nil {
			return nil, err
		}
		c[typeName] = t
	}

	return c, nil
}

func convertOverride(registry *markers.Registry, name string, yo yamlOverride) (*Override, error) {
	t := &Override{
		Schema:            yo.Schema,
		FieldMask:         sets.NewString(yo.FieldMask...),
		FieldOverrides:    make(map[string]*Override, len(yo.FieldOverrides)),
		AdditionalMarkers: make(markers.MarkerValues, len(yo.AdditionalMarkers)),
	}

	if err := parseMarkers(registry, markers.DescribesField, yo.AdditionalMarkers, t.AdditionalMarkers); err != nil {
		return nil, fmt.Errorf("failed to parse additional marker for field %q: %w", name, err)
	}

	if err := parseMarkers(registry, markers.DescribesType, yo.AdditionalMarkers, t.AdditionalMarkers); err != nil {
		return nil, fmt.Errorf("failed to parse additional marker for type %q: %w", name, err)
	}

	for fieldName, yField := range yo.FieldOverrides {
		f, err := convertOverride(registry, name+".fieldOverrides."+fieldName, yField)
		if err != nil {
			return nil, err
		}
		t.FieldOverrides[fieldName] = f
	}

	if yo.ItemOverride != nil {
		i, err := convertOverride(registry, name+".itemOverride", *yo.ItemOverride)
		if err != nil {
			return nil, err
		}
		t.ItemOverride = i
	}

	return t, nil
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
