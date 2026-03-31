package custom

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/rancher/shepherd/clients/rancher"
	v1 "github.com/rancher/shepherd/clients/rancher/v1"
	"github.com/rancher/tfp-automation/config"
	"github.com/rancher/tfp-automation/framework"
	"github.com/rancher/tfp-automation/framework/set/resources/rancher2"
	nested "github.com/rancher/tfp-automation/tests/extensions/nestedModules"
	"github.com/rancher/tfp-automation/tests/extensions/provisioning"
	"github.com/stretchr/testify/require"
)

// CreateCustomCluster creates a custom cluster and returns terraform options with the provisioned Steve object.
func CreateCustomCluster(t *testing.T, client *rancher.Client, cattleConfig map[string]any, moduleKeyPath, dataDir string) (string, *terraform.Options, string, *v1.SteveAPIObject) {
	cattleConfig, err := provisioning.UniquifyTerraform(cattleConfig)
	require.NoError(t, err)

	rancherConfig, terraformConfig, terratestConfig, _ := config.LoadTFPConfigs(cattleConfig)

	_, keyPath := rancher2.SetKeyPath(moduleKeyPath, terratestConfig.PathToRepo, terraformConfig.Provider)
	terraformOptions := framework.Setup(t, terraformConfig, terratestConfig, keyPath)

	nestedRancherModuleDir, perTestTerraformOptions, err := nested.CreateNestedModules(terraformConfig, terratestConfig, terraformOptions, t.Name(), dataDir)
	require.NoError(t, err)

	newFile, rootBody, file := rancher2.InitializeNestedMainTFs(nestedRancherModuleDir)
	defer file.Close()

	clusters, _ := provisioning.Provision(t, client, client, rancherConfig, terraformConfig, terratestConfig, "", "", perTestTerraformOptions,
		[]map[string]any{cattleConfig}, newFile, rootBody, file, false, false, true, nil, nil, nestedRancherModuleDir)
	require.NotEmpty(t, clusters)

	return nestedRancherModuleDir, perTestTerraformOptions, keyPath, clusters[0]
}
