// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/cortexproject/cortex/blob/master/tools/doc-generator/parser.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Cortex Authors.

package parse

import (
	"flag"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/flagext"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/weaveworks/common/logging"

	"github.com/grafana/mimir/pkg/ingester"
	"github.com/grafana/mimir/pkg/util/fieldcategory"
)

var (
	yamlFieldNameParser   = regexp.MustCompile("^[^,]+")
	yamlFieldInlineParser = regexp.MustCompile("^[^,]*,inline$")
)

// ExamplerConfig can be implemented by configs to provide examples.
// If string is non-empty, it will be added as comment.
// If yaml value is non-empty, it will be marshaled as yaml under the same key as it would appear in config.
type ExamplerConfig interface {
	ExampleDoc() (comment string, yaml interface{})
}

type FieldExample struct {
	Comment string
	Yaml    interface{}
}

type ConfigBlock struct {
	Name          string
	Desc          string
	Entries       []*ConfigEntry
	FlagsPrefix   string
	FlagsPrefixes []string
}

func (b *ConfigBlock) Add(entry *ConfigEntry) {
	b.Entries = append(b.Entries, entry)
}

type EntryKind string

const (
	KindBlock EntryKind = "block"
	KindField EntryKind = "field"
)

type ConfigEntry struct {
	Kind     EntryKind
	Name     string
	Required bool

	// In case the Kind is KindBlock
	Block     *ConfigBlock
	BlockDesc string
	Root      bool

	// In case the Kind is KindField
	FieldFlag     string
	FieldDesc     string
	FieldType     string
	FieldDefault  string
	FieldExample  *FieldExample
	FieldCategory string
}

func (e ConfigEntry) Description() string {
	if e.FieldCategory == "" || e.FieldCategory == "basic" {
		return e.FieldDesc
	}

	return fmt.Sprintf("(%s) %s", e.FieldCategory, e.FieldDesc)
}

type RootBlock struct {
	Name       string
	Desc       string
	StructType reflect.Type
}

func Flags(cfg flagext.RegistererWithLogger, logger log.Logger) map[uintptr]*flag.Flag {
	fs := flag.NewFlagSet("", flag.PanicOnError)
	cfg.RegisterFlags(fs, logger)

	flags := map[uintptr]*flag.Flag{}
	fs.VisitAll(func(f *flag.Flag) {
		// Skip deprecated flags
		if f.Value.String() == "deprecated" {
			return
		}

		ptr := reflect.ValueOf(f.Value).Pointer()
		flags[ptr] = f
	})

	return flags
}

