//(C) Copyright [2022] Hewlett Packard Enterprise Development LP
//
//Licensed under the Apache License, Version 2.0 (the "License"); you may
//not use this file except in compliance with the License. You may obtain
//a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//License for the specific language governing permissions and limitations
// under the License.

package logs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// LogFormat is custom type created for the log formats supported by ODIM
type LogFormat uint32

// log formats supported by ODIM
const (
	SyslogFormat LogFormat = iota
	JSONFormat
)

var priorityLogFields = []string{
	"host",
	"threadname",
	"procid",
	"messageid",
}

const (
	actionid        = "actionid"
	actionname      = "actionname"
	defaultThreadId = "0"
	processname     = "processname"
	threadid        = "threadid"
	threadname      = "threadname"
	transactionid   = "transactionid"
)

var syslogPriorityNumerics = map[string]int8{
	"panic":   8,
	"fatal":   9,
	"error":   11,
	"warn":    12,
	"warning": 12,
	"info":    14,
	"debug":   15,
	"trace":   15,
}

var logFields = map[string][]string{
	"account": {
		"user",
		"roleID",
	},
	"request": {
		"method",
		"resource",
		"requestBody",
	},
	"response": {
		"responseCode",
	},
}

// SysLogFormatter implements logrus Format interface. It provides a formatter for odim in syslog format
type SysLogFormatter struct{}

var Log *logrus.Entry

func init() {
	Log = logrus.NewEntry(logrus.New())
}

// Adorn adds the fields to Log variable
func Adorn(m logrus.Fields) {
	Log = Log.WithFields(m)
}

// LogWithFields add fields to log
func LogWithFields(ctx context.Context) *logrus.Entry {
	transID := ctx.Value("transactionid")
	processName := ctx.Value("processname")
	threadID := ctx.Value("threadid")
	actionName := ctx.Value("actionname")
	threadName := ctx.Value("threadname")
	actionID := ctx.Value("actionid")
	fields := logrus.Fields{
		"processname":   processName,
		"transactionid": transID,
		"actionid":      actionID,
		"actionname":    actionName,
		"threadid":      threadID,
		"threadname":    threadName,
		"messageid":     actionName,
	}
	return Log.WithFields(fields)
}

// Format renders a log in syslog format
func (f *SysLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	level := entry.Level.String()
	priorityNumber := findSysLogPriorityNumeric(level)
	sysLogMsg := fmt.Sprintf("<%d>%s %s ", priorityNumber, "1", entry.Time.UTC().Format(time.RFC3339))
	sysLogMsg = formatPriorityFields(entry, sysLogMsg)
	sysLogMsg = formatStructuredFields(entry, sysLogMsg)
	for k, v := range logFields {
		if accountLog, present := formatSyslog(k, v, entry); present {
			sysLogMsg = fmt.Sprintf("%s %s", sysLogMsg, accountLog)
		}
	}

	sysLogMsg = fmt.Sprintf("%s %s", sysLogMsg, entry.Message)
	return append([]byte(sysLogMsg), '\n'), nil
}

func findSysLogPriorityNumeric(level string) int8 {
	return syslogPriorityNumerics[level]
}

// formatStructuredFields is used to create structured fields for log
func formatStructuredFields(entry *logrus.Entry, msg string) string {
	var transID, processName, actionID, actionName, threadID, threadName string
	if val, ok := entry.Data["processname"]; ok {
		if val != nil {
			processName = val.(string)
		}
	}
	if val, ok := entry.Data["transactionid"]; ok {
		if val != nil {
			transID = val.(string)
		}
	}
	if val, ok := entry.Data["actionid"]; ok {
		if val != nil {
			actionID = val.(string)
		}
	}
	if val, ok := entry.Data["actionname"]; ok {
		if val != nil {
			actionName = val.(string)
		}
	}
	if val, ok := entry.Data["threadid"]; ok {
		if val != nil {
			threadID = val.(string)
		}
	}
	if val, ok := entry.Data["threadname"]; ok {
		if val != nil {
			threadName = val.(string)
		}
	}
	if transID != "" {
		msg = fmt.Sprintf("%s [process@1 processName=\"%s\" transactionID=\"%s\" actionID=\"%s\" actionName=\"%s\" threadID=\"%s\" threadName=\"%s\"]", msg, processName, transID, actionID, actionName, threadID, threadName)
	}
	return msg
}
func formatPriorityFields(entry *logrus.Entry, msg string) string {
	present := true
	for _, v := range priorityLogFields {
		if val, ok := entry.Data[v]; ok {
			present = false
			msg = fmt.Sprintf("%s %v ", msg, val)
		}
	}
	if !present {
		msg = msg[:len(msg)-1]
	}
	return msg
}

