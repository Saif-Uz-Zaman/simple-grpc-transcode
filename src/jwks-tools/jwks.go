package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	tokenutill "simple-grpc-transcode/src/user/token-utill"
)

func main() {
	JWKS := tokenutill.GenerateJWK()
	dst := &bytes.Buffer{}
	if err := json.Compact(dst, []byte(JWKS)); err != nil {
		panic(err)
	}
	fmt.Println(dst.String())
}
