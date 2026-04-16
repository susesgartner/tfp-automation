package nodepools

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rancher/tfp-automation/config"
	"github.com/rancher/tfp-automation/framework/set/defaults/providers/aws"
	"github.com/rancher/tfp-automation/framework/set/defaults/rancher2/clusters"
	rancher2resources "github.com/rancher/tfp-automation/framework/set/resources/rancher2"
)

type AWSInstanceGroup struct {
	ResourceName string
	AMI          string
	InstanceType string
	Quantity     int64
}

func UsesNodepools(terratestConfig *config.TerratestConfig) bool {
	return len(terratestConfig.Nodepools) > 0
}

func TotalNodeCount(terratestConfig *config.TerratestConfig) int64 {
	if !UsesNodepools(terratestConfig) {
		return terratestConfig.EtcdCount + terratestConfig.ControlPlaneCount + terratestConfig.WorkerCount
	}

	var totalNodeCount int64

	for _, pool := range terratestConfig.Nodepools {
		totalNodeCount += pool.Quantity
	}

	return totalNodeCount
}

func BuildRoleFlags(terraformConfig *config.TerraformConfig, terratestConfig *config.TerratestConfig) ([]string, error) {
	if !UsesNodepools(terratestConfig) {
		return buildRoleFlagsFromCounts(terratestConfig), nil
	}

	roleFlags := make([]string, 0, TotalNodeCount(terratestConfig))

	for count, pool := range terratestConfig.Nodepools {
		_, err := rancher2resources.SetResourceNodepoolValidation(terraformConfig, pool, strconv.Itoa(count))
		if err != nil {
			return nil, err
		}

		poolRoleFlags := buildPoolRoleFlags(pool)
		for i := int64(0); i < pool.Quantity; i++ {
			roleFlags = append(roleFlags, poolRoleFlags)
		}
	}

	return roleFlags, nil
}

func BuildAWSInstanceGroups(terraformConfig *config.TerraformConfig, terratestConfig *config.TerratestConfig) ([]AWSInstanceGroup, error) {
	if !UsesNodepools(terratestConfig) {
		return buildAWSInstanceGroupsFromCounts(terraformConfig, terratestConfig), nil
	}

	instanceGroups := make([]AWSInstanceGroup, 0, len(terratestConfig.Nodepools))

	for count, pool := range terratestConfig.Nodepools {
		_, err := rancher2resources.SetResourceNodepoolValidation(terraformConfig, pool, strconv.Itoa(count))
		if err != nil {
			return nil, err
		}

		ami := terraformConfig.AWSConfig.AMI
		instanceType := terraformConfig.AWSConfig.AWSInstanceType
		if pool.Worker && terraformConfig.MixedArchitecture {
			ami = terraformConfig.AWSConfig.ARMAMI
			instanceType = terraformConfig.AWSConfig.ARMInstanceType
		}

		instanceGroups = append(instanceGroups, AWSInstanceGroup{
			ResourceName: fmt.Sprintf("%s-pool-%d", terraformConfig.ResourcePrefix, count),
			AMI:          ami,
			InstanceType: instanceType,
			Quantity:     pool.Quantity,
		})
	}

	return instanceGroups, nil
}

func BuildAWSPublicIPExpression(terraformConfig *config.TerraformConfig, terratestConfig *config.TerratestConfig) (string, error) {
	instanceGroups, err := BuildAWSInstanceGroups(terraformConfig, terratestConfig)
	if err != nil {
		return "", err
	}

	groupExpressions := make([]string, 0, len(instanceGroups))
	for _, group := range instanceGroups {
		groupExpressions = append(groupExpressions, fmt.Sprintf("%s.%s.*.public_ip", aws.AwsInstance, group.ResourceName))
	}

	return fmt.Sprintf("flatten([%s])", strings.Join(groupExpressions, ", ")), nil
}

func buildRoleFlagsFromCounts(terratestConfig *config.TerratestConfig) []string {
	totalNodeCount := TotalNodeCount(terratestConfig)
	roleFlags := make([]string, 0, totalNodeCount)

	for i := int64(0); i < terratestConfig.EtcdCount; i++ {
		roleFlags = append(roleFlags, clusters.EtcdRoleFlag)
	}

	for i := int64(0); i < terratestConfig.ControlPlaneCount; i++ {
		roleFlags = append(roleFlags, clusters.ControlPlaneRoleFlag)
	}

	for i := int64(0); i < terratestConfig.WorkerCount; i++ {
		roleFlags = append(roleFlags, clusters.WorkerRoleFlag)
	}

	return roleFlags
}

func buildPoolRoleFlags(pool config.Nodepool) string {
	roleFlags := make([]string, 0, 3)

	if pool.Etcd {
		roleFlags = append(roleFlags, clusters.EtcdRoleFlag)
	}

	if pool.Controlplane {
		roleFlags = append(roleFlags, clusters.ControlPlaneRoleFlag)
	}

	if pool.Worker {
		roleFlags = append(roleFlags, clusters.WorkerRoleFlag)
	}

	return strings.Join(roleFlags, " ")
}

func buildAWSInstanceGroupsFromCounts(terraformConfig *config.TerraformConfig, terratestConfig *config.TerratestConfig) []AWSInstanceGroup {
	instanceGroups := []AWSInstanceGroup{
		{
			ResourceName: terraformConfig.ResourcePrefix + "-etcd",
			AMI:          terraformConfig.AWSConfig.AMI,
			InstanceType: terraformConfig.AWSConfig.AWSInstanceType,
			Quantity:     terratestConfig.EtcdCount,
		},
		{
			ResourceName: terraformConfig.ResourcePrefix + "-control-plane",
			AMI:          terraformConfig.AWSConfig.AMI,
			InstanceType: terraformConfig.AWSConfig.AWSInstanceType,
			Quantity:     terratestConfig.ControlPlaneCount,
		},
	}

	workerAMI := terraformConfig.AWSConfig.AMI
	workerInstanceType := terraformConfig.AWSConfig.AWSInstanceType
	if terraformConfig.MixedArchitecture {
		workerAMI = terraformConfig.AWSConfig.ARMAMI
		workerInstanceType = terraformConfig.AWSConfig.ARMInstanceType
	}

	instanceGroups = append(instanceGroups, AWSInstanceGroup{
		ResourceName: terraformConfig.ResourcePrefix + "-worker",
		AMI:          workerAMI,
		InstanceType: workerInstanceType,
		Quantity:     terratestConfig.WorkerCount,
	})

	return instanceGroups
}
