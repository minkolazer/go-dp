package config

// EnvYaml is a yaml config format
// Remote - config for remote commands
// Local - config for local commands
// Defaults is merged to Local and Remote config
type EnvYaml struct {
	Parent   string            `yaml:"parent"`
	Hidden   bool              `yaml:"hidden"`
	Template map[string]string `yaml:"template"`
	Defaults EnvTargetYaml     `yaml:"defaults"`
	Remote   EnvTargetYaml     `yaml:"remote"`
	Local    EnvTargetYaml     `yaml:"local"`
	Targets  map[string]struct {
		Defaults EnvTargetYaml `yaml:"defaults"`
		Remote   EnvTargetYaml `yaml:"remote"`
		Local    EnvTargetYaml `yaml:"local"`
	} `yaml:"targets"`
}

// EnvTargetYaml is a EnvYaml target
type EnvTargetYaml struct {
	Hosts  sliceOrString `yaml:"hosts"`
	User   string        `yaml:"user"`
	Branch string        `yaml:"branch"`
	URL    string        `yaml:"url"`
	Path   string        `yaml:"path"`

	Log mapOrString `yaml:"log"`
	Cmd mapOrString `yaml:"cmd"`
	Cat mapOrString `yaml:"cat"`
}

// custom types for shorten yamls
type mapOrString map[string]string
type sliceOrString []string

// implements the yaml.Unmarshaler interface for types that could be string or maps
func (m *mapOrString) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// To make unmarshal fill the plain data struct rather than calling UnmarshalYAML
	// again, we have to hide it using a type indirection

	var mapValue map[string]string
	err := unmarshal(&mapValue)
	if err != nil {
		var stringValue string
		err := unmarshal(&stringValue)
		if err != nil {
			return err
		}

		*m = map[string]string{"0": stringValue}
	} else {
		*m = mapValue
	}
	return nil
}

// implements the yaml.Unmarshaler interface for types that could be string or array
func (s *sliceOrString) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	var sliceValue []string

	err = unmarshal(&sliceValue)
	if err == nil {
		*s = sliceValue
		return
	}

	var stringValue string
	if err = unmarshal(&stringValue); err != nil {
		return
	}

	*s = []string{stringValue}
	return
}
