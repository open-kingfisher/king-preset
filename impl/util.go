package impl

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"regexp"
)

const (
	CalicoIPAddr                      = "cni.projectcalico.org~1ipAddrs" // cni.projectcalico.org/ipAddrs /为特殊字符在jsonPatch中要修改为~1
	RequiredPodAnnotations            = "fix.pod.ip"
	EndpointExternalIPEnableLabels    = "endpoint-external-ip"
	RequiredServiceExternalIPLabels   = "externalIP"
	RequiredServiceExternalPortLabels = "externalPort"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// 为Pod指定特定的Node
func mutateNodeName(nodeName string) (patch []patchOperation) {
	return append(patch, patchOperation{
		Op:    "add",
		Path:  "/spec/nodeName",
		Value: nodeName,
	})
}

// 为Pod添加注解使用calico 'cni.projectcalico.org/ipAddrs' 这个特性
func addAnnotation(ipAddr string) (patch []patchOperation) {
	return append(patch, patchOperation{
		Op:    "add",
		Path:  "/metadata/annotations/" + CalicoIPAddr,
		Value: ipAddr,
	})
}

// 为endpoint添加ip
func addSubset(subset corev1.EndpointSubset, index int) (patch patchOperation) {
	if index == 0 {
		return patchOperation{
			Op:   "add",
			Path: "/subsets",
			Value: []corev1.EndpointSubset{
				subset,
			},
		}
	} else {
		return patchOperation{
			Op:    "add",
			Path:  fmt.Sprintf("/subsets/%d", index),
			Value: subset,
		}
	}
}

// 检查IP地址是否合法
func CheckIp(ip string) bool {
	//addr := strings.Trim(ip, " ")
	regStr := `^(([1-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.)(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){2}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`
	if match, _ := regexp.MatchString(regStr, ip); match {
		return true
	}
	return false
}

// 检查Port是否合法
func CheckPort(port string) bool {
	regStr := `^([1-9]|[1-9]\d{1,3}|[1-5]\d{4}|6[0-5]{2}[0-3][0-5])$`
	if match, _ := regexp.MatchString(regStr, port); match {
		return true
	}
	return false
}

// slice item 重复检查
func CheckNotDuplicate(list []string) bool {
	tmp := make(map[string]string)
	for _, i := range list {
		tmp[i] = ""
	}
	if len(tmp) != len(list) {
		return false
	}
	return true
}
