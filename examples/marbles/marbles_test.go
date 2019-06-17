package main

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/optherium/cckit/convert"
	examplecert "github.com/optherium/cckit/examples/cert"
	"github.com/optherium/cckit/extensions/owner"
	"github.com/optherium/cckit/identity"
	"github.com/optherium/cckit/state"
	testcc "github.com/optherium/cckit/testing"
	expectcc "github.com/optherium/cckit/testing/expect"
)

func TestMarbles(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cars Suite")
}

var _ = Describe(`Marbles`, func() {

	//Create chaincode mock
	cc := testcc.NewMockStub(`marbles`, New())

	// load actor certificates from github.com/optherium/cckit/examples/cert
	actors, err := identity.ActorsFromPemFile(
		`SOME_MSP`,
		map[string]string{`operator`: `s7techlab.pem`, `owner1`: `victor-nosov.pem`},
		examplecert.Content)
	if err != nil {
		panic(err)
	}

	BeforeSuite(func() {
		// Init chaincode from operator
		expectcc.ResponseOk(cc.From(actors[`operator`]).Init())
	})

	Describe("Chaincode owner", func() {
		It("Allow everyone to retrieve chaincode owner", func() {

			// get info about chaincode owner
			owner := expectcc.PayloadIs(cc.Invoke(`owner`), &identity.Entry{}).(identity.Entry)
			Expect(owner.GetSubject()).To(Equal(actors[`operator`].GetSubject()))
			Expect(owner.Is(actors[`operator`])).To(BeTrue())
		})
	})

	Describe("Marble owner", func() {

		It("Allow chaincode owner to register marble owner", func() {
			expectcc.ResponseOk(
				// register owner1 certificate as potential marble owner
				cc.From(actors[`operator`]).Invoke(`marbleOwnerRegister`, actors[`owner1`]))
		})

		It("Disallow non chaincode owner to register marble owner", func() {
			expectcc.ResponseError(
				cc.From(actors[`owner1`]).Invoke(`marbleOwnerRegister`, actors[`owner1`]),
				owner.ErrOwnerOnly)
		})

		It("Disallow chaincode owner to register duplicate marble owner", func() {
			expectcc.ResponseError(
				cc.From(actors[`operator`]).Invoke(`marbleOwnerRegister`, actors[`owner1`]),
				state.AlreadyExistsError)
		})

		It("Disallow to pass non SerializedIdentity json", func() {
			expectcc.ResponseError(
				cc.From(actors[`owner1`]).Invoke(`marbleOwnerRegister`, `some weird string`),
				convert.ErrUnableToConvertValueToStruct)
		})

	})
})
