package impl

const (
	CalicoIPAddr           = "cni.projectcalico.org~1ipAddrs" // cni.projectcalico.org/ipAddrs /为特殊字符在jsonPatch中要修改为~1
	RequiredPodAnnotations = "fix.pod.ip"
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
