package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/manzil-infinity180/helm-crud-go-sdk/helm"
)

func setupHelmReleaseRoutes(router *gin.Engine) {
	router.POST("/api/helm", helm.InstallHelmChart)
	router.GET("/api/helm", helm.GetRelease)
	router.GET("/api/helm/list", helm.GetReleaseList)
	router.PUT("/api/helm", helm.UpgradeChart)
	router.DELETE("/api/helm", helm.DeleteChart)
	router.PUT("/api/helm/version", helm.RollbackChart)
	router.POST("/api/remote", helm.InstallHelmChartFromRemoteUrl)
}

func SetupRoutes(router *gin.Engine) {
	// Initialize all route groups
	setupHelmReleaseRoutes(router)
}
