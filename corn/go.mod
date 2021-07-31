module zmyjobs/corn

go 1.16

require (
	github.com/go-redis/redis/v8 v8.11.1
	github.com/huobirdcenter/huobi_golang v0.0.0-20210226095227-8a30a95b6d0d
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lestrrat-go/strftime v1.0.5 // indirect
	github.com/nntaoli-project/goex v1.3.1
	github.com/robfig/cron v1.2.0
	github.com/shopspring/decimal v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/xyths/hs v0.29.1
	golang.org/x/net v0.0.0-20210726213435-c6fcb2dbf985
	gorm.io/driver/mysql v1.1.1
	gorm.io/gorm v1.21.12
)

require zmyjobs/goex v0.0.0

replace zmyjobs/goex => ../goex
