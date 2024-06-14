package conf

import (
	"testing"
)

func TestInit(t *testing.T) {
	Init("./conf.json")
	//fmt.Printf("%v", GConfig)
}
