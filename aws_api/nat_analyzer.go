package aws_api

func AnalyzeFlow(region string, subnetIds []string, networkInterfacesFilePath string) error {
	go startNetworkInterfaceRecorder(region)
	logGroups := getSubnetFlowLogGroups(subnetIds)
	startLogScraper(logGroups)
	return nil
}

func startNetworkInterfaceRecorder(region string) {

}

func getSubnetFlowLogGroups(subnetIds []string) []string {
	ret := []string{}

	return ret
}

func startLogScraper(logGroups []string) {

}
