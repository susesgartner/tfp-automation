package custom

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/rancher/shepherd/clients/rancher"
	v1 "github.com/rancher/shepherd/clients/rancher/v1"
	"github.com/rancher/shepherd/extensions/defaults/namespaces"
	"github.com/rancher/tfp-automation/config"
	"github.com/rancher/tfp-automation/defaults/stevetypes"
	"github.com/rancher/tfp-automation/framework"
	"github.com/rancher/tfp-automation/framework/set/resources/rancher2"
	nested "github.com/rancher/tfp-automation/tests/extensions/nestedModules"
	"github.com/rancher/tfp-automation/tests/extensions/provisioning"
	"github.com/stretchr/testify/require"
)

// SetupCustomCluster creates a custom cluster and returns terraform options with the provisioned Steve object.
func SetupCustomCluster(t *testing.T, client *rancher.Client, cattleConfig map[string]any, moduleKeyPath, dataDir string) (string, *terraform.Options, string, *v1.SteveAPIObject) {
	uniqueCattleConfigs, err := provisioning.UniquifyTerraform([]map[string]any{cattleConfig})
	require.NoError(t, err)

	rancherConfig, terraformConfig, terratestConfig, _ := config.LoadTFPConfigs(uniqueCattleConfigs[0])

	_, keyPath := rancher2.SetKeyPath(moduleKeyPath, terratestConfig.PathToRepo, terraformConfig.Provider)
	terraformOptions := framework.Setup(t, terraformConfig, terratestConfig, keyPath)

	nestedRancherModuleDir, perTestTerraformOptions, err := nested.CreateNestedModules(terraformConfig, terratestConfig, terraformOptions, t.Name(), dataDir)
	require.NoError(t, err)

	newFile, rootBody, file := rancher2.InitializeNestedMainTFs(nestedRancherModuleDir)
	defer file.Close()

	clusterIDs, _ := provisioning.Provision(t, client, client, rancherConfig, terraformConfig, terratestConfig, "", "", perTestTerraformOptions,
		uniqueCattleConfigs, newFile, rootBody, file, false, false, true, nil, nil, nestedRancherModuleDir)
	require.NotEmpty(t, clusterIDs)

	cluster, err := client.Steve.SteveType(stevetypes.Provisioning).ByID(namespaces.FleetDefault + "/" + terraformConfig.ResourcePrefix)
	require.NoError(t, err)

	return nestedRancherModuleDir, perTestTerraformOptions, keyPath, cluster
}
