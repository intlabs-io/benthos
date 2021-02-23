package output

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/Jeffail/benthos/v3/internal/interop/plugins"
	"github.com/Jeffail/benthos/v3/lib/log"
	"github.com/Jeffail/benthos/v3/lib/metrics"
	"github.com/Jeffail/benthos/v3/lib/types"
	"github.com/Jeffail/benthos/v3/lib/util/config"
)

//------------------------------------------------------------------------------

// PluginConstructor is a func that constructs a Benthos output plugin. These
// are plugins that are specific to certain use cases, experimental, private or
// otherwise unfit for widespread general use. Any number of plugins can be
// specified when using Benthos as a framework.
//
// The configuration object will be the result of the PluginConfigConstructor
// after overlaying the user configuration.
type PluginConstructor func(
	config interface{},
	manager types.Manager,
	logger log.Modular,
	metrics metrics.Type,
) (types.Output, error)

// PluginConfigConstructor is a func that returns a pointer to a new and fully
// populated configuration struct for a plugin type.
type PluginConfigConstructor func() interface{}

// PluginConfigSanitiser is a function that takes a configuration object for a
// plugin and returns a sanitised (minimal) version of it for printing in
// examples and plugin documentation.
//
// This function is useful for when a plugins configuration struct is very large
// and complex, but can sometimes be expressed in a more concise way without
// losing the original intent.
type PluginConfigSanitiser func(conf interface{}) interface{}

type pluginSpec struct {
	constructor     outputConstructor
	confConstructor PluginConfigConstructor
	confSanitiser   PluginConfigSanitiser
	description     string
}

// pluginSpecs is a map of all output plugin type specs.
var pluginSpecs = map[string]pluginSpec{}

// GetDeprecatedPlugin returns a constructor for an old plugin if it exists.
func GetDeprecatedPlugin(name string) (ConstructorFunc, bool) {
	spec, ok := pluginSpecs[name]
	if !ok {
		return nil, false
	}
	return ConstructorFunc(spec.constructor), true
}

// RegisterPlugin registers a plugin by a unique name so that it can be
// constructed similar to regular outputs. If configuration is not needed for
// this plugin then configConstructor can be nil. A constructor for the plugin
// itself must be provided.
func RegisterPlugin(
	typeString string,
	configConstructor PluginConfigConstructor,
	constructor PluginConstructor,
) {
	spec := pluginSpecs[typeString]
	spec.constructor = fromSimpleConstructor(func(
		conf Config,
		mgr types.Manager,
		log log.Modular,
		stats metrics.Type,
	) (Type, error) {
		return constructor(conf.Plugin, mgr, log, stats)
	})
	spec.confConstructor = configConstructor
	pluginSpecs[typeString] = spec
	plugins.Add(typeString, "output")
}

// DocumentPlugin adds a description and an optional configuration sanitiser
// function to the definition of a registered plugin. This improves the
// documentation generated by PluginDescriptions.
func DocumentPlugin(
	typeString, description string,
	configSanitiser PluginConfigSanitiser,
) {
	spec := pluginSpecs[typeString]
	spec.description = description
	spec.confSanitiser = configSanitiser
	pluginSpecs[typeString] = spec
}

// PluginCount returns the number of registered plugins. This does NOT count the
// standard set of components.
func PluginCount() int {
	return len(pluginSpecs)
}

//------------------------------------------------------------------------------

var pluginHeader = "This document was generated with `benthos --list-output-plugins`." + `

This document lists any output plugins that this flavour of Benthos offers
beyond the standard set.`

// PluginDescriptions generates and returns a markdown formatted document
// listing each registered plugin and an example configuration for it.
func PluginDescriptions() string {
	// Order alphabetically
	names := []string{}
	for name := range pluginSpecs {
		names = append(names, name)
	}
	sort.Strings(names)

	buf := bytes.Buffer{}
	buf.WriteString("Output Plugins\n")
	buf.WriteString(strings.Repeat("=", 14))
	buf.WriteString("\n\n")
	buf.WriteString(pluginHeader)
	buf.WriteString("\n\n")

	buf.WriteString("### Contents\n\n")
	for i, name := range names {
		buf.WriteString(fmt.Sprintf("%v. [`%v`](#%v)\n", i+1, name, name))
	}

	if len(names) == 0 {
		buf.WriteString("There are no plugins loaded.")
	} else {
		buf.WriteString("\n")
	}

	// Append each description
	for i, name := range names {
		var confBytes []byte

		if confCtor := pluginSpecs[name].confConstructor; confCtor != nil {
			conf := NewConfig()
			conf.Type = name
			conf.Plugin = confCtor()
			if confSanit, err := SanitiseConfig(conf); err == nil {
				confBytes, _ = config.MarshalYAML(confSanit)
			}
		}

		buf.WriteString("## ")
		buf.WriteString("`" + name + "`")
		buf.WriteString("\n")
		if confBytes != nil {
			buf.WriteString("\n``` yaml\n")
			buf.Write(confBytes)
			buf.WriteString("```\n")
		}
		if desc := pluginSpecs[name].description; len(desc) > 0 {
			buf.WriteString("\n")
			buf.WriteString(desc)
			buf.WriteString("\n")
		}
		if i != (len(names) - 1) {
			buf.WriteString("\n")
		}
	}
	return buf.String()
}

//------------------------------------------------------------------------------
