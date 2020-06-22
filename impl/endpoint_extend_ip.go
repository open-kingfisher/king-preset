package impl

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/open-kingfisher/king-utils/common/log"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"

	"net/http"
)

func MutateEndpointExtendIp(c *gin.Context) {
	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if err := c.ShouldBindBodyWith(&ar, binding.JSON); err != nil {
		log.Errorf("Can't unmarshal body to AdmissionReview: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
		c.JSON(http.StatusInternalServerError, err)
		return
	} else {
		// mutate handle
		admissionResponse = mutateExternalIp(&ar)
		admissionReview := v1beta1.AdmissionReview{}
		if admissionResponse != nil {
			admissionReview.Response = admissionResponse
			if ar.Request != nil {
				admissionReview.Response.UID = ar.Request.UID
			}
		}
		c.JSON(http.StatusOK, admissionReview)
	}
}

func ValidateEndpointExtendIp(c *gin.Context) {
	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if err := c.ShouldBindBodyWith(&ar, binding.JSON); err != nil {
		log.Errorf("Can't unmarshal body to AdmissionReview: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
		c.JSON(http.StatusInternalServerError, err)
		return
	} else {
		// validate handle
		admissionResponse = validateService(&ar)
		admissionReview := v1beta1.AdmissionReview{}
		if admissionResponse != nil {
			admissionReview.Response = admissionResponse
			if ar.Request != nil {
				admissionReview.Response.UID = ar.Request.UID
			}
		}
		c.JSON(http.StatusOK, admissionReview)
	}
}

func mutateExternalIp(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var (
		originalLabels map[string]string
		patch          []patchOperation
	)

	log.Infof("Mutate: AdmissionReview: Kind=%v, Namespace=%v Name=%v UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)

	var endpoint corev1.Endpoints
	switch req.Kind.Kind {
	case "Endpoints":
		if err := json.Unmarshal(req.Object.Raw, &endpoint); err != nil {
			log.Errorf("Mutate: Can't unmarshal raw object to endpoint: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		log.Infof("Mutate: AdmissionReview Resource: %+v", endpoint)
		originalLabels = endpoint.Labels

	default:
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}
	subset := corev1.EndpointSubset{
		Addresses: []corev1.EndpointAddress{},
		Ports:     []corev1.EndpointPort{},
	}
	if originalLabels[EndpointExtend] == EndpointExternalIPEnableLabels {
		// 通过label获取ip
		allowed, result, ipList := getIPByLabels(RequiredServiceExternalIPLabels, originalLabels)
		if !allowed {
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: result,
				},
			}
		}
		for _, ip := range ipList {
			subset.Addresses = append(subset.Addresses, corev1.EndpointAddress{IP: ip})
		}
		// 通过label获取端口
		allowed, result, portList := getPortByLabels(RequiredServiceExternalPortLabels, originalLabels)
		if !allowed {
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: result,
				},
			}
		}
		for _, port := range portList {
			subset.Ports = append(subset.Ports, corev1.EndpointPort{
				Name: strconv.Itoa(port["name"]),
				Port: int32(port["port"]),
			})
		}
		patchSubset := addSubset(subset, len(endpoint.Subsets))
		patch = append(patch, patchSubset)
	}

	if originalLabels[EndpointExtend] == EndpointBackupIPEnableLabels {
		// 通过label获取ip
		allowed, result, backupIpList := getIPByLabels(RequiredServiceBackupIPLabels, originalLabels)
		if !allowed {
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: result,
				},
			}
		}

		if endpoint.Subsets != nil {
			originalIP := make([]string, 0)
			for _, subsets := range endpoint.Subsets {
				for _, addresses := range subsets.Addresses {
					originalIP = append(originalIP, addresses.IP)
				}
			}
			// 原始IP和backupIP不相等的情况，才去移除，相等说明要启用backupIP
			if !EqualSlice(originalIP, backupIpList) {
				for addressesIndex, subsets := range endpoint.Subsets {
					patchAddresses := make([]corev1.EndpointAddress, 0)
					for _, addresses := range subsets.Addresses {
						state := func() bool {
							tmp := false
							for _, backupIp := range backupIpList {
								if addresses.IP == backupIp {
									tmp = true
								}
							}
							return tmp
						}()
						// state true 说明原始的IP在备份列表中，跳过这次添加
						if state {
							continue
						}
						// 最终的结果
						patchAddresses = append(patchAddresses, addresses)
					}
					if len(patchAddresses) != 0 {
						patchSubset := replaceAddresses(patchAddresses, addressesIndex)
						patch = append(patch, patchSubset)
					}
				}
			}
		}
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		errorMassage := fmt.Sprintf("json.Marshal patch: '%+v' error: %s", patch, err)
		log.Errorf(errorMassage)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: errorMassage,
			},
		}
	}

	log.Infof("Mutate: AdmissionResponse Patch: %v\n", string(patchBytes))
	// 当前仅支持patchType为JSONPatch的AdmissionResponse
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType { // 不用单独生成变量，直接命名函数返回
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func validateService(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var (
		originalServiceLabels map[string]string
		resourceName          string
	)
	log.Infof("Validate: AdmissionReview: Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, resourceName, req.UID, req.Operation, req.UserInfo)

	switch req.Kind.Kind {
	case "Service":
		var service corev1.Service
		if err := json.Unmarshal(req.Object.Raw, &service); err != nil {
			log.Errorf("Validate: Can't unmarshal raw object to Service: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName = service.Name
		// endpoint-external-ip: enabled
		// 例如:  externalIP: "192.168.10.1-192.168.10.11"
		// externalPort: "80-8080"
		// endpoint-backup-ip: enabled
		// backupIP: "192.168.10.1-192.168.10.11"
		originalServiceLabels = service.Labels

	default:
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	log.Info("Validate: original service labels: ", originalServiceLabels)
	// external ip 功能相关校验
	if originalServiceLabels[EndpointExtend] == EndpointExternalIPEnableLabels {
		// 校验externalIP是否存在，是否符合规定
		if allowed, result := validateIP(RequiredServiceExternalIPLabels, originalServiceLabels); !allowed {
			return &v1beta1.AdmissionResponse{
				Allowed: allowed,
				Result: &metav1.Status{
					Reason: metav1.StatusReason(result),
				},
			}
		}
		// 校验externalPort是否存在，是否符合规定
		if allowed, result := validatePort(RequiredServiceExternalPortLabels, originalServiceLabels); !allowed {
			return &v1beta1.AdmissionResponse{
				Allowed: allowed,
				Result: &metav1.Status{
					Reason: metav1.StatusReason(result),
				},
			}
		}
	}
	// backup ip 相关功能
	if originalServiceLabels[EndpointExtend] == EndpointBackupIPEnableLabels {
		// 校验backupIP是否存在，是否符合规定
		if allowed, result := validateIP(RequiredServiceBackupIPLabels, originalServiceLabels); !allowed {
			return &v1beta1.AdmissionResponse{
				Allowed: allowed,
				Result: &metav1.Status{
					Reason: metav1.StatusReason(result),
				},
			}
		}
	}

	return &v1beta1.AdmissionResponse{
		Allowed: true,
	}
}

// 校验IP是否存在，是否符合规定
func validateIP(label string, originalServiceLabels map[string]string) (allowed bool, result string) {
	allowed = true
	if v, ok := originalServiceLabels[label]; !ok {
		allowed = false
		result = fmt.Sprintf("Validate: Required service labels '%s' are not set", label)
	} else {
		ip := strings.Split(v, "-")
		for _, i := range ip {
			if !CheckIp(i) {
				allowed = false
				result = fmt.Sprintf("Validate: Required service labels '%s' '%s' format error. Example: 192.168.10.10-10.10.10.10", label, i)
			}
		}
	}
	return
}

// 校验externalPort是否存在
func validatePort(label string, originalServiceLabels map[string]string) (allowed bool, result string) {
	allowed = true
	if v, ok := originalServiceLabels[label]; !ok {
		allowed = false
		result = fmt.Sprintf("Validate: Required service labels '%s' are not set", label)
	} else {
		port := strings.Split(v, "-")
		if !CheckNotDuplicate(port) {
			allowed = false
			result = fmt.Sprintf("Validate: Required service labels '%s' %v duplicate. Example: 80-8080", label, port)
		}
		for _, i := range port {
			if !CheckPort(i) {
				allowed = false
				result = fmt.Sprintf("Validate: Required service labels '%s' '%s' format error. Example: 80-8080", label, i)
			}
		}
	}
	return
}

// 获取IP
func getIPByLabels(label string, originalLabels map[string]string) (allowed bool, result string, ip []string) {
	if v, ok := originalLabels[label]; !ok {
		errorMassage := fmt.Sprintf("Mutate: Required endpoint labels '%s' are not set", label)
		log.Errorf(errorMassage)
		return allowed, errorMassage, ip
	} else {
		ipList := strings.Split(v, "-")
		for _, i := range ipList {
			if !CheckIp(i) {
				errorMassage := fmt.Sprintf("Mutate: Required service labels '%s' '%s' format error. Example: 192.168.10.10-10.10.10.10", label, i)
				log.Errorf(errorMassage)
				return allowed, errorMassage, ip
			}
			ip = append(ip, i)
		}
	}
	return true, result, ip
}

// 获取端口
func getPortByLabels(label string, originalLabels map[string]string) (allowed bool, result string, port []map[string]int) {
	if v, ok := originalLabels[label]; !ok {
		errorMassage := fmt.Sprintf("Mutate: Required endpoint labels '%s' are not set", label)
		log.Errorf(errorMassage)
		return allowed, errorMassage, port
	} else {
		portList := strings.Split(v, "-")
		if !CheckNotDuplicate(portList) {
			errorMassage := fmt.Sprintf("Mutate: Required service labels '%s' %v duplicate. Example: 80-8080", label, portList)
			log.Errorf(errorMassage)
			return allowed, errorMassage, port
		}
		for index, i := range portList {
			if !CheckPort(i) {
				errorMassage := fmt.Sprintf("Mutate: Required service labels '%s' '%s' format error. Example: 80-8080", label, i)
				log.Errorf(errorMassage)
				return allowed, errorMassage, port
			}
			portInt, _ := strconv.Atoi(i)
			port = append(port, map[string]int{
				"name": index,
				"port": portInt,
			})
		}
	}
	return true, result, port
}
