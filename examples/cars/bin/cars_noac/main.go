package cars

import (
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/optherium/cckit/examples/cars"
)

func main() {
	cc := cars.NewWithoutAccessControl()
	if err := shim.Start(cc); err != nil {
		fmt.Printf("Error starting Cars chaincode: %s", err)
	}
}