// Config returns a slice of ConfigBlocks. The first ConfigBlock is a resursively expanded cfg.
// The remaining entries in the slice are all (root or not) ConfigBlocks.
func Config(block *ConfigBlock, cfg interface{}, flags map[uintptr]*flag.Flag) ([]*ConfigBlock, error) {
	blocks := []*ConfigBlock{}

	// If the input block is nil it means we're generating the doc for the top-level block
	if block == nil {
		block = &ConfigBlock{}
		blocks = append(blocks, block)
	}

	// The input config is expected to be addressable.
	if reflect.TypeOf(cfg).Kind() != reflect.Ptr {
		t := reflect.TypeOf(cfg)
		return nil, fmt.Errorf("%s is a %s while a %s is expected", t, t.Kind(), reflect.Ptr)
	}

	// The input config is expected to be a pointer to struct.
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%s is a %s while a %s is expected", v, v.Kind(), reflect.Struct)
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.FieldByIndex(field.Index)

		// Skip fields explicitly marked as "hidden" in the doc
		if isFieldHidden(field) {
			continue
		}

		// Skip fields not exported via yaml (unless they're inline)
		fieldName := getFieldName(field)
		if fieldName == "" && !isFieldInline(field) {
			continue
		}

		// Skip field types which are non configurable
		if field.Type.Kind() == reflect.Func {
			continue
		}

		// Skip deprecated fields we're still keeping for backward compatibility
		// reasons (by convention we prefix them by UnusedFlag)
		if strings.HasPrefix(field.Name, "UnusedFlag") {
			continue
		}

		// Handle custom fields in vendored libs upon which we have no control.
		fieldEntry, err := getCustomFieldEntry(field, fieldValue, flags)
		if err != nil {
			return nil, err
		}
		if fieldEntry != nil {
			block.Add(fieldEntry)
			continue
		}

		// Recursively re-iterate if it's a struct
		if field.Type.Kind() == reflect.Struct {
			// Check whether the sub-block is a root config block
			rootName, rootDesc, isRoot := isRootBlock(field.Type)

			// Since we're going to recursively iterate, we need to create a new sub
			// block and pass it to the doc generation function.
			var subBlock *ConfigBlock

			if !isFieldInline(field) {
				var blockName string
				var blockDesc string

				if isRoot {
					blockName = rootName
					blockDesc = rootDesc
				} else {
					blockName = fieldName
					blockDesc = getFieldDescription(field, "")
				}

				subBlock = &ConfigBlock{
					Name: blockName,
					Desc: blockDesc,
				}

				block.Add(&ConfigEntry{
					Kind:      KindBlock,
					Name:      fieldName,
					Required:  isFieldRequired(field),
					Block:     subBlock,
					BlockDesc: blockDesc,
					Root:      isRoot,
				})

				if isRoot {
					blocks = append(blocks, subBlock)
				}
			} else {
				subBlock = block
			}

			// Recursively generate the doc for the sub-block
			otherBlocks, err := Config(subBlock, fieldValue.Addr().Interface(), flags)
			if err != nil {
				return nil, err
			}

			blocks = append(blocks, otherBlocks...)
			continue
		}

		fieldType, err := getFieldType(field.Type)
		if err != nil {
			return nil, errors.Wrapf(err, "config=%s.%s", t.PkgPath(), t.Name())
		}

		fieldFlag, err := getFieldFlag(field, fieldValue, flags)
		if err != nil {
			return nil, errors.Wrapf(err, "config=%s.%s", t.PkgPath(), t.Name())
		}
		if fieldFlag == nil {
			block.Add(&ConfigEntry{
				Kind:          KindField,
				Name:          fieldName,
				Required:      isFieldRequired(field),
				FieldDesc:     getFieldDescription(field, ""),
				FieldType:     fieldType,
				FieldExample:  getFieldExample(fieldName, field.Type),
				FieldCategory: getFieldCategory(field, ""),
			})
			continue
		}

		block.Add(&ConfigEntry{
			Kind:          KindField,
			Name:          fieldName,
			Required:      isFieldRequired(field),
			FieldFlag:     fieldFlag.Name,
			FieldDesc:     getFieldDescription(field, fieldFlag.Usage),
			FieldType:     fieldType,
			FieldDefault:  getFieldDefault(field, fieldFlag.DefValue),
			FieldExample:  getFieldExample(fieldName, field.Type),
			FieldCategory: getFieldCategory(field, fieldFlag.Name),
		})
	}

	return blocks, nil
}

func getFieldName(field reflect.StructField) string {
	name := field.Name
	tag := field.Tag.Get("yaml")

	// If the tag is not specified, then an exported field can be
	// configured via the field name (lowercase), while an unexported
	// field can't be configured.
	if tag == "" {
		if unicode.IsLower(rune(name[0])) {
			return ""
		}

		return strings.ToLower(name)
	}

	// Parse the field name
	fieldName := yamlFieldNameParser.FindString(tag)
	if fieldName == "-" {
		return ""
	}

	return fieldName
}

