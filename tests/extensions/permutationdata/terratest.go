package permutationdata

import (
	"github.com/rancher/shepherd/extensions/configoperations"
	"github.com/rancher/shepherd/extensions/configoperations/permutations"
)

const (
	TerratestKey  = "terratest"
	k8sVersionKey = "kubernetesVersion"
)

func CreateK8sPermutation(config map[string]any) (permutations.Permutation, error) {
	k8sKeyPath := []string{TerratestKey, k8sVersionKey}
	k8sKeyValue, err := configoperations.GetValue(k8sKeyPath, config)
	k8sPermutation := permutations.CreatePermutation(k8sKeyPath, k8sKeyValue.([]any), nil)

	return k8sPermutation, err
}
