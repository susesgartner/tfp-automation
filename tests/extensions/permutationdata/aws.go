package permutationdata

import (
	"github.com/rancher/shepherd/extensions/configoperations"
	"github.com/rancher/shepherd/extensions/configoperations/permutations"
	"github.com/rancher/tfp-automation/defaults/modules"
)

const (
	amiKey       = "ami"
	awsConfigKey = "awsConfig"
)

func createAMIPermutation(config map[string]any) (permutations.Permutation, error) {
	amiKeyPath := []string{TerraformKey, awsConfigKey, amiKey}
	amiKeyValue, err := configoperations.GetValue(amiKeyPath, config)
	amiPermutation := permutations.CreatePermutation(amiKeyPath, amiKeyValue.([]any), nil)

	return amiPermutation, err
}

func CreateAMIRelationships(config map[string]any) ([]permutations.Relationship, error) {
	amiPermutation, err := createAMIPermutation(config)
	amiRelationship := permutations.CreateRelationship(modules.EC2, nil, nil, []permutations.Permutation{amiPermutation})

	return []permutations.Relationship{amiRelationship}, err
}
