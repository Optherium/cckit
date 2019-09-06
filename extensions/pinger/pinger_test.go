package pinger

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	examplecert "github.com/optherium/cckit/examples/cert"
	"github.com/optherium/cckit/identity"
	"github.com/optherium/cckit/router"
	testcc "github.com/optherium/cckit/testing"
	expectcc "github.com/optherium/cckit/testing/expect"
)

func TestPinger(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pinger suite")
}

func New() *router.Chaincode {
	r := router.New(`pingable`).
		Init(router.EmptyContextHandler).
		Invoke(FuncPing, Ping)
	return router.NewChaincode(r)
}

var _ = Describe(`Pinger`, func() {

	//Create chaincode mock
	cc := testcc.NewMockStub(`cars`, New())
	invokerIdentity, err := identity.FromFile(`SOME_MSP`, `s7techlab.pem`, examplecert.Content)
	if err != nil {
		panic(err)
	}

	Describe("Pinger", func() {

		It("Allow anynone to invoke ping method", func() {
			//invoke chaincode method from authority actor
			pingInfo := expectcc.PayloadIs(cc.From(invokerIdentity).Invoke(FuncPing), &PingInfo{}).(PingInfo)
			Expect(pingInfo.InvokerID).To(Equal(invokerIdentity.GetID()))
			Expect(pingInfo.InvokerCert).To(Equal(invokerIdentity.GetPEM()))

			//check that we have event
			pingInfoEvent := expectcc.EventPayloadIs(cc.ChaincodeEvent, &PingInfo{}).(PingInfo)
			Expect(pingInfoEvent.InvokerID).To(Equal(invokerIdentity.GetID()))
			Expect(pingInfoEvent.InvokerCert).To(Equal(invokerIdentity.GetPEM()))
		})

	})
})
