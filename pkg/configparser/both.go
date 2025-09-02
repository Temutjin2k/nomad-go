package configparser

func LoadAndParseYaml(filepath string, v any) error {
	if err := LoadYamlFile(filepath); err != nil {
		return err
	}

	return Parse(v)
}
