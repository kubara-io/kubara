package cluster_test

import (
	"context"
	"testing"

	flags "github.com/kubara-io/kubara/cmd"
	"github.com/kubara-io/kubara/cmd/cluster"
	"github.com/kubara-io/kubara/cmd/testutil"
	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClusterCommand(t *testing.T) {

	command := cluster.NewClusterCommand()

	assert.Equal(t, "cluster", command.Name)
	assert.Equal(t, "Manage your kubara cluster configurations", command.Usage)
	assert.Equal(t, "kubara cluster [command]", command.UsageText)
	assert.Equal(t, "Enables the configuration and quick setup of clusters", command.Description)

	addCommand := cluster.CreateAddClusterCommand()

	assert.Equal(t, "add", addCommand.Name)
	assert.Equal(t, "Add a new spoke cluster to your config", addCommand.Usage)
	assert.Equal(t, "kubara cluster add <cluster-name>", addCommand.UsageText)
	assert.Equal(t, "Adds a new spoke cluster to an existing config yaml", addCommand.Description)

	listCommand := cluster.CreateClusterList()

	assert.Equal(t, "list", listCommand.Name)
	assert.Equal(t, "List all the cluster in the config file", listCommand.Usage)
	assert.Equal(t, "kubara cluster ls", listCommand.UsageText)
	assert.Equal(t, "List all the clusters available in the current config.yaml file", listCommand.Description)

}

func TestListAllClustersNoError(t *testing.T) {
	dir := t.TempDir()
	configPath := testutil.CreateTestConfig(t, dir, testutil.CreateTestCluster(t))

	testutil.CreateDefaultGenerateTestEnv(t, dir)

	cliFlags := flags.NewGlobalFlags().CLIFlags()
	app := testutil.CreateTestAppWithFlags(cliFlags, cluster.NewClusterCommand())
	args := []string{"kubara", "--config-file", configPath, "--work-dir", dir, "cluster", "list"}
	err := app.Run(context.Background(), args)
	require.NoError(t, err)
}

func TestAddNewSpokesCluster(t *testing.T) {
	spokeName := "coolNewSpoke"
	dir := t.TempDir()
	configPath := testutil.CreateTestConfig(t, dir, testutil.CreateTestCluster(t))

	testutil.CreateDefaultGenerateTestEnv(t, dir)

	cliFlags := flags.NewGlobalFlags().CLIFlags()
	app := testutil.CreateTestAppWithFlags(cliFlags, cluster.NewClusterCommand())
	args := []string{"kubara", "--config-file", configPath, "--work-dir", dir, "cluster", "add", spokeName}
	err := app.Run(context.Background(), args)

	require.NoError(t, err)
	configStore := config.NewConfigStoreWithCatalog(configPath, catalog.LoadOptions{})
	configStore.Load()

	currentConfig := configStore.GetConfig()

	//implicit assumption that the clusters are sorted by time added
	spokeCluster := currentConfig.Clusters[1]
	assert.Equal(t, spokeName, spokeCluster.Name)
}
