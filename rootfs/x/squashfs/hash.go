// Reading verity hash table:
// <https://gitlab.com/cryptsetup/cryptsetup/-/wikis/DMVerity>
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"
)

const (
	root = `6be2e6c51894c7e02048927a010e4a032b37e0fd79bd203b6b6d757f57fff70b`
	salt = `b58a4975ffd68b396aec8e0c945becdc8a2881049972ebcc2c103c07d0a7ec83`
	top  = `
		0181 7ded 3f5d bf6d ec67 3864 21b6 3e5d
		7580 382a e0b8 eebd ea46 1328 0f74 fb0a
		9a0f 5465 912e ffce 7f41 251d 6fc5 5300
		4615 aa15 dd1f ce47 3b0b 153c 70fa 2eaf
		23b6 04a6 ba09 f240 4615 4d86 bb01 aa1d
		7b93 bcd6 018a 839a 210f 0cba 277f 04dd
		30b3 fe20 d884 1591 cafe 3e02 5a0d 3141
		c7df a2d3 fc9b 785a bbae 0086 9136 82e5`
	second_level_last_block = `
		416f 288f 3d9d 32a9 42e4 73a0 b980 1447
		6ea1 df01 7e7c d08e 8d58 f923 6f81 6519
		d592 c55f e1ca ea57 edf2 2826 84dc 7558
		bca7 c255 9344 a68e d423 43d5 56c6 a731
		5bbf 6d05 846a f2dc 97ce 72f4 6f1d 6a45
		13e8 d038 1669 8baa 1327 9a7c 0f8c 47b4
		9952 9951 170c 7611 5cf4 7777 1919 a460
		6e4e ce6b ac5e 8324 ec87 937e 9b44 ecc7
		3da0 cc3a 8060 f267 f626 e8e8 d200 f685
		2fdf 5829 7ef1 81eb 81d4 8a49 a206 a627
		bcad bb81 5be7 a60c e453 4369 f48d a1d1
		de37 1812 b1be 00b9 4432 7973 2c6b 148c
		f96b 198a 63f1 5312 f4a1 be76 a3bf b27f
		62ff d4ee 2c07 0528 767b 971d 8405 7a5d
		a8f9 a6f1 6564 e9b8 c8c9 f295 ab7a 9ce4
		f379 dc14 d877 9461 7b13 0084 a744 f1a2
		ce56 c715 be08 a7cd e13e bbd1 6ceb 63ee
		ae69 9e57 fb9f 3ae8 7f94 a168 6348 1119
		9254 4c88 7482 f2c9 ca4e cabb 71e5 93dc
		eb73 5662 e9e9 3ed4 a0a1 d578 5f2c 7f02
		62bb 8df6 dac2 8425 0680 e0f9 b46b 330d
		436e 9287 8f83 7466 0aa6 017e f0d6 71b6
		9ec5 e50f e1a2 7f94 3c12 dba7 2c99 b1bd
		3b6c b658 413a e3b4 5f36 81a8 7f38 46c3
		7768 227b afc2 49da 1d44 21e2 c328 c20b
		c22c 60c7 f1d4 9674 8263 586d e4b9 d2e0
		4258 89ee 88b3 d7f8 7fc5 1af0 7766 82b3
		91f0 b7d4 1593 b9dc ff9f ec1b b6b7 74b5
		59f0 2041 67db 24d1 4ba7 d267 307b 6c42
		9558 b097 2175 9d91 2e9b 2e70 5a0e 4913
		c715 dc40 c01e 56df b272 a3a2 6b97 27a7
		0327 17a5 4aec db18 7d20 2eea 795b f53e
		9850 1648 ea6f a68d 598c 0478 1460 9901
		e695 3346 da08 c3e1 9f2b d222 8417 5da4`
)

func main() {
	fmt.Printf("Recalculated root hash: %s\n", verity(decode(top)))
	fmt.Printf("Expected root hash:     %s\n", root)
	fmt.Println("")
	fmt.Printf("Last hash in second level: %s\n", verity(decode(second_level_last_block)))
	top := stripSpace(top)
	fmt.Printf("Expected:                  %s\n", top[len(top)-64:])
}

func verity(data []byte) string {
	salt := decode(salt)
	var hashblock [4096]byte
	copy(hashblock[:], data)

	hash := sha256.New()
	_, _ = hash.Write(salt)
	_, _ = hash.Write(hashblock[:])
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func stripSpace(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

func decode(hexademical string) []byte {
	data, err := hex.DecodeString(stripSpace(hexademical))
	if err != nil {
		panic(err)
	}
	return data
}
