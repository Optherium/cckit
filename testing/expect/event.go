package expect

import (
	"github.com/hyperledger/fabric/protos/peer"
	g "github.com/onsi/gomega"
	"github.com/optherium/cckit/convert"
)

func EventIs(event *peer.ChaincodeEvent, expectName string, expectPayload interface{}) {
	g.Expect(event.EventName).To(g.Equal(expectName), `event name not match`)

	EventPayloadIs(event, expectPayload)
}

// EventPayloadIs expects peer.ChaincodeEvent payload can be marshaled to
// target interface{} and returns converted value
func EventPayloadIs(event *peer.ChaincodeEvent, target interface{}) interface{} {
	g.Expect(event).NotTo(g.BeNil())
	data, err := convert.FromBytes(event.Payload, target)
	description := ``
	if err != nil {
		description = err.Error()
	}
	g.Expect(err).To(g.BeNil(), description)
	return data
}