func getFieldType(t reflect.Type) (string, error) {
	// Handle custom data types used in the config
	switch t.String() {
	case reflect.TypeOf(&url.URL{}).String():
		return "url", nil
	case reflect.TypeOf(time.Duration(0)).String():
		return "duration", nil
	case reflect.TypeOf(flagext.StringSliceCSV{}).String():
		return "string", nil
	case reflect.TypeOf(flagext.CIDRSliceCSV{}).String():
		return "string", nil
	case reflect.TypeOf([]*relabel.Config{}).String():
		return "relabel_config...", nil
	case reflect.TypeOf(ingester.ActiveSeriesCustomTrackersConfig{}).String():
		return "map of tracker name (string) to matcher (string)", nil
	}

	// Fallback to auto-detection of built-in data types
	switch t.Kind() {
	case reflect.Bool:
		return "boolean", nil

	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		fallthrough
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		return "int", nil

	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		return "float", nil

	case reflect.String:
		return "string", nil

	case reflect.Slice:
		// Get the type of elements
		elemType, err := getFieldType(t.Elem())
		if err != nil {
			return "", err
		}

		return "list of " + elemType, nil

	case reflect.Map:
		return fmt.Sprintf("map of %s to %s", t.Key(), t.Elem().String()), nil

	default:
		return "", fmt.Errorf("unsupported data type %s", t.Kind())
	}
}

func ReflectType(typ string) reflect.Type {
	switch typ {
	case "string":
		return reflect.TypeOf("")
	case "url":
		return reflect.TypeOf(flagext.URLValue{})
	case "duration":
		return reflect.TypeOf(time.Duration(0))
	case "time":
		return reflect.TypeOf(flagext.Time{})
	case "boolean":
		return reflect.TypeOf(false)
	case "int":
		return reflect.TypeOf(0)
	case "float":
		return reflect.TypeOf(0.0)
	case "list of string":
		return reflect.TypeOf([]string{})
	case "map of string to string":
		fallthrough
	case "map of tracker name (string) to matcher (string)":
		return reflect.TypeOf(map[string]string{})
	case "relabel_config...":
		return reflect.TypeOf([]*relabel.Config{})
	case "map of string to float64":
		return reflect.TypeOf(map[string]float64{})
	case "list of duration":
		return reflect.TypeOf([]time.Duration{})
	default:
		panic("unknown field type " + typ)
	}
}

func getFieldFlag(field reflect.StructField, fieldValue reflect.Value, flags map[uintptr]*flag.Flag) (*flag.Flag, error) {
	if isAbsentInCLI(field) {
		return nil, nil
	}
	fieldPtr := fieldValue.Addr().Pointer()
	fieldFlag, ok := flags[fieldPtr]
	if !ok {
		return nil, nil
	}

	return fieldFlag, nil
}

func getFieldExample(fieldKey string, fieldType reflect.Type) *FieldExample {
	ex, ok := reflect.New(fieldType).Interface().(ExamplerConfig)
	if !ok {
		return nil
	}
	comment, yml := ex.ExampleDoc()
	return &FieldExample{
		Comment: comment,
		Yaml:    map[string]interface{}{fieldKey: yml},
	}
}

