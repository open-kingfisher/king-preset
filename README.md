# Kingfisher king-preset
[![License](https://img.shields.io/badge/license-Apache%202-4EB1BA.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)
[![Go Report Card](https://goreportcard.com/badge/github.com/open-kingfisher/king-preset)](https://goreportcard.com/report/github.com/open-kingfisher/king-preset)

通过准入控制器实现根据需求对Kubernetes相关资源进行预设操作，如：sidecar注入，pod ip地址固定等，Service扩展外部IP等

## 依赖

- Golang： `Go >= 1.13`
- Kubernetes CNI: Calico >= 3.11.2
- Kubernetes: 开启 ValidatingAdmissionWebhook, MutatingAdmissionWebhook 准入控制器

## 特性

- 实现`Pod IP地址固定`的需求，对于需要固定IP地址的业务可以使用此模块部署
- 实现`Service添加Kubernetes集群外部IP`的功能
- 实现`Service可以设置备份Pod IP`功能， 可以用于单独Pod测试

## 部署

* 执行deployment目录下面的deployment.sh，会根据deployment_all_in_one.yaml进行部署
>```shell
>./deployment.sh
>```

## 卸载

* 执行deployment目录下面的uninstall.sh
>```shell
>./uninstall.sh
>```

## 使用说明
* Pod IP地址固定
    * 项目中deployment/statefulset.yaml为示例部署SatefulSet的YAML文件，需要注意以下几点
        * `只能使用SatefulSet才可以`，不能使用Deployment或者DaemonSet等其他部署方式
        * metadata.labels 和 spec.template.metadata.labels 必须添加 `fix-pod-ip: enabled` 此标签
        * spec.template.metadata.annotations 必须添加如下类型的注解，其中一个Pod将在node01.example.kingfisher.com节点上面并绑定10.10.10.101这个IP，其他Pod以此类推
        >```yaml
        >fix.pod.ip: "[{\"node01.example.kingfisher.com\":[\"10.10.10.101\"]},{\"node002.example.kingfisher.com\":[\"10.10.10.102\"]},{\"node003.example.kingfisher.com\":[\"10.10.10.103\"]}]"
        >```
       * spec.replicas 副本数量必须`小于等于` spec.template.metadata.annotations 这个注释转换成列表后的长度

* Service支持外部IP
    * 项目中deployment/service.yaml为示例部署service的YAML文件，需要注意以下几点
        * metadata.labels 添加 `endpoint-extend: endpoint-external-ip` 此标签表示开启外部IP添加功能
        * metadata.labels 添加 `externalIP: 192.168.10.115-192.168.10.116-192.168.10.117` 代表想要添加的外部IP地址，使用`-`分隔
        * metadata.labels 添加 `externalPort: 80-8080` 代表想要添加的外部IP的端口，使用`-`分隔`
    * 检查配置是否生效 `kubectl get endpoints external -n kingfisher-system`
    
        >```json
        >{
        >    "apiVersion": "v1",
        >    "kind": "Endpoints",
        >    "metadata": {
        >        "annotations": {
        >            "endpoints.kubernetes.io/last-change-trigger-time": "2020-06-12T02:36:37Z"
        >        },
        >        "creationTimestamp": "2020-06-12T02:36:38Z",
        >        "labels": {
        >            "endpoint-external-ip": "enabled",
        >            "externalIP": "192.168.10.115-192.168.10.116-192.168.10.117",
        >            "externalPort": "80-8080"
        >        },
        >        "name": "external",
        >        "namespace": "kingfisher-system",
        >        "resourceVersion": "56564640",
        >        "selfLink": "/api/v1/namespaces/kingfisher/endpoints/external",
        >        "uid": "e1d85dbd-7bbe-4d59-96c8-21073e00e5ed"
        >    },
        >    "subsets": [
        >        {
        >            "addresses": [
        >                {
        >                    "ip": "192.168.10.115"
        >                },
        >                {
        >                    "ip": "192.168.10.116"
        >                },
        >                {
        >                    "ip": "192.168.10.117"
        >                }
        >            ],
        >            "ports": [
        >                {
        >                    "name": "0",
        >                    "port": 80,
        >                    "protocol": "TCP"
        >                },
        >                {
        >                    "name": "1",
        >                    "port": 8080,
        >                    "protocol": "TCP"
        >                }
        >            ]
        >        }
        >    ]
        >}
    >```
* Service支持备份Pod IP，一旦主IP不可用备份IP将可以使用
    * 项目中deployment/service.yaml为示例部署service的YAML文件，需要注意以下几点
        * metadata.labels 添加 `endpoint-extend: endpoint-backup-ip` 此标签表示开启外部IP添加功能
        * metadata.labels 添加 `externalIP: 192.168.10.115-192.168.10.116-192.168.10.117` 代表想要暂时不启用的备份IP地址，使用`-`分隔，此IP必须是此Service可以正常选择到的Pod IP
    * 检查配置是否生效 `kubectl get endpoints nginx -n kingfisher-system`  可以看到10.244.2.62不在其中
    
        >```json
        >{
        >   "apiVersion": "v1",
        >    "kind": "Endpoints",
        >    "metadata": {
        >        "creationTimestamp": "2020-06-18T02:57:40Z",
        >        "labels": {
        >            "backupIP": "10.244.2.62",
        >            "endpoint-extend": "endpoint-backup-ip"
        >        },
        >        "name": "nginx",
        >        "namespace": "default",
        >       "resourceVersion": "58472728",
        >       "selfLink": "/api/v1/namespaces/default/endpoints/nginx",
        >       "uid": "feb4d17d-ccc2-4042-bfd4-59ad5b45fbf1"
        >    },
        >    "subsets": [
        >       {
        >           "addresses": [
        >               {
        >                   "ip": "10.244.2.63",
        >                   "nodeName": "node03",
        >                   "targetRef": {
        >                       "kind": "Pod",
        >                       "name": "nginx-75b6fcb998-n6vnt",
        >                       "namespace": "default",
        >                       "resourceVersion": "58463150",
        >                       "uid": "30c17ecf-fc17-41a7-8452-e9346305ac7c"
        >                   }
        >               }
        >           ],
        >           "ports": [
        >               {
        >                   "port": 80,
        >                   "protocol": "TCP"
        >               }
        >           ]
        >       }
        >   ]
        >}
    >```

## Makefile的使用

- 根据需求修改对应的REGISTRY变量，即可修改推送的仓库地址
- 编译成二进制文件： make build
- 生成镜像推送到镜像仓库： make push

## 联系我们
- [交流群](https://github.com/open-kingfisher/community/blob/master/contact_us/README.md)