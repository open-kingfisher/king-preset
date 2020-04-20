module kingfisher/king-preset

go 1.14

require (
	cloud.google.com/go v0.50.0 // indirect
	cloud.google.com/go/bigquery v1.3.0 // indirect
	cloud.google.com/go/pubsub v1.1.0 // indirect
	cloud.google.com/go/storage v1.4.0 // indirect
	dmitri.shuralyov.com/gpu/mtl v0.0.0-20191203043605-d42048ed14fd // indirect
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/creack/pty v1.1.9 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/docker/docker v1.13.1 // indirect
	github.com/envoyproxy/go-control-plane v0.9.1 // indirect
	github.com/gin-contrib/cors v1.3.0 // indirect
	github.com/gin-gonic/gin v1.4.0
	github.com/golang/groupcache v0.0.0-20191227052852-215e87163ea7 // indirect
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/google/pprof v0.0.0-20191218002539-d4f498aebedc // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/kr/pty v1.1.8 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/prometheus/client_golang v1.2.1 // indirect
	github.com/prometheus/client_model v0.1.0 // indirect
	github.com/rogpeppe/go-internal v1.5.1 // indirect
	go.opencensus.io v0.22.2 // indirect
	golang.org/x/crypto v0.0.0-20191227163750-53104e6ec876 // indirect
	golang.org/x/exp v0.0.0-20191227195350-da58074b4299 // indirect
	golang.org/x/image v0.0.0-20191214001246-9130b4cfad52 // indirect
	golang.org/x/mobile v0.0.0-20191210151939-1a1fef82734d // indirect
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553 // indirect
	golang.org/x/oauth2 v0.0.0-20191202225959-858c2ad4c8b6 // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
	golang.org/x/sys v0.0.0-20191228213918-04cbcbbfeed8 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	golang.org/x/tools v0.0.0-20191230220329-2aa90c603ae3 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543 // indirect
	google.golang.org/api v0.15.0 // indirect
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/genproto v0.0.0-20191230161307-f3c370f40bfb // indirect
	google.golang.org/grpc v1.26.0 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.2.7 // indirect
	gotest.tools v2.2.0+incompatible // indirect
	k8s.io/api v0.0.0-20190820101039-d651a1528133
	k8s.io/apimachinery v0.0.0-20190820100750-21ddcbbef9e1
	k8s.io/client-go v11.0.0+incompatible // indirect
	k8s.io/kubernetes v1.13.1 // indirect
	k8s.io/metrics v0.0.0-20190822063148-e60d8d0865eb // indirect
	kingfisher/kf v0.0.0-00010101000000-000000000000
)

replace (
	github.com/docker/docker => github.com/docker/docker v0.7.3-0.20190924004649-91870ed38213
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
	kingfisher/kf => ../kf
)
