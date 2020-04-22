package impl

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/open-kingfisher/king-utils/common/log"
	"k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"

	"net/http"
)

func MutateFixPodIP(c *gin.Context) {
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
		admissionResponse = mutate(&ar)
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

func ValidateFixPodIP(c *gin.Context) {
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
		admissionResponse = validate(&ar)
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

func mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var (
		originalAnnotations map[string]string
		resourceName        string
		generateName        string
		patch               []patchOperation
	)

	log.Infof("Mutate: AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, resourceName, req.UID, req.Operation, req.UserInfo)

	switch req.Kind.Kind {
	case "Pod":
		var pod corev1.Pod
		if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
			log.Errorf("Mutate: Can't unmarshal raw object to pod: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		log.Infof("Mutate: AdmissionReview Resource: %+v", pod)
		resourceName, generateName, originalAnnotations = pod.Name, pod.GenerateName, pod.Annotations
	default:
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	if v, ok := originalAnnotations[RequiredPodAnnotations]; !ok {
		log.Errorf("Required pod annotation '%s' are not set", RequiredPodAnnotations)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: fmt.Sprintf("Mutate: Required pod annotation '%s' are not set", RequiredPodAnnotations),
			},
		}
	} else {
		ip := []map[string][]string{}
		if err := json.Unmarshal([]byte(v), &ip); err != nil {
			errorMassage := fmt.Sprintf("Mutate: Unmarshal '%s' value error: %s", RequiredPodAnnotations, err)
			log.Errorf(errorMassage)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: errorMassage,
				},
			}
		} else {
			podNumString := strings.TrimPrefix(resourceName, generateName)
			if podNum, err := strconv.Atoi(podNumString); err != nil {
				errorMassage := fmt.Sprintf("Mutate: strconv.Atoi '%s' to int error: %s", podNumString, err)
				log.Errorf(errorMassage)
				return &v1beta1.AdmissionResponse{
					Result: &metav1.Status{
						Message: errorMassage,
					},
				}
			} else {
				ipMap := ip[podNum]
				patchNodeName := make([]patchOperation, 0)
				patchAnnotation := make([]patchOperation, 0)
				for nodeName, ipAddr := range ipMap {
					// 指定Pod的节点
					patchNodeName = mutateNodeName(nodeName)
					// 指定注解
					if ipByte, err := json.Marshal(ipAddr); err != nil {
						errorMassage := fmt.Sprintf("Mutate: json.Marshal ip address '%s' error: %s", ipAddr, err)
						log.Errorf(errorMassage)
						return &v1beta1.AdmissionResponse{
							Result: &metav1.Status{
								Message: errorMassage,
							},
						}
					} else {
						patchAnnotation = addAnnotation(string(ipByte))
					}
				}
				patch = append(patch, patchNodeName...)
				patch = append(patch, patchAnnotation...)
			}
		}
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		errorMassage := fmt.Sprintf("json.Marshal patch: '%s' error: %s", patch, err)
		log.Errorf(errorMassage)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: errorMassage,
			},
		}
	}

	log.Infof("Mutate: AdmissionResponse: patch=%v\n", string(patchBytes))
	// 当前仅支持patchType为JSONPatch的AdmissionResponse
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func validate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var (
		originalPodAnnotations map[string]string
		resourceName           string
		replicas               *int32
	)

	log.Infof("Validate: AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, resourceName, req.UID, req.Operation, req.UserInfo)

	switch req.Kind.Kind {
	case "StatefulSet":
		var sts appsv1.StatefulSet
		if err := json.Unmarshal(req.Object.Raw, &sts); err != nil {
			log.Errorf("Validate: Can't unmarshal raw object to StatefulSet: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName = sts.Name
		// 获取StatefulSet下Pod模板注解，里面应该有此次固定IP的地址
		// 例如: fixed.pod.ip: "[{\"node1\":\"192.168.101.10\"},{\"node2\":\"192.168.102.10\"},{\"node3\":\"192.168.103.10\"}]"
		originalPodAnnotations = sts.Spec.Template.Annotations
		// 获取StatefulSet副本数
		replicas = sts.Spec.Replicas
	default:
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	allowed := true
	var result *metav1.Status
	log.Info("original pod annotations: ", originalPodAnnotations)
	log.Info("required pod annotations: ", RequiredPodAnnotations)
	if v, ok := originalPodAnnotations[RequiredPodAnnotations]; !ok {
		allowed = false
		result = &metav1.Status{
			Reason: metav1.StatusReason(fmt.Sprintf("Validate: Required pod annotation '%s' are not set", RequiredPodAnnotations)),
		}
	} else {
		ip := []map[string][]string{}
		if err := json.Unmarshal([]byte(v), &ip); err != nil {
			allowed = false
			result = &metav1.Status{
				Reason: metav1.StatusReason(fmt.Sprintf("Validate: Unmarshal '%s' value error: %s", RequiredPodAnnotations, err)),
			}
		} else {
			if replicas == nil {
				allowed = false
				result = &metav1.Status{
					Reason: metav1.StatusReason(fmt.Sprintf("Validate: Replicas is empty")),
				}
			} else {
				// 副本数量必须小于所提供的IP数量
				if len(ip) < int(*replicas) {
					allowed = false
					result = &metav1.Status{
						Reason: metav1.StatusReason(fmt.Sprintf("Validate: Replicas count must %d less than or equal to ip count %d", *replicas, len(ip))),
					}
				}
			}
		}
	}
	return &v1beta1.AdmissionResponse{
		Allowed: allowed,
		Result:  result,
	}
}
