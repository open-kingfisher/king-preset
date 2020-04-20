## Kingfisher king-preset

通过准入控制器实现根据需求对Pod进行预设操作，如：sidecar注入，pod ip地址固定等

## 依赖

- Golang： `Go >= 1.13`
- Kubernetes CNI: Calico >= 3.11.2
- Kubernetes: 开启 ValidatingAdmissionWebhook, MutatingAdmissionWebhook 准入控制器

## 特性

实现`Pod IP地址固定`的需求，对于需要固定IP地址的业务可以使用此模块部署

## 部署

* 使用k8s的CertificateSigningRequest API生成由k8s CA签署的证书 (需要在集群Master上运行），执行后会生成名字为king-preset的Secrets
和名字为king-preset的CSR（CertificateSigningRequest）
>```shell
>./webhook-create-signed-cert.sh
>```
* 生成caBundle（需要在集群Master上运行） 输出的内容填写到 `mutatingwebhook.yaml` 和 `validatingwebhook.yaml` 的caBundle字段上面
>```shell
>./webhook-patch-ca-bundle.sh
>```
* 部署准入控制器webhook
>```shell
>kubectl create -f mutatingwebhook.yaml
>kubectl create -f validatingwebhook.yaml
>kubectl create -f deployment.yaml
>```

## 使用说明

* 项目中deployment/statefulset.yaml为示例部署SatefulSet的YAML文件，需要注意以下几点
    * `只能使用SatefulSet才可以`，不能使用Deployment或者DaemonSet等其他部署方式
    * metadata.labels 和 spec.template.metadata.labels 必须添加 `fix-pod-ip: enabled` 此标签
    * spec.template.metadata.annotations 必须添加如下类型的注解，其中一个Pod将在node01.example.kingfisher.com节点上面并绑定10.10.10.101这个IP，其他Pod以此类推
    >```yaml
    >fix.pod.ip: "[{\"node01.example.kingfisher.com\":[\"10.10.10.101\"]},{\"node002.example.kingfisher.com\":[\"10.10.10.102\"]},{\"node003.example.kingfisher.com\":[\"10.10.10.103\"]}]"
    >```
   * spec.replicas 副本数量必须`小于等于` spec.template.metadata.annotations 这个注释转换成列表后的长度

## Makefile的使用

- 根据需求修改对应的REGISTRY变量，即可修改推送的仓库地址
- 编译成二进制文件： make build
- 生成镜像推送到镜像仓库： make push


