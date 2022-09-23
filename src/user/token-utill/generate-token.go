package tokenutill

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

const (
	privateKeyPath = "keys/priv.key"
	publicKeyPath  = "keys/pub.key"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
)

func init() {
	signBytes, err := ioutil.ReadFile(privateKeyPath)
	fatal(err)

	privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	fatal(err)

	verifyBytes, err := ioutil.ReadFile(publicKeyPath)
	fatal(err)

	publicKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
	fatal(err)
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type AccessClaims struct {
	UserName string
	Id       int32
	jwt.RegisteredClaims
}

func GenerateToken(id int32, name string) string {
	claims := AccessClaims{
		UserName: name,
		Id:       id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 12)),
			Issuer:    "simple-jwt-provider",
		},
	}

	t := jwt.NewWithClaims(jwt.GetSigningMethod("RS256"), claims)

	tokenString, err := t.SignedString(privateKey)
	fatal(err)

	return tokenString
}

type JWKS struct {
	Keys []jwk.Key `json:"keys"`
}

func GenerateJWK() []byte {
	key, err := jwk.FromRaw(publicKey)

	if err != nil {
		fmt.Printf("failed to create symmetric key: %s\n", err)
	}

	if _, ok := key.(jwk.RSAPublicKey); !ok {
		fmt.Printf("expected jwk.SymmetricKey, got %T\n", key)
	}

	key.Set(jwk.KeyIDKey, "JWK_ID")

	jwks := JWKS{Keys: []jwk.Key{key}}

	buf, err := json.MarshalIndent(jwks, "", "  ")
	if err != nil {
		fmt.Printf("failed to marshal key into JSON: %s\n", err)
	}

	return buf
}
