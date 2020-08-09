package checkOperaiton

import (
	"checkResource/getData"
)

func DoCheck() []byte {
	getData.SetFilteredNamespace()
	var mysqlPodList []string = getData.GetMysqlPodsList()
	var k8sPodList []string = getData.GetK8sPodsList()
	MysqlNotK8s, K8sNotMysql := getData.ComparePods(mysqlPodList, k8sPodList)
	responseMsg := getData.SendWarning(MysqlNotK8s, K8sNotMysql)
	return responseMsg
}
