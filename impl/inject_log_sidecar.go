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
	"net/http"
)

func MutateInjectLogSidecar(c *gin.Context) {
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
		admissionResponse = MutateLog(&ar)
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

func ValidateInjectLogSidecar(c *gin.Context) {
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
		admissionResponse = ValidateLog(&ar)
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

func MutateLog(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var (
		originalLables map[string]string
		resourceName   string
		patch          []patchOperation
		pod            corev1.Pod
	)

	log.Infof("Mutate: AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, resourceName, req.UID, req.Operation, req.UserInfo)

	switch req.Kind.Kind {
	case "Pod":
		if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
			log.Errorf("Mutate: Can't unmarshal raw object to pod: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		log.Infof("Mutate: AdmissionReview Resource: %+v", pod)
		resourceName, originalLables = pod.Name, pod.Labels
	default:
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	if v, ok := originalLables[InjectLogSidecarRequiredPodAnnotations]; !ok {
		log.Errorf("Required pod label '%s' are not set", InjectLogSidecarRequiredPodAnnotations)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: fmt.Sprintf("Mutate: Required pod label '%s' are not set", InjectLogSidecarRequiredPodAnnotations),
			},
		}
	} else {
		if v == Enabled {
			// volumes不一定存在
			index := 0
			if pod.Spec.Volumes != nil {
				index = len(pod.Spec.Volumes)
			}
			patch = append(patch, addLogVolume(index))
			// container一定存在
			patch = append(patch, addLogContainer(len(pod.Spec.Containers)))

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
		PatchType: func() *v1beta1.PatchType { // 不用单独生成变量，直接命名函数返回
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func ValidateLog(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
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
						Reason: metav1.StatusReason(fmt.Sprintf("Validate: Replicas count %d less than or equal to ip count %d", *replicas, len(ip))),
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
