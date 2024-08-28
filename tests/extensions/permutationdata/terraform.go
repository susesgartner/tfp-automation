package permutationdata

import (
	"github.com/rancher/shepherd/extensions/configoperations"
	"github.com/rancher/shepherd/extensions/configoperations/permutations"
)

const (
	TerraformKey = "terraform"
	moduleKey    = "module"
	cniKey       = "networkPlugin"
)

func CreateProviderPermutation(config map[string]any) (permutations.Permutation, error) {
	providerKeyPath := []string{TerraformKey, moduleKey}
	providerKeyValue, err := configoperations.GetValue(providerKeyPath, config)
	providerPermutation := permutations.CreatePermutation(providerKeyPath, providerKeyValue.([]any), nil)

	return providerPermutation, err
}

func CreateCNIPermutation(config map[string]any) (permutations.Permutation, error) {
	cniKeyPath := []string{TerraformKey, cniKey}
	cniKeyValue, err := configoperations.GetValue(cniKeyPath, config)
	cniPermutation := permutations.CreatePermutation(cniKeyPath, cniKeyValue.([]any), nil)

	return cniPermutation, err
}