func formatSyslog(logType string, logFields []string, entry *logrus.Entry) (string, bool) {
	isPresent := false
	msg := fmt.Sprintf("[%s@1 ", logType)
	for _, v := range logFields {
		if val, ok := entry.Data[v]; ok {
			isPresent = true
			msg = fmt.Sprintf("%s %s=\"%v\" ", msg, v, val)
		}
	}
	msg = msg[:len(msg)-1]
	return fmt.Sprintf("%s]", msg), isPresent
}

// SetLogLevel sets the given input as log level
func SetLogLevel(level string) {
	switch strings.ToLower(level) {
	case "panic":
		Log.Logger.SetLevel(logrus.PanicLevel)
	case "fatal":
		Log.Logger.SetLevel(logrus.FatalLevel)
	case "error":
		Log.Logger.SetLevel(logrus.ErrorLevel)
	case "warn":
		Log.Logger.SetLevel(logrus.WarnLevel)
	case "info":
		Log.Logger.SetLevel(logrus.InfoLevel)
	case "debug":
		Log.Logger.SetLevel(logrus.DebugLevel)
	case "trace":
		Log.Logger.SetLevel(logrus.TraceLevel)
	default:
		Log.Logger.SetLevel(logrus.WarnLevel)
		Log.Warn("Configured invalid log level. Setting warn as the log level.")
	}
}

// SetLogFormat sets the giving input as logging format
func SetLogFormat(format LogFormat) {
	switch format {
	case JSONFormat:
		Log.Logger.SetFormatter(&logrus.JSONFormatter{})
	case SyslogFormat:
		Log.Logger.SetFormatter(&SysLogFormatter{})
	default:
		Log.Logger.SetFormatter(&SysLogFormatter{})
		Log.Warn("Configured invalid log format. Setting syslog as the log format.")
	}
}

// Convert the log format to a string.
func (format LogFormat) String() string {
	if b, err := format.MarshalText(); err == nil {
		return string(b)
	}
	return "unknown_log_format"
}

// ParseLogFormat takes a string level and returns the log format.
func ParseLogFormat(format string) (LogFormat, error) {
	switch strings.ToLower(format) {
	case "syslog":
		return SyslogFormat, nil
	case "json":
		return JSONFormat, nil
	}

	var lf LogFormat
	return lf, fmt.Errorf("invalid log format : %s", format)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (format *LogFormat) UnmarshalText(text []byte) error {
	l, err := ParseLogFormat(string(text))
	if err != nil {
		return err
	}

	*format = l
	return nil
}

// MarshalText will validate the log format and return the corresponding string
func (format LogFormat) MarshalText() ([]byte, error) {
	switch format {
	case SyslogFormat:
		return []byte("syslog"), nil
	case JSONFormat:
		return []byte("json"), nil
	}

	return nil, fmt.Errorf("invalid log format %d", format)
}

// CreateContextForLogging is to get the context with all the parameters those needed for log
func CreateContextForLogging(ctx context.Context, transactionId string, threadName string, actionID string, actionName string, podName string) context.Context {
	// Add Action ID and Action Name in logs
	ctx = context.WithValue(ctx, actionid, actionID)
	ctx = context.WithValue(ctx, actionname, actionName)

	// Add values in context (TransactionID, ThreadName, ThreadID)
	ctx = context.WithValue(ctx, transactionid, transactionId)
	ctx = context.WithValue(ctx, threadname, threadName)
	ctx = context.WithValue(ctx, threadid, defaultThreadId)
	ctx = context.WithValue(ctx, processname, podName)

	return ctx
}
