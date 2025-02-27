package permutationsdata

import (
	"strings"

	"github.com/rancher/shepherd/pkg/config/operations"
	"github.com/rancher/shepherd/pkg/config/operations/permutations"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/rancher/tfp-automation/config"
)

const (
	moduleKey         = "module"
	cniKey            = "cni"
	awsConfigKey      = "awsConfig"
	amiKey            = "ami"
	resourcePrefixKey = "resourcePrefix"
)

func CreateModulePermutation(cattleConfig map[string]any) (*permutations.Permutation, error) {
	moduleKeyPath := []string{config.TerraformConfigurationFileKey, moduleKey}
	moduleKeyValue, err := operations.GetValue(moduleKeyPath, cattleConfig)
	if err != nil {
		return nil, err
	}

	if _, ok := moduleKeyValue.([]any); !ok {
		moduleKeyValue = []any{moduleKeyValue}
	}
	modulePermutation := permutations.CreatePermutation(moduleKeyPath, moduleKeyValue.([]any), nil)

	return &modulePermutation, nil
}

func CreateCNIPermutation(cattleConfig map[string]any) (*permutations.Permutation, error) {
	cniKeyPath := []string{config.TerraformConfigurationFileKey, cniKey}
	cniKeyValue, err := operations.GetValue(cniKeyPath, cattleConfig)
	if err != nil {
		return nil, err
	}

	if _, ok := cniKeyValue.([]any); !ok {
		cniKeyValue = []any{cniKeyValue}
	}
	cniPermutation := permutations.CreatePermutation(cniKeyPath, cniKeyValue.([]any), nil)

	return &cniPermutation, nil
}

func createAMIPermutation(cattleConfig map[string]any) (*permutations.Permutation, error) {
	amiKeyPath := []string{config.TerraformConfigurationFileKey, awsConfigKey, amiKey}
	amiKeyValue, err := operations.GetValue(amiKeyPath, cattleConfig)
	amiPermutation := permutations.CreatePermutation(amiKeyPath, amiKeyValue.([]any), nil)

	return &amiPermutation, err
}

func CreateAMIRelationships(cattleConfig map[string]any) ([]permutations.Relationship, error) {
	moduleKeyPath := []string{config.TerraformConfigurationFileKey, moduleKey}
	moduleKeyValue, err := operations.GetValue(moduleKeyPath, cattleConfig)
	if err != nil {
		return nil, err
	}

	if _, ok := moduleKeyValue.([]any); !ok {
		moduleKeyValue = []any{moduleKeyValue}
	}

	amiPermutation, err := createAMIPermutation(cattleConfig)
	if err != nil {
		return nil, err
	}

	var amiRelationships []permutations.Relationship
	for _, module := range moduleKeyValue.([]any) {
		if !strings.Contains(module.(string), "ec2") {
			continue
		}

		amiRelationship := permutations.CreateRelationship(module, nil, nil, []permutations.Permutation{*amiPermutation})
		amiRelationships = append(amiRelationships, amiRelationship)
	}

	return amiRelationships, err
}

func UniquifyTerraform(cattleConfigs []map[string]any) ([]map[string]any, error) {
	resourcePrefix := []string{config.TerraformConfigurationFileKey, resourcePrefixKey}
	var uniqueCattleConfigs []map[string]any
	for _, cattleConfig := range cattleConfigs {
		cattleConfig, err := uniquifyField(resourcePrefix, cattleConfig)
		if err != nil {
			return nil, err
		}

		uniqueCattleConfigs = append(uniqueCattleConfigs, cattleConfig)
	}

	return uniqueCattleConfigs, nil
}

func uniquifyField(keyPath []string, cattleConfig map[string]any) (map[string]any, error) {
	keyPathValue, err := operations.GetValue(keyPath, cattleConfig)
	if err != nil {
		return nil, err
	}

	keyPathValue = namegen.AppendRandomString(keyPathValue.(string))

	uniqueCattleConfig, err := operations.ReplaceValue(keyPath, keyPathValue, cattleConfig)
	if err != nil {
		return nil, err
	}

	return uniqueCattleConfig, nil
}
