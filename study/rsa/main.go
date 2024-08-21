package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
)

func main() {

	// fmt.Println(Generate())
	fmt.Println(Test())
}

var key = "-----BEGIN PRIVATE KEY-----\nMIICdgIBADANBgkqhkiG9w0BAQEFAASCAmAwggJcAgEAAoGBALUl3AEBxXrDeKcF\nIKx2TYDekg/M/T7JA+8O3eGV14hWlfoAVCXYXN5dzMhHeE36DE7h19SWf3rU7sXR\nsHoQK16rSqhEucV7CLj9M1cTRlShrqhu0drLxgUwqtInmV5kdljAvkTiAKcSmPik\nelB3m3UmnceLdes5+PYEIHursvS/AgMBAAECgYAfLEq16aYgQC8tHtbGlv0zZhng\nmjgia9k/dGF+hpi2n5/ji9bvRFKG+cFZ3eK4GIWxtW+858E8VBRa+oDSIKI1uER2\n20uHCeLM8ISgZ/qEVPGjXDfCovxW1mxc0md1NlLX24axeW8DF6uTkXbaHspzmwwZ\niyufjjghK0ddSmlOMQJBANzXFrRJxf7dRVKscnt1jUoSXdnt6xd7iAawg7ThSbC7\nahM0+YxsxBLALw2ayuuj849l6naZp4NUJglMjq07HBUCQQDR/QIz0Q01ZdCoLRMR\nJM21z/KO07W2xAYRQQIpcK2nPpUwlpK+HlbLZZWVuogY3p55q7TI1mREeYQ5SJYE\nLL6DAkBA52H/2JK9RdDC7HW0/SZqN52nl/n469BdjvEWbwPWUi5puK8C61Bw5lSt\n3el3ebbyVRSkiKInwcpv/zULiozFAkAfRAq16GSNFNHSmJOEM/SlI4c8GO2vftRg\ncUt/HBXfFwRjrae/wwitVDzHhHSLL2ptN1G9rZ5US7uSQ+qCSJ89AkEAhZiz7tsx\nxfrUIXV7SJaXbqsPF8ZFiH6grvFVZvYTH9copZjlyLX20HfOom/j1scR5wWMMPui\niJaJem/jcF77QQ==\n-----END PRIVATE KEY-----\n"

func Test() error {
	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return fmt.Errorf("block error")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey([]byte(block.Bytes))
	if err != nil {
		return err
	}

	fmt.Println(privateKey)

	return nil
}

func Generate() error {
	priKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return errors.WithStack(err)
	}

	pk := x509.MarshalPKCS1PrivateKey(priKey)
	fmt.Println(string(pk))
	b1 := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: pk,
	}

	if err := WriteFile("rsa_private_key.pem", func(w io.Writer) error {
		return pem.Encode(w, &b1)
	}); err != nil {
		return errors.WithStack(err)
	}

	pk2 := x509.MarshalPKCS1PublicKey(&priKey.PublicKey)
	fmt.Println(string(pk2))
	b2 := pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pk2,
	}

	if err := WriteFile("rsa_public_key.pem", func(w io.Writer) error {
		return pem.Encode(w, &b2)
	}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func WriteFile(filename string, fn func(w io.Writer) error) error {
	f, err := os.Create(filename)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()

	return fn(f)
}
