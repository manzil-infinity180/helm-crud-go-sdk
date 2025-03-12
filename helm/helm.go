package helm

import (
	"fmt"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
)

// act as secrets
var helmDriver string = os.Getenv("HELM_DRIVER")

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// creating our custom RESTClientGetter or use settings.RESTClientGetter()
func getKubeConfigWithContext(contextName string) (genericclioptions.RESTClientGetter, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		if home := homeDir(); home != "" {
			kubeconfig = fmt.Sprintf("%s/.kube/config", home)
		}
	}

	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %v", err)
	}

	// Ensure context exists
	if _, exists := config.Contexts[contextName]; !exists {
		return nil, fmt.Errorf("context %s not found in kubeconfig", contextName)
	}

	configFlags := genericclioptions.NewConfigFlags(true)
	configFlags.KubeConfig = &kubeconfig
	configFlags.Context = &contextName

	return configFlags, nil
}

func initActionConfig(settings *cli.EnvSettings, namespace, contextName string) (*action.Configuration, error) {
	restClientGetter, err := getKubeConfigWithContext(contextName)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig")
	}
	actionConfig := new(action.Configuration)

	if namespace == "" {
		namespace = settings.Namespace()
	}

	if err := actionConfig.Init(
		//settings.RESTClientGetter(),
		restClientGetter,
		namespace,
		helmDriver,
		log.Printf); err != nil {
		return nil, err
	}

	return actionConfig, nil
}

/*
TODO: ADD SUPPORT FOR REMOTE
installClient.ChartPathOptions.RepoURL = "https://charts.bitnami.com/bitnami"
release, err := installClient.Run("nginx", nil) // Install the Nginx Helm chart
*/

func InstallHelmChartFromRemoteUrl(c *gin.Context) {
	rel, err := installOrUpgradeRelease("hello-nginxx", "https://charts.bitnami.com/bitnami", "nginx", nil, "default")
	//rel, err := installOrUpgradeRelease2("add-wds6",
	//	"oci://ghcr.io/kubestellar/kubestellar/core-chart",
	//	"0.26.0",
	//	map[string]interface{}{
	//		"kubeflex-operator.install": false,
	//		"InstallPCHs":               false,
	//		"WDSes":                     []map[string]string{{"name": "wds6"}},
	//	}, "default",
	//)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"release": rel,
	})
}
func installOrUpgradeRelease(releaseName string, repoUrl string, chartName string, values map[string]interface{}, targetNamespace string) (*release.Release, error) {
	settings := cli.New()
	//actionConfig, err := initActionConfig(settings, "default", "kind-kubeflex")
	actionConfig, err := initActionConfig(settings, "default", "wds1")
	if err != nil {
		return nil, err
	}

	var chartPathOptions action.ChartPathOptions = action.ChartPathOptions{
		RepoURL: repoUrl,
	}

	chart, err := getChart(chartPathOptions, chartName, settings)
	fmt.Println(chart)
	if err != nil {
		return nil, err
	}

	var rel *release.Release
	var requestError error

	histClient := action.NewHistory(actionConfig)
	histClient.Max = 1
	if _, err := histClient.Run(releaseName); err == driver.ErrReleaseNotFound {
		clientInstall := action.NewInstall(actionConfig)
		clientInstall.ReleaseName = releaseName
		clientInstall.Namespace = targetNamespace
		clientInstall.ChartPathOptions = chartPathOptions

		fmt.Println(clientInstall)

		rel, requestError = clientInstall.Run(chart, values)
	} else {
		clientUpgrade := action.NewUpgrade(actionConfig)
		clientUpgrade.Namespace = targetNamespace
		clientUpgrade.ChartPathOptions = chartPathOptions

		fmt.Println(clientUpgrade)

		rel, requestError = clientUpgrade.Run(releaseName, chart, values)
	}

	return rel, requestError
}

func getChart(chartPathOption action.ChartPathOptions, chartName string, settings *cli.EnvSettings) (*chart.Chart, error) {
	chartPath, err := chartPathOption.LocateChart(chartName, settings)
	if err != nil {
		return nil, err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}

	return chart, nil
}

