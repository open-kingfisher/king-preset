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
		originalLabels      map[string]string
		originalAnnotations map[string]string
		resourceName        string
		patch               []patchOperation
		pod                 corev1.Pod
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
		resourceName, originalLabels, originalAnnotations = pod.Name, pod.Labels, pod.Annotations
	default:
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	if v, ok := originalLabels[InjectLogSidecarRequiredPodAnnotations]; !ok {
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	} else {
		if v == Enabled {
			// volumes不一定存在
			index := 0
			if pod.Spec.Volumes != nil {
				index = len(pod.Spec.Volumes)
			}
			configMapName := GetDeploymentNameByPod(pod.GetGenerateName())
			patch = append(patch, addLogConfigMapVolume(index, configMapName))
			patch = append(patch, addLogFileDirectoryVolume(index+1))

			// 设置监控脚本执行周期
			metricInterval := "60" // 默认60s
			if interval, ok := originalAnnotations[MetricInterval]; ok {
				metricInterval = interval
			}
			// 业务日志目录
			logFileDirectory := "/var/log"
			if directory, ok := originalAnnotations[LogFileDirectory]; ok {
				logFileDirectory = directory
			}
			// container一定存在，添加日志容器
			patch = append(patch, addLogContainer(len(pod.Spec.Containers), metricInterval, logFileDirectory))

			// 业务容器添加日志目录
			for indexContainer, container := range pod.Spec.Containers {
				indexVolume := 0
				if container.VolumeMounts != nil {
					indexVolume = len(container.VolumeMounts)
				}
				patch = append(patch, addBusinessLogVolume(indexContainer, indexVolume, logFileDirectory))
			}

			// 添加prometheus注解
			pPatch := addPrometheusAnnotation(configMapName)
			patch = append(patch, pPatch...)

		}
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		errorMassage := fmt.Sprintf("json.Marshal patch: '%s' error: %v", patch, err)
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
		originalLabels         map[string]string
		originalPodAnnotations map[string]string
		resourceName           string
	)

	log.Infof("Validate: AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, resourceName, req.UID, req.Operation, req.UserInfo)

	// 删除的时候也要删除ConfigMap
	if req.Operation == v1beta1.Delete {
		if err := DeleteConfigMap(req.Name, req.Namespace); err != nil {
			log.Errorf("delete configMap error: name=%s, namespace:%s %v", req.Name, req.Namespace, err)
		}
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	switch req.Kind.Kind {
	case "Deployment":
		var dep appsv1.Deployment
		if err := json.Unmarshal(req.Object.Raw, &dep); err != nil {
			log.Errorf("Validate: Can't unmarshal raw object to Deployment: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName = dep.Name
		originalLabels = dep.Labels
		originalPodAnnotations = dep.Spec.Template.Annotations
		log.Infof("Validate: Deployment for %v", dep)
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
		originalLabels = sts.Labels
		originalPodAnnotations = sts.Spec.Template.Annotations
		log.Infof("Validate: StatefulSet for %v", sts)
	default:
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	log.Info("original annotations: ", originalLabels)

	value, ok := originalLabels[InjectLogSidecarRequiredPodAnnotations]
	if !ok {
		// log-injection不存在要删除ConfigMap
		// 在用户去除log-injection标签进行提交的时候，还会过一下这个准入控制器，用于
		// 删除ConfigMap，再次提交将不会过这个准入控制器
		if err := DeleteConfigMap(resourceName, req.Namespace); err != nil {
			log.Errorf("delete configMap error: name=%s, namespace:%s %v", resourceName, req.Namespace, err)
		}
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}
	if value == Enabled {
		if directory, ok := originalPodAnnotations[LogFileDirectory]; !ok {
			return &v1beta1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Reason: metav1.StatusReason(fmt.Sprintf("Validate: Required spec.template.annotation '%s' are not set", LogFileDirectory)),
				},
			}
		} else if directory == "" {
			return &v1beta1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Reason: metav1.StatusReason(fmt.Sprintf("Validate: Required spec.template.annotation '%s' are not empty", LogFileDirectory)),
				},
			}
		}
		if err := GetConfigMap(resourceName, req.Namespace); err != nil { // configMap不存在的情况下创建对应configMap
			// 创建configMap
			data := map[string]string{
				LogMetricsShell: "#!/bin/sh\n#输出文件必须放到/tmp/目录下面并且以.prom结尾\necho 'qps{Business=\"example\",product=\"example_product\"} 90' > /tmp/example.prom",
			}
			log.Info("stating create configMap")
			if err := CreateConfigMap(resourceName, req.Namespace, data); err != nil {
				log.Errorf("create configMap error: name=%s, namespace:%s %v", resourceName, req.Namespace, err)
				return &v1beta1.AdmissionResponse{
					Allowed: false,
					Result: &metav1.Status{
						Reason: metav1.StatusReason(fmt.Sprintf("Validate: Create ConfigMap '%s' is failure: '%s'", resourceName, err)),
					},
				}
			}
		}
		// 删除的时候也要删除ConfigMap
		if req.Operation == v1beta1.Delete {
			if err := DeleteConfigMap(resourceName, req.Namespace); err != nil {
				log.Errorf("delete configMap error: name=%s, namespace:%s %v", resourceName, req.Namespace, err)
			}
		}
	} else {
		// log-injection值不为enabled也要删除ConfigMap
		// 在用户去除log-injection标签进行提交的时候，还会过一下这个准入控制器，用于
		// 删除ConfigMap，再次提交将不会过这个准入控制器
		if err := DeleteConfigMap(resourceName, req.Namespace); err != nil {
			log.Errorf("delete configMap error: name=%s, namespace:%s %v", resourceName, req.Namespace, err)
		}
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	return &v1beta1.AdmissionResponse{
		Allowed: true,
	}
}
