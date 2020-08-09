package getData

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	conf "checkResource/conf"

	//"git.code.oa.com/SCF/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	//"encoding/csv"

	//"io"
	"fmt"
	"log"

	//"os"
	//"strconv"
	_ "github.com/go-sql-driver/mysql"
)

// global variable as filter map of namespace
var FilteredNamespace map[string]int = make(map[string]int)

// convert string list to map
func SetFilteredNamespace() {
	var FilteredNSStr []string = conf.Cfg.K8sCluster.FilteredNS
	for _, itemStr := range FilteredNSStr {
		FilteredNamespace[itemStr] = 1
	}
}

func ConnMysql(user, passwd, ip, port, db string) *sql.DB {
	DBDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, passwd, ip, port, db)
	mysqlDB, err := sql.Open("mysql", DBDSN)
	//mysqlDB.SetConnMaxLifeTime()
	if err != nil {
		panic(err.Error())
	}
	return mysqlDB
}

func GetMysqlPodsList() (MysqlPodsList []string) {

	// this part can be optimized as softcode with config file
	ip := conf.Cfg.Mysql.Ip
	port := conf.Cfg.Mysql.Port
	user := conf.Cfg.Mysql.User
	passwd := conf.Cfg.Mysql.Passwd
	db := conf.Cfg.Mysql.Db
	mysql := ConnMysql(user, passwd, ip, port, db)
	defer func() {
		mysql.Close()
		fmt.Println("MySQL Connection closed!")
	}()
	fmt.Println("MySQL Connected!")

	// get each pod info in db
	queryCmd := "SELECT `vm_id`, `pod_name`, `namespace` FROM `t_vm` "
	queryCmd += "WHERE `status` NOT IN (10, 31) AND TimeStampDiff(MINUTE, `create_time`, NOW())>30"
	rows, err := mysql.Query(queryCmd)
	if err != nil {
		log.Println(err)
	}
	defer rows.Close()
	//var mysqlPodsList []string

	for rows.Next() {
		var vmID string
		var mysqlPod string
		var podNS string
		if err := rows.Scan(&vmID, &mysqlPod, &podNS); err != nil {
			log.Println(err)
			continue
		}

		// filter namespace
		if _, ok := FilteredNamespace[podNS]; ok {
			continue
		}

		if conf.Cfg.IsFilterNewVK && len(mysqlPod) > 3 && mysqlPod[:3] == "ts-" {
			continue
		}

		MysqlPodsList = append(MysqlPodsList, vmID)
	}

	if err != nil {
		log.Println(err)
	}

	/*
		fmt.Println("Pods in Mysql:")
		for _, pod1 := range MysqlPodsList {
			fmt.Println(pod1)
		}
		fmt.Println("")
	*/
	return MysqlPodsList
}

func GetK8sPodsList() []string {
	var err error

	// set path of config file
	k8sConfigPath := conf.Cfg.K8sCluster.ConfigPath
	// get k8s config
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", k8sConfigPath)
	if err != nil {
		log.Println(err)
	}
	// connect k8s
	clientSet, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		log.Println(err)
	}
	fmt.Println("K8S Connected Successful")

	// get k8s pod info
	podsList, err := clientSet.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		log.Println(err)
	}

	var k8sPodsNameList []string

	// set 30-mins-ago time point with format required by api
	var timeCheckpoint metav1.Time = metav1.NewTime(time.Now().Add(-time.Minute * 30))

	for _, item := range podsList.Items {
		// ignore pods created in 30 mins
		if !item.CreationTimestamp.Before(&timeCheckpoint) {
			continue
		}
		// filter pods by namespace
		if _, ok := FilteredNamespace[item.Namespace]; ok {
			continue
		}

		if conf.Cfg.IsFilterNewVK && len(item.Name) > 3 && item.Name[:3] == "ts-" {
			continue
		}

		k8sPodsNameList = append(k8sPodsNameList, item.Name)
	}
	/*
		fmt.Println("Pods in K8s:")
		for _, pod2 := range k8sPodsNameList {
			fmt.Println(pod2)
		}
		fmt.Println("")
	*/
	return k8sPodsNameList
}

