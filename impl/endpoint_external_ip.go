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

func MutateEndpointExternalIp(c *gin.Context) {
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

func ValidateEndpointExternalIp(c *gin.Context) {
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
		// external ip 没有开启直接返回
		if originalLabels[EndpointExternalIPEnableLabels] != "enabled" {
			return &v1beta1.AdmissionResponse{
				Allowed: true,
			}
		}
	default:
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}
	subset := corev1.EndpointSubset{
		Addresses: []corev1.EndpointAddress{},
		Ports:     []corev1.EndpointPort{},
	}
	if v, ok := originalLabels[RequiredServiceExternalIPLabels]; !ok {
		log.Errorf("Mutate: Required endpoint labels '%s' are not set", RequiredServiceExternalIPLabels)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: fmt.Sprintf("Mutate: Required endpoint labels '%s' are not set", RequiredServiceExternalIPLabels),
			},
		}
	} else {
		ip := strings.Split(v, "-")
		for _, i := range ip {
			if !CheckIp(i) {
				errorMassage := fmt.Sprintf("Validate: Required service labels '%s' '%s' format error. Example: 192.168.10.10-10.10.10.10", RequiredServiceExternalIPLabels, i)
				log.Errorf(errorMassage)
				return &v1beta1.AdmissionResponse{
					Result: &metav1.Status{
						Message: errorMassage,
					},
				}
			}
			subset.Addresses = append(subset.Addresses, corev1.EndpointAddress{IP: i})
		}
	}

	if v, ok := originalLabels[RequiredServiceExternalPortLabels]; !ok {
		log.Errorf("Mutate:  Required endpoint labels '%s' are not set", RequiredServiceExternalPortLabels)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: fmt.Sprintf("Mutate: Required endpoint labels '%s' are not set", RequiredServiceExternalPortLabels),
			},
		}
	} else {
		ports := strings.Split(v, "-")
		if !CheckNotDuplicate(ports) {
			errorMassage := fmt.Sprintf("Validate: Required service labels '%s' %v duplicate. Example: 80-8080", RequiredServiceExternalPortLabels, ports)
			log.Errorf(errorMassage)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: errorMassage,
				},
			}
		}
		for index, i := range ports {
			if !CheckPort(i) {
				errorMassage := fmt.Sprintf("Validate: Required service labels '%s' '%s' format error. Example: 80-8080", RequiredServiceExternalPortLabels, i)
				log.Errorf(errorMassage)
				return &v1beta1.AdmissionResponse{
					Result: &metav1.Status{
						Message: errorMassage,
					},
				}
			}
			port, _ := strconv.Atoi(i)
			subset.Ports = append(subset.Ports, corev1.EndpointPort{
				Name: strconv.Itoa(index),
				Port: int32(port),
			})
		}
	}
	patchSubset := addSubset(subset, len(endpoint.Subsets))
	patch = append(patch, patchSubset)
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
		originalServiceLabels = service.Labels
		// external ip 没有开启直接返回
		if originalServiceLabels[EndpointExternalIPEnableLabels] != "enabled" {
			return &v1beta1.AdmissionResponse{
				Allowed: true,
			}
		}
	default:
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	allowed := true
	var result *metav1.Status
	log.Info("Validate: original service labels: ", originalServiceLabels)
	// 校验externalIP是否存在
	if v, ok := originalServiceLabels[RequiredServiceExternalIPLabels]; !ok {
		allowed = false
		result = &metav1.Status{
			Reason: metav1.StatusReason(fmt.Sprintf("Validate: Required service labels '%s' are not set", RequiredServiceExternalIPLabels)),
		}
	} else {
		ip := strings.Split(v, "-")
		for _, i := range ip {
			if !CheckIp(i) {
				allowed = false
				result = &metav1.Status{
					Reason: metav1.StatusReason(fmt.Sprintf("Validate: Required service labels '%s' '%s' format error. Example: 192.168.10.10-10.10.10.10", RequiredServiceExternalIPLabels, i)),
				}
			}
		}
	}
	// 校验externalPort是否存在
	if v, ok := originalServiceLabels[RequiredServiceExternalPortLabels]; !ok {
		allowed = false
		result = &metav1.Status{
			Reason: metav1.StatusReason(fmt.Sprintf("Validate: Required service labels '%s' are not set", RequiredServiceExternalPortLabels)),
		}
	} else {
		port := strings.Split(v, "-")
		if !CheckNotDuplicate(port) {
			allowed = false
			result = &metav1.Status{
				Reason: metav1.StatusReason(fmt.Sprintf("Validate: Required service labels '%s' %v duplicate. Example: 80-8080", RequiredServiceExternalPortLabels, port)),
			}
		}
		for _, i := range port {
			if !CheckPort(i) {
				allowed = false
				result = &metav1.Status{
					Reason: metav1.StatusReason(fmt.Sprintf("Validate: Required service labels '%s' '%s' format error. Example: 80-8080", RequiredServiceExternalPortLabels, i)),
				}
			}
		}
	}
	return &v1beta1.AdmissionResponse{
		Allowed: allowed,
		Result:  result,
	}
}
