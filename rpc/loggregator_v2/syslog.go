package loggregator_v2

import (
	"bytes"
	fmt "fmt"
	"strconv"
	"time"

	"code.cloudfoundry.org/rfc5424"
)

// 47450 is the registered enterprise ID for the Cloud Foundry Foundation.
// See: https://www.iana.org/assignments/enterprise-numbers/enterprise-numbers
const (
	gaugeStructuredDataID   = "gauge@47450"
	counterStructuredDataID = "counter@47450"
	timerStructuredDataID   = "timer@47450"
)

type syslogConfig struct {
	hostname  string
	appName   string
	processID string
}

// SyslogOption configures the behavior of Envelope.Syslog.
type SyslogOption func(*syslogConfig)

// WithSyslogHostname changes the hostname of the resulting syslog messages.
func WithSyslogHostname(hostname string) SyslogOption {
	return func(c *syslogConfig) {
		c.hostname = hostname
	}
}

// WithSyslogAppName changes the app name of the resulting syslog messages.
func WithSyslogAppName(appName string) SyslogOption {
	return func(c *syslogConfig) {
		c.appName = appName
	}
}

// WithSyslogProcessID changes the process id of the resulting syslog messages.
func WithSyslogProcessID(processID string) SyslogOption {
	return func(c *syslogConfig) {
		c.processID = processID
	}
}

// Syslog converts an envelope into syslog messages. Typically, this will be
// one to one (envelope to syslog) but for certain envelope type such as
// gauges the single envelope maps onto multiple syslog messages (one per gauge
// metric).
func (m *Envelope) Syslog(opts ...SyslogOption) ([][]byte, error) {
	c := &syslogConfig{
		processID: m.InstanceId,
		appName:   m.SourceId,
	}

	for _, o := range opts {
		o(c)
	}

	priority, err := generatePriority(m)
	if err != nil {
		return nil, err
	}
	timestamp := time.Unix(0, m.GetTimestamp()).UTC()

	switch m.GetMessage().(type) {
	case *Envelope_Log:
		msg := basicSyslogMessage(c, timestamp, priority)
		msg.Message = appendNewline(removeNulls(m.GetLog().Payload))
		d, err := msg.MarshalBinary()
		return [][]byte{d}, err
	case *Envelope_Gauge:
		metrics := m.GetGauge().GetMetrics()
		messages := make([][]byte, 0, len(metrics))
		for name, g := range metrics {
			msg := basicSyslogMessage(c, timestamp, priority)
			msg.StructuredData = []rfc5424.StructuredData{
				{
					ID: gaugeStructuredDataID,
					Parameters: []rfc5424.SDParam{
						{
							Name:  "name",
							Value: name,
						},
						{
							Name:  "value",
							Value: strconv.FormatFloat(g.GetValue(), 'g', -1, 64),
						},
						{
							Name:  "unit",
							Value: g.GetUnit(),
						},
					},
				},
			}
			d, err := msg.MarshalBinary()
			if err != nil {
				return nil, err
			}
			messages = append(messages, d)
		}
		return messages, nil
	case *Envelope_Counter:
		msg := basicSyslogMessage(c, timestamp, priority)
		msg.StructuredData = []rfc5424.StructuredData{
			{
				ID: counterStructuredDataID,
				Parameters: []rfc5424.SDParam{
					{
						Name:  "name",
						Value: m.GetCounter().GetName(),
					},
					{
						Name:  "total",
						Value: fmt.Sprint(m.GetCounter().GetTotal()),
					},
					{
						Name:  "delta",
						Value: fmt.Sprint(m.GetCounter().GetDelta()),
					},
				},
			},
		}
		d, err := msg.MarshalBinary()
		return [][]byte{d}, err
	case *Envelope_Event:
		msg := basicSyslogMessage(c, timestamp, priority)
		msg.Message = []byte(fmt.Sprintf(
			"%s: %s\n",
			m.GetEvent().GetTitle(),
			m.GetEvent().GetBody(),
		))
		d, err := msg.MarshalBinary()
		return [][]byte{d}, err
	case *Envelope_Timer:
		msg := basicSyslogMessage(c, timestamp, priority)
		msg.StructuredData = []rfc5424.StructuredData{
			{
				ID: timerStructuredDataID,
				Parameters: []rfc5424.SDParam{
					{
						Name:  "name",
						Value: m.GetTimer().GetName(),
					},
					{
						Name:  "start",
						Value: fmt.Sprint(m.GetTimer().GetStart()),
					},
					{
						Name:  "stop",
						Value: fmt.Sprint(m.GetTimer().GetStop()),
					},
				},
			},
		}
		d, err := msg.MarshalBinary()
		return [][]byte{d}, err
	default:
		msg := basicSyslogMessage(c, timestamp, priority)
		d, err := msg.MarshalBinary()
		return [][]byte{d}, err
	}
}

func basicSyslogMessage(
	c *syslogConfig,
	timestamp time.Time,
	priority rfc5424.Priority,
) rfc5424.Message {
	return rfc5424.Message{
		Priority:  priority,
		Timestamp: timestamp,
		Hostname:  c.hostname,
		AppName:   c.appName,
		ProcessID: c.processID,
		Message:   []byte("\n"),
	}
}

func generatePriority(e *Envelope) (rfc5424.Priority, error) {
	if l := e.GetLog(); l != nil {
		switch l.Type {
		case Log_OUT:
			return rfc5424.Info + rfc5424.User, nil
		case Log_ERR:
			return rfc5424.Error + rfc5424.User, nil
		default:
			return 0, fmt.Errorf("invalid log type: %s", l.Type)
		}
	}
	return rfc5424.Info + rfc5424.User, nil
}

func removeNulls(msg []byte) []byte {
	return bytes.Replace(msg, []byte{0}, nil, -1)
}

func appendNewline(msg []byte) []byte {
	if !bytes.HasSuffix(msg, []byte("\n")) {
		msg = append(msg, byte('\n'))
	}
	return msg
}