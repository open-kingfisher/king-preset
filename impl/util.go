package impl

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"reflect"
	"regexp"
)

const (
	CalicoIPAddr                      = "cni.projectcalico.org~1ipAddrs" // cni.projectcalico.org/ipAddrs /为特殊字符在jsonPatch中要修改为~1
	RequiredPodAnnotations            = "fix.pod.ip"
	EndpointExternalIPEnableLabels    = "endpoint-external-ip"
	RequiredServiceExternalIPLabels   = "externalIP"
	RequiredServiceExternalPortLabels = "externalPort"

	EndpointBackupIPEnableLabels  = "endpoint-backup-ip"
	RequiredServiceBackupIPLabels = "backupIP"

	Enabled                                = "enabled"
	Disabled                               = "disabled"
	EndpointExtend                         = "endpoint-extend"
	InjectLogSidecarRequiredPodAnnotations = "log-injection"
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

// 为endpoint删除ip
func deleteAddresses(addressesIndex, ipIndex int) (patch patchOperation) {
	return patchOperation{
		Op:   "remove",
		Path: fmt.Sprintf("/subsets/%d/addresses/%d", addressesIndex, ipIndex),
	}
}

// 为endpoint删除ip
func replaceAddresses(addresses []corev1.EndpointAddress, addressesIndex int) (patch patchOperation) {
	return patchOperation{
		Op:    "replace",
		Path:  fmt.Sprintf("/subsets/%d/addresses", addressesIndex),
		Value: addresses,
	}
}

// 为Containers添加log container
func addLogContainer(index int) (patch patchOperation) {
	container := corev1.Container{
		Name:  "king-log",
		Image: "registry.wap.sina.cn/kingfisher/king-exporter:v1.1",
		VolumeMounts: []corev1.VolumeMount{
			corev1.VolumeMount{
				Name:      "king-log",
				ReadOnly:  true,
				MountPath: "/opt",
			},
		},
	}
	return patchOperation{
		Op:    "add",
		Path:  fmt.Sprintf("/spec/containers/%d", index),
		Value: container,
	}
}

// 为Volumes添加configMap
func addLogVolume(index int) (patch patchOperation) {
	volume := corev1.Volume{
		Name: "king-log",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "log",
				},
				DefaultMode: func() *int32 {
					var mode int32 = 420
					return &mode
				}(),
			},
		},
	}
	if index == 0 {
		return patchOperation{
			Op:    "add",
			Path:  "/volumes",
			Value: volume,
		}
	} else {
		return patchOperation{
			Op:    "add",
			Path:  fmt.Sprintf("/spec/volumes/%d", index),
			Value: volume,
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

// 比较slice是否相等
func EqualSlice(a, b []string) bool {
	return reflect.DeepEqual(a, b)
}
