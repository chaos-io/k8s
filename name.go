package k8s

import (
	"crypto"
	"math/big"
)

var b26Alphabet = []byte("abcdefghijkmnopqrstuvwxyz-")

func base26Encode(input []byte) []byte {
	var result []byte
	x := big.NewInt(0).SetBytes(input)
	base := big.NewInt(int64(len(b26Alphabet)))
	zero := big.NewInt(0)
	mod := &big.Int{}
	for x.Cmp(zero) != 0 {
		x.DivMod(x, base, mod) // 对x取余数
		result = append(result, b26Alphabet[mod.Int64()])
	}
	return result
}

func GetEncodeString(id string) string {
	if len(id) > 0 {
		bytes := crypto.MD5.New().Sum([]byte(id))
		return string(base26Encode(bytes[0:10]))
	}
	return ""
}
