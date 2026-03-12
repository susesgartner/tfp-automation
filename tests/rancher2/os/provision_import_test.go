//go:build os

package os

import (
	"os"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/defaults/namespaces"
	shepherdConfig "github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/config/operations"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tests/actions/qase"
	"github.com/rancher/tests/actions/workloads"
	"github.com/rancher/tests/actions/workloads/deployment"
	"github.com/rancher/tests/actions/workloads/pods"
	"github.com/rancher/tests/validation/provisioning/resources/standarduser"
	"github.com/rancher/tfp-automation/config"
	"github.com/rancher/tfp-automation/defaults/keypath"
	"github.com/rancher/tfp-automation/defaults/modules"
	"github.com/rancher/tfp-automation/defaults/stevetypes"
	"github.com/rancher/tfp-automation/framework"
	"github.com/rancher/tfp-automation/framework/cleanup"
	"github.com/rancher/tfp-automation/framework/set/provisioning/imported"
	"github.com/rancher/tfp-automation/framework/set/resources/rancher2"
	tfpQase "github.com/rancher/tfp-automation/pipeline/qase"
	"github.com/rancher/tfp-automation/pipeline/qase/results"
	"github.com/rancher/tfp-automation/tests/extensions/provisioning"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type UpgradeImportedClusterTestSuite struct {
	suite.Suite
	client             *rancher.Client
	standardUserClient *rancher.Client
	session            *session.Session
	cattleConfig       map[string]any
	rancherConfig      *rancher.Config
	terraformConfig    *config.TerraformConfig
	terratestConfig    *config.TerratestConfig
	terraformOptions   *terraform.Options
}

func (p *UpgradeImportedClusterTestSuite) SetupSuite() {
	testSession := session.NewSession()
	p.session = testSession

	client, err := rancher.NewClient("", testSession)
	require.NoError(p.T(), err)

	p.client = client

	p.cattleConfig = shepherdConfig.LoadConfigFromFile(os.Getenv(shepherdConfig.ConfigEnvironmentKey))
	p.rancherConfig, p.terraformConfig, p.terratestConfig, _ = config.LoadTFPConfigs(p.cattleConfig)

	_, keyPath := rancher2.SetKeyPath(keypath.RancherKeyPath, p.terratestConfig.PathToRepo, "")
	terraformOptions := framework.Setup(p.T(), p.terraformConfig, p.terratestConfig, keyPath)
	p.terraformOptions = terraformOptions
}

func (p *UpgradeImportedClusterTestSuite) TestTfpUpgradeImportedCluster() {
	var err error
	var testUser, testPassword string
	var clusterIDs []string

	p.standardUserClient, testUser, testPassword, err = standarduser.CreateStandardUser(p.client)
	require.NoError(p.T(), err)

	tests := []struct {
		name   string
		module string
	}{
		{"OS_RKE2_Imported", modules.ImportEC2RKE2},
		//{"OS_K3S_Imported", modules.ImportEC2K3s},
	}

	for _, tt := range tests {
		newFile, rootBody, file := rancher2.InitializeMainTF(p.terratestConfig)
		defer file.Close()

		configMap, err := provisioning.UniquifyTerraform([]map[string]any{p.cattleConfig})
		require.NoError(p.T(), err)

		_, err = operations.ReplaceValue([]string{"terraform", "module"}, tt.module, configMap[0])
		require.NoError(p.T(), err)

		rancher, terraform, terratest, _ := config.LoadTFPConfigs(configMap[0])

		p.Run((tt.name), func() {
			_, keyPath := rancher2.SetKeyPath(keypath.RancherKeyPath, p.terratestConfig.PathToRepo, "")
			defer cleanup.Cleanup(p.T(), p.terraformOptions, keyPath)

			clusterIds, _ := provisioning.Provision(p.T(), p.client, p.standardUserClient, rancher, terraform, terratest, testUser, testPassword, p.terraformOptions, configMap, newFile, rootBody, file, false, false, true, clusterIDs, nil)

			cluster, err := p.client.Steve.SteveType(stevetypes.Provisioning).ByID(namespaces.FleetDefault + "/" + clusterIds[0])
			require.NoError(p.T(), err)

			logrus.Infof("Verifying the cluster is ready (%s)", cluster.Name)
			provisioning.VerifyClustersState(p.T(), p.client, clusterIDs)
			require.NoError(p.T(), err)

			logrus.Infof("Verifying cluster deployments (%s)", cluster.Name)
			err = deployment.VerifyClusterDeployments(p.client, cluster)
			require.NoError(p.T(), err)

			logrus.Infof("Verifying cluster pods (%s)", cluster.Name)
			err = pods.VerifyClusterPods(p.client, cluster)
			require.NoError(p.T(), err)

			workloadConfigs := new(workloads.Workloads)
			operations.LoadObjectFromMap(workloads.WorkloadsConfigurationFileKey, p.cattleConfig, workloadConfigs)

			logrus.Infof("Creating workloads on cluster: %s", cluster.Name)
			createdWorkloads, err := workloads.CreateWorkloads(p.client, cluster.Name, *workloadConfigs)
			require.NoError(p.T(), err)

			logrus.Infof("Verifying workloads on cluster: %s", cluster.Name)
			_, err = workloads.VerifyWorkloads(p.client, cluster.Name, *createdWorkloads)
			require.NoError(p.T(), err)

			err = imported.SetUpgradeImportedCluster(p.client, terraform)
			require.NoError(p.T(), err)

			logrus.Infof("Verifying workloads on cluster: %s", cluster.Name)
			_, err = workloads.VerifyWorkloads(p.client, cluster.Name, *createdWorkloads)
			require.NoError(p.T(), err)
		})

		params := tfpQase.GetProvisioningSchemaParams(configMap[0])
		err = qase.UpdateSchemaParameters(tt.name, params)
		if err != nil {
			logrus.Warningf("Failed to upload schema parameters %s", err)
		}
	}

	if p.terratestConfig.LocalQaseReporting {
		results.ReportTest(p.terratestConfig)
	}
}

func TestTfpUpgradeImportedClusterTestSuite(t *testing.T) {
	suite.Run(t, new(UpgradeImportedClusterTestSuite))
}
