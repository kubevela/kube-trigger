package cronjob

import "github.com/sirupsen/logrus"

var (
	logger *logrus.Entry
)

func init() {
	logger = logrus.WithField("source", cronJobType)
}
