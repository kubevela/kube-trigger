package cronjob

import (
	"fmt"
	"strings"
	"time"
)

type Config struct {
	Schedule string `json:"schedule"`
	TimeZone string `json:"timeZone"`
}

func (c *Config) String() string {
	if strings.Contains(c.Schedule, "TZ") {
		return c.Schedule
	}

	if c.TimeZone != "" {
		if _, err := time.LoadLocation(c.TimeZone); err != nil {
			return c.Schedule
		}

		return fmt.Sprintf("TZ=%s %s", c.TimeZone, c.Schedule)
	}

	return c.Schedule
}

func formatSchedule(c Config) string {
	if strings.Contains(c.Schedule, "TZ") {
		logger.Warnf("using TZ in schedule is not supported, use timeZone instead")
	}

	return c.String()
}
