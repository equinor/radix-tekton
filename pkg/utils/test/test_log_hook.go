package test

/* The log hook to type logrus logs, written within the logic under test
Example:
func setupLog(t *testing.T) {
	log.AddHook(test.NewTestLogHook(t, log.DebugLevel).
		ModifyFormatter(func(f *log.TextFormatter) { f.DisableTimestamp = true }))
}

func TestSomething(t *testing.T) {
	setupLog(t)
	//test is here
}
*/

import (
	log "github.com/sirupsen/logrus"
	"testing"
)

type testLogHook struct {
	t         *testing.T
	levels    []log.Level
	formatter *log.TextFormatter
}

func NewTestLogHook(t *testing.T, level log.Level) testLogHook {
	var logLevels []log.Level
	for _, logLevel := range log.AllLevels {
		if logLevel <= level {
			logLevels = append(logLevels, logLevel)
		}
	}
	log.SetLevel(level)
	formatter := log.TextFormatter{
		DisableColors: true,
	}
	log.SetFormatter(&formatter)
	return testLogHook{t: t, levels: logLevels, formatter: &formatter}
}

func (m testLogHook) ModifyFormatter(modify func(*log.TextFormatter)) testLogHook {
	modify(m.formatter)
	return m
}

func (m testLogHook) DisableTimestamp(disable bool) testLogHook {
	m.formatter.DisableTimestamp = disable
	return m
}

func (m testLogHook) Levels() []log.Level {
	return m.levels
}

func (m testLogHook) Fire(entry *log.Entry) error {
	return nil
}
