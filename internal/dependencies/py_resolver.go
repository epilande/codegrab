package dependencies

import (

)

type PyResolver struct{}

func (r* PyResolver) Resolve(fileContent []byte, filePath string, projectRoot string, projectModuleName string) ([]string, error) {
	return []string{}, nil
}
