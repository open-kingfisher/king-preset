package impl

import (
	"github.com/open-kingfisher/king-utils/common/log"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetConfigMap(name, namespace string) error {
	if clientSet, err := K8SClient(); err != nil {
		log.Errorf("get clientSet error: %v", err)
		return err
	} else {
		if _, err := clientSet.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{}); err != nil {
			log.Errorf("get configMap: %s namespace: %s error: %v", name, namespace, err)
			return err
		}
	}
	return nil
}

func CreateConfigMap(name, namespace string, data map[string]string) error {
	if clientSet, err := K8SClient(); err != nil {
		log.Errorf("get clientSet error: %v", err)
		return err
	} else {
		configMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: data,
		}
		if _, err := clientSet.CoreV1().ConfigMaps(namespace).Create(configMap); err != nil {
			log.Errorf("create configMap: %s namespace: %s error: %v", name, namespace, err)
			return err
		}
	}
	return nil
}

func DeleteConfigMap(name, namespace string) error {
	if clientSet, err := K8SClient(); err != nil {
		log.Errorf("get clientSet error: %v", err)
		return err
	} else {
		if err := clientSet.CoreV1().ConfigMaps(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
			log.Errorf("delete configMap: %s namespace: %s error: %v", name, namespace, err)
			return err
		}
	}
	return nil
}