func ComparePods(mysqlPods []string, k8sPods []string) (MysqlNotK8s []string, K8sNotMysql []string) {
	//var MysqlNotK8s []string
	//var K8sNotMysql []string
	mysqlPodsStat := make(map[string]bool)
	for _, mysqlPod := range mysqlPods {
		mysqlPodsStat[mysqlPod] = false
	}

	for _, k8sPod := range k8sPods {

		if _, ok := mysqlPodsStat[k8sPod]; ok {
			mysqlPodsStat[k8sPod] = true
			continue
		} else {
			K8sNotMysql = append(K8sNotMysql, k8sPod)
		}
	}
	for k, v := range mysqlPodsStat {
		if !v {
			MysqlNotK8s = append(MysqlNotK8s, k)
		}
	}

	fmt.Println("Pods in Mysql not K8s:")
	for _, pod1 := range MysqlNotK8s {
		fmt.Println(pod1)
	}
	fmt.Println("")
	fmt.Println("Pods in K8s not Mysql:")
	for _, pod2 := range K8sNotMysql {
		fmt.Println(pod2)
	}

	return MysqlNotK8s, K8sNotMysql

}

func SendWarning(MysqlNotK8s []string, K8sNotMysql []string) []byte {
	var ChatIdList []string = conf.Cfg.Alert.ChadId
	// statusCode = 0 表示成功
	// statusCode = 1 表示创建请求失败
	// statusCode = 2 表示执行请求失败
	// statusCode = 3 表示Json解析失败
	var statusCode int = 0
	var statusMsg string = "Success"

	type DirtyData struct {
		StatusCode	int
		StatusMsg	string
		InK8sNotMysql []string
		InMysqlNotK8s []string
	}

	var DData DirtyData
	for _, receiver := range ChatIdList {

		var DD DirtyData
		var WarningText string
		WarningText += "Dirty data list:\\n\\n"
		WarningText += "Pods in Mysql not K8s:\\n"
		for _, item := range MysqlNotK8s {
			WarningText += item + "\\n"
			DD.InMysqlNotK8s = append(DD.InMysqlNotK8s, item)
		}
		WarningText += "\\nPods in K8s not Mysql:\\n"
		for _, item := range K8sNotMysql {
			WarningText += item + "\\n"
			DD.InK8sNotMysql = append(DD.InK8sNotMysql, item)
		}

		JsText := "[{\"type\":\"text\", \"text\": \"" + WarningText + "\"}]"
		JsToSend := []byte(JsText)
		Url := "http://9.109.39.49:8090/api/v1/msg?chatid=" + receiver + "&style=" + conf.Cfg.Alert.Style
		req, err := http.NewRequest("POST", Url, bytes.NewBuffer(JsToSend))
		if err != nil {
			statusCode = 1
			statusMsg = "Error! Creating request failed!"
			log.Println(err)
			break
		}
		req.Header.Add("X-Token", "pctPZDtm")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			statusCode = 2
			statusMsg = "Error! Executing request failed!"
			log.Println(err)
			break
		}
		defer resp.Body.Close()

		statuscode := resp.StatusCode
		hea := resp.Header
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("\nRequest Part:")
		fmt.Println(statuscode)
		fmt.Println(hea)
		fmt.Println(string(body))
		fmt.Println("")
		DData = DD
	}
	DData.StatusCode = statusCode;
	DData.StatusMsg = statusMsg;
	JsToReturn, err := json.Marshal(DData)
	if err != nil {
		log.Println(err)
		returnMsg := "{\"StatusCode\":3,\"StatusMsg\":\"Marshal Json failed!\"}"
		return ([]byte)(returnMsg)
	}
	return JsToReturn
}