func getCustomFieldEntry(field reflect.StructField, fieldValue reflect.Value, flags map[uintptr]*flag.Flag) (*ConfigEntry, error) {
	if field.Type == reflect.TypeOf(logging.Level{}) || field.Type == reflect.TypeOf(logging.Format{}) {
		fieldFlag, err := getFieldFlag(field, fieldValue, flags)
		if err != nil {
			return nil, err
		}

		return &ConfigEntry{
			Kind:          KindField,
			Name:          getFieldName(field),
			Required:      isFieldRequired(field),
			FieldFlag:     fieldFlag.Name,
			FieldDesc:     fieldFlag.Usage,
			FieldType:     "string",
			FieldDefault:  getFieldDefault(field, fieldFlag.DefValue),
			FieldCategory: getFieldCategory(field, fieldFlag.Name),
		}, nil
	}
	if field.Type == reflect.TypeOf(flagext.URLValue{}) {
		fieldFlag, err := getFieldFlag(field, fieldValue, flags)
		if err != nil {
			return nil, err
		}

		return &ConfigEntry{
			Kind:          KindField,
			Name:          getFieldName(field),
			Required:      isFieldRequired(field),
			FieldFlag:     fieldFlag.Name,
			FieldDesc:     fieldFlag.Usage,
			FieldType:     "url",
			FieldDefault:  getFieldDefault(field, fieldFlag.DefValue),
			FieldCategory: getFieldCategory(field, fieldFlag.Name),
		}, nil
	}
	if field.Type == reflect.TypeOf(flagext.Secret{}) {
		fieldFlag, err := getFieldFlag(field, fieldValue, flags)
		if err != nil {
			return nil, err
		}

		return &ConfigEntry{
			Kind:          KindField,
			Name:          getFieldName(field),
			Required:      isFieldRequired(field),
			FieldFlag:     fieldFlag.Name,
			FieldDesc:     fieldFlag.Usage,
			FieldType:     "string",
			FieldDefault:  getFieldDefault(field, fieldFlag.DefValue),
			FieldCategory: getFieldCategory(field, fieldFlag.Name),
		}, nil
	}
	if field.Type == reflect.TypeOf(model.Duration(0)) {
		fieldFlag, err := getFieldFlag(field, fieldValue, flags)
		if err != nil {
			return nil, err
		}

		return &ConfigEntry{
			Kind:          KindField,
			Name:          getFieldName(field),
			Required:      isFieldRequired(field),
			FieldFlag:     fieldFlag.Name,
			FieldDesc:     fieldFlag.Usage,
			FieldType:     "duration",
			FieldDefault:  getFieldDefault(field, fieldFlag.DefValue),
			FieldCategory: getFieldCategory(field, fieldFlag.Name),
		}, nil
	}
	if field.Type == reflect.TypeOf(flagext.Time{}) {
		fieldFlag, err := getFieldFlag(field, fieldValue, flags)
		if err != nil {
			return nil, err
		}

		return &ConfigEntry{
			Kind:          KindField,
			Name:          getFieldName(field),
			Required:      isFieldRequired(field),
			FieldFlag:     fieldFlag.Name,
			FieldDesc:     fieldFlag.Usage,
			FieldType:     "time",
			FieldDefault:  getFieldDefault(field, fieldFlag.DefValue),
			FieldCategory: getFieldCategory(field, fieldFlag.Name),
		}, nil
	}

	return nil, nil
}

func getFieldCategory(field reflect.StructField, name string) string {
	if category, ok := fieldcategory.GetOverride(name); ok {
		return category.String()
	}
	return field.Tag.Get("category")
}

func getFieldDefault(field reflect.StructField, fallback string) string {
	if v := getDocTagValue(field, "default"); v != "" {
		return v
	}

	return fallback
}

func isFieldHidden(f reflect.StructField) bool {
	return getDocTagFlag(f, "hidden")
}

func isAbsentInCLI(f reflect.StructField) bool {
	return getDocTagFlag(f, "nocli")
}

func isFieldRequired(f reflect.StructField) bool {
	return getDocTagFlag(f, "required")
}

func isFieldInline(f reflect.StructField) bool {
	return yamlFieldInlineParser.MatchString(f.Tag.Get("yaml"))
}

func getFieldDescription(f reflect.StructField, fallback string) string {
	if desc := getDocTagValue(f, "description"); desc != "" {
		return desc
	}

	return fallback
}

func isRootBlock(t reflect.Type) (string, string, bool) {
	for _, rootBlock := range RootBlocks {
		if t == rootBlock.StructType {
			return rootBlock.Name, rootBlock.Desc, true
		}
	}

	return "", "", false
}

func getDocTagFlag(f reflect.StructField, name string) bool {
	cfg := parseDocTag(f)
	_, ok := cfg[name]
	return ok
}

func getDocTagValue(f reflect.StructField, name string) string {
	cfg := parseDocTag(f)
	return cfg[name]
}

func parseDocTag(f reflect.StructField) map[string]string {
	cfg := map[string]string{}
	tag := f.Tag.Get("doc")

	if tag == "" {
		return cfg
	}

	for _, entry := range strings.Split(tag, "|") {
		parts := strings.SplitN(entry, "=", 2)

		switch len(parts) {
		case 1:
			cfg[parts[0]] = ""
		case 2:
			cfg[parts[0]] = parts[1]
		}
	}

	return cfg
}