func InstallHelmChart(c *gin.Context) {
	settings := cli.New()

	actionConfig, err := initActionConfig(settings, "", "wds1")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to init action config",
			"error": err.Error()})
		return
	}
	client := action.NewInstall(actionConfig)
	/**
	* We need to care about 3 things their
	* ReleaseName, Namespace and chartPath
	 */
	type parameters struct {
		ReleaseName  string                 `json:"releaseName"`
		ChartPath    string                 `json:"chartPath"`
		Namespace    string                 `json:"namespace"`
		ReleaseValue map[string]interface{} `json:"releaseValue"`
	}
	params := parameters{}
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client.ReleaseName = params.ReleaseName
	client.Namespace = params.Namespace

	chart, err := loader.Load(params.ChartPath) // it will locally load the helm chart file
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "issue while loading the chartPath",
			"error":   err.Error()})
		return
	}
	// TODO: add the logic to fetch the releaseValue (values.yaml)
	//_, err = client.Run(chart, params.ReleaseValue)
	_, err = client.Run(chart, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "issue from client.Run",
			"error":   err.Error()})
		return
	}

	fmt.Printf("Chart %s installed successfully\n", params.ReleaseName)
	c.JSON(http.StatusCreated, gin.H{
		"message": fmt.Sprintf("%s release installed successfully", params.ReleaseName),
	})

}
func GetReleaseList(c *gin.Context) {
	settings := cli.New()

	actionConfig, err := initActionConfig(settings, "", "wds1")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to init action config",
			"error": err.Error()})
		return
	}

	client := action.NewList(actionConfig)
	// Only list deployed
	client.Deployed = true
	results, err := client.Run()

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "issue from client.Run",
			"error":   err.Error()})
		return
	}
	// TODO: Return what you need
	// the template it return is of base64
	c.JSON(http.StatusOK, gin.H{
		"result": results,
	})
}
func GetRelease(c *gin.Context) {
	settings := cli.New()
	actionConfig, err := initActionConfig(settings, "", "wds1")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to init action config",
			"error": err.Error()})
		return
	}
	client := action.NewGet(actionConfig)
	type parameters struct {
		ReleaseName string `json:"releaseName"`
	}
	params := parameters{}
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	release, err := client.Run(params.ReleaseName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "issue from client.Run",
			"error":   err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("%s release fetched successfully", params.ReleaseName),
		"result":  release,
	})
}
func UpgradeChart(c *gin.Context) {
	settings := cli.New()
	actionConfig, err := initActionConfig(settings, "", "wds1")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to init action config",
			"error": err.Error()})
		return
	}
	client := action.NewUpgrade(actionConfig)

	type parameters struct {
		ReleaseName string `json:"releaseName"`
		ChartPath   string `json:"chartPath"`
		Namespace   string `json:"namespace"`
	}
	params := parameters{}
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fmt.Println(settings.RESTClientGetter())
	fmt.Println(settings.Namespace())
	client.Namespace = params.Namespace
	chart, err := loader.Load(params.ChartPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "issue from loader.Load",
			"error":   err.Error()})
		return
	}
	//_, err = client.Run(params.ReleaseName, chart, params.ReleaseValue)
	_, err = client.Run(params.ReleaseName, chart, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "issue from client.Run",
			"error":   err.Error()})
		return
	}
	fmt.Printf("Chart %s upgraded successfully\n", params.ReleaseName)
	c.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("%s release chart upgrade successfully", params.ReleaseName),
		"namespace": settings.Namespace(),
	})
}
func DeleteChart(c *gin.Context) {
	settings := cli.New()
	actionConfig, err := initActionConfig(settings, "", "wds1")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to init action config",
			"error": err.Error()})
		return
	}
	type parameters struct {
		ReleaseName string `json:"releaseName"`
	}
	params := parameters{}
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client := action.NewUninstall(actionConfig)

	_, err = client.Run(params.ReleaseName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "issue from client.Run",
			"error":   err.Error()})
		return
	}
	fmt.Printf("Chart %s deleted successfully\n", params.ReleaseName)
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("%s release chart uninstalled successfully", params.ReleaseName),
	})
}
func RollbackChart(c *gin.Context) {
	settings := cli.New()
	actionConfig, err := initActionConfig(settings, "", "wds1")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to init action config",
			"error": err.Error()})
		return
	}
	client := action.NewRollback(actionConfig)

	client.Version = 2 // set the version
	type parameters struct {
		ReleaseName string `json:"releaseName"`
	}
	params := parameters{}
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = client.Run(params.ReleaseName)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("%s release chart rollback successfully", params.ReleaseName),
	})
}
