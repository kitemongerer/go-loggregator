package conversion_test

import (
	v2 "code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/go-loggregator/v1/conversion"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("HTTP", func() {
	ValueText := func(s string) *v2.Value {
		return &v2.Value{&v2.Value_Text{Text: s}}
	}

	ValueInteger := func(i int64) *v2.Value {
		return &v2.Value{&v2.Value_Integer{Integer: i}}
	}

	Context("given a v2 envelope", func() {
		It("converts to a v1 envelope", func() {
			v2Envelope := &v2.Envelope{
				SourceId: "b3015d69-09cd-476d-aace-ad2d824d5ab7",
				Message: &v2.Envelope_Timer{
					Timer: &v2.Timer{
						Name:  "http",
						Start: 99,
						Stop:  100,
					},
				},
				DeprecatedTags: map[string]*v2.Value{
					"request_id":          ValueText("954f61c4-ac84-44be-9217-cdfa3117fb41"),
					"peer_type":           ValueText("Client"),
					"method":              ValueText("GET"),
					"uri":                 ValueText("/hello-world"),
					"remote_address":      ValueText("10.1.1.0"),
					"user_agent":          ValueText("Mozilla/5.0"),
					"status_code":         ValueInteger(200),
					"content_length":      ValueInteger(1000000),
					"instance_index":      ValueInteger(10),
					"routing_instance_id": ValueText("application-id"),
					"forwarded":           ValueText("6.6.6.6\n8.8.8.8"),
				},
			}
			expectedV1Envelope := &events.Envelope{
				EventType: events.Envelope_HttpStartStop.Enum(),
				HttpStartStop: &events.HttpStartStop{
					StartTimestamp: proto.Int64(99),
					StopTimestamp:  proto.Int64(100),
					RequestId: &events.UUID{
						Low:  proto.Uint64(0xbe4484acc4614f95),
						High: proto.Uint64(0x41fb1731facd1792),
					},
					ApplicationId: &events.UUID{
						Low:  proto.Uint64(0x6d47cd09695d01b3),
						High: proto.Uint64(0xb75a4d822dadceaa),
					},
					PeerType:      events.PeerType_Client.Enum(),
					Method:        events.Method_GET.Enum(),
					Uri:           proto.String("/hello-world"),
					RemoteAddress: proto.String("10.1.1.0"),
					UserAgent:     proto.String("Mozilla/5.0"),
					StatusCode:    proto.Int32(200),
					ContentLength: proto.Int64(1000000),
					InstanceIndex: proto.Int32(10),
					InstanceId:    proto.String("application-id"),
					Forwarded:     []string{"6.6.6.6", "8.8.8.8"},
				},
			}

			envelopes := conversion.ToV1(v2Envelope)
			Expect(len(envelopes)).To(Equal(1))
			converted := envelopes[0]

			_, err := proto.Marshal(converted)
			Expect(err).ToNot(HaveOccurred())
			Expect(*converted).To(MatchFields(IgnoreExtras, Fields{
				"EventType":     Equal(expectedV1Envelope.EventType),
				"HttpStartStop": Equal(expectedV1Envelope.HttpStartStop),
			}))
		})
	})
})
