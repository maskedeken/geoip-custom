module github.com/JMVoid/ipip2mmdb

go 1.14

require (
	github.com/EvilSuperstars/go-cidrman v0.0.0-20190607145828-28e79e32899a // indirect
	github.com/maxmind/mmdbwriter v0.0.0-20200911190049-91ab57d2e8e9
	github.com/oschwald/geoip2-golang v1.4.0
	github.com/sirupsen/logrus v1.6.0

	v2ray.com/core v4.19.1+incompatible
	google.golang.org/protobuf v1.25.0
)

replace v2ray.com/core => github.com/v2fly/v2ray-core v0.0.0-20201114050607-7cc8b7500687