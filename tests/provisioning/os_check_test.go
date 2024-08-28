package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rancher/shepherd/clients/rancher"
	v1 "github.com/rancher/shepherd/clients/rancher/v1"
	"github.com/rancher/shepherd/extensions/configoperations"
	"github.com/rancher/shepherd/extensions/configoperations/permutations"
	"github.com/rancher/shepherd/extensions/defaults"
	"github.com/rancher/shepherd/extensions/defaults/stevetypes"
	"github.com/rancher/shepherd/extensions/sshkeys"
	"github.com/rancher/shepherd/extensions/steve"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/nodes"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tfp-automation/tests/extensions/permutationdata"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type OSCheck struct {
	suite.Suite
	client             *rancher.Client
	session            *session.Session
	standardUserClient *rancher.Client
	permutedConfigs    []map[string]any
	clusters           []*v1.SteveAPIObject
}

type SSHCluster struct {
	id    string
	nodes []*nodes.Node
}

func (k *OSCheck) TearDownSuite() {
	k.session.Cleanup()
}

func (k *OSCheck) SetupSuite() {
	testSession := session.NewSession()
	k.session = testSession

	client, err := rancher.NewClient("", testSession)
	require.NoError(k.T(), err)
	k.client = client

	cattleConfig := config.LoadConfigFromFile(os.Getenv(config.ConfigEnvironmentKey))

	k8sPermutation, err := permutationdata.CreateK8sPermutation(cattleConfig)
	require.NoError(k.T(), err)

	cniPermutation, err := permutationdata.CreateCNIPermutation(cattleConfig)
	require.NoError(k.T(), err)

	amiRelationships, err := permutationdata.CreateAMIRelationships(cattleConfig)

	providerPermutation, err := permutationdata.CreateProviderPermutation(cattleConfig)
	require.NoError(k.T(), err)
	providerPermutation.KeyPathValueRelationships = append(providerPermutation.KeyPathValueRelationships, amiRelationships...)

	k.permutedConfigs, err = permutations.Permute([]permutations.Permutation{providerPermutation, k8sPermutation, cniPermutation}, cattleConfig)
	require.NoError(k.T(), err)

}

func (k *OSCheck) TestProvisioning() {
	k.Run("test", func() {

		var configs []map[string]any

		modulePath := []string{"terraform", "module"}
		k8sVersionPath := []string{"terratest", "kubernetesVersion"}
		for _, permutedConfig := range k.permutedConfigs {
			k8sVersion, err := configoperations.GetValue(k8sVersionPath, permutedConfig)
			infraProvider, err := configoperations.GetValue(modulePath, permutedConfig)
			require.NoError(k.T(), err)

			var module string
			logrus.Info(module)
			if strings.Contains(k8sVersion.(string), "rancher") {
				module = infraProvider.(string) + "_rke1"
			} else if strings.Contains(k8sVersion.(string), "rke2") {
				module = infraProvider.(string) + "_rke2"
			} else if strings.Contains(k8sVersion.(string), "k3s") {
				module = infraProvider.(string) + "_k3s"
			}

			nodeDriverConfig, err := configoperations.DeepCopyMap(permutedConfig)
			require.NoError(k.T(), err)
			nodeDriverConfig, err = configoperations.ReplaceValue(modulePath, module+"_nodedriver", nodeDriverConfig)
			require.NoError(k.T(), err)

			customConfig, err := configoperations.DeepCopyMap(permutedConfig)
			require.NoError(k.T(), err)
			customConfig, err = configoperations.ReplaceValue(modulePath, module+"_custom", customConfig)
			require.NoError(k.T(), err)

			importedConfig, err := configoperations.DeepCopyMap(permutedConfig)
			require.NoError(k.T(), err)
			importedConfig, err = configoperations.ReplaceValue(modulePath, module+"_imported", importedConfig)
			require.NoError(k.T(), err)

			configs = append(configs, nodeDriverConfig, customConfig, importedConfig)
		}

		for _, permutedConfig := range configs {
			logrus.Info("---------------------------------------------")
			indented, _ := json.MarshalIndent(permutedConfig, "", "    ")
			converted := string(indented)
			fmt.Println(converted)
		}

		logrus.Info("------STATS------")
		logrus.Infof("Configs: %v", len(configs))
		logrus.Info("---------------------------------------------")

		/*
			for _, cluster := range k.clusters {
				provisioning.VerifyCluster(k.T(), k.client, nil, &cluster)
			}*/
	})
}

func (k *OSCheck) TestNodeReboot() {
	var SSHClusters []SSHCluster
	for _, cluster := range k.clusters {
		sshUser, err := sshkeys.GetSSHUser(k.client, cluster)
		require.NoError(k.T(), err)

		steveClient, err := steve.GetDownstreamClusterClient(k.client, cluster.ID)
		require.NoError(k.T(), err)

		nodesSteveObjList, err := steveClient.SteveType(stevetypes.Node).List(nil)
		require.NoError(k.T(), err)

		var sshNodes []*nodes.Node
		for _, node := range nodesSteveObjList.Data {
			clusterNode, err := sshkeys.GetSSHNodeFromMachine(k.client, sshUser, &node)
			require.NoError(k.T(), err)

			sshNodes = append(sshNodes, clusterNode)
		}

		SSHClusters = append(SSHClusters, SSHCluster{id: cluster.ID, nodes: sshNodes})
	}

	nodeNum := len(SSHClusters[0].nodes)
	for i := range nodeNum {
		for _, cluster := range SSHClusters {
			err := ec2.RebootNode(k.client, *cluster.nodes[i], cluster.id)
			require.NoError(k.T(), err)
		}

		for _, cluster := range clusters {
			err := steve.WaitForResourceState(k.client.Steve, cluster, "active", time.Second, defaults.FifteenMinuteTimeout)
			require.NoError(k.T(), err)
		}
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestOSCheckTestSuite(t *testing.T) {
	suite.Run(t, new(OSCheck))
}
