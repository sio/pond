package database

import (
	"secrets/crypto"
	"testing"

	"bytes"
)

func BenchmarkSecurePath(b *testing.B) {
	key, err := crypto.LocalKey("../tests/keys/storage")
	if err != nil {
		b.Fatal(err)
	}
	db := &DB{key: key}
	paths := [][]string{
		{"hello", "world"},
		{"very long path", "hello world", "lorem ipsum dolor sit amet", "b9ca718f79cf4a13f6f8baed65c695e7b53ecf912eeeb21e2cd2f7d869c817423c6f3d1a1e31e127c884afaaad47d0b6a5b7e618d4a2d9444c2fd4966f316a49779977da1cb8d6b4576d8853d0dd6b5da64f165e529388cf22301bc4f147f22f547c12dbdedb314a114e92521e12d641fce052e03d7a79fb7c88b14894cf67bd5fde08a7935f07b9a8fca5f96bab5b4c18383e31d7c0a628bb767dcb31a0b62d9d26daa9869b76faf1692efabc80c34bf37826ae381a5c6ffeb4d2fabb27f8e9ddcf849eb42cb594d318f1d69d1e3b8a8d214c22792487a639d187d43733a15733bff61f5922f7116fb8b36917f2c8db3ed9cbfeac413a6a8697f975b438ef5a38b513df9b61053252b787bec786dbfe5adad2a7fae789cab5dc1ce8ce24923d8e269ed120493e3adb5771861a7923c96f3a9ee15198a44f51c62e3da3bce78f6ae1318a652c97dcd51327183b2f68db8b7f1cf240f35871a99bc7a4c39d96b69ee68cacc58a3162164b63220a24584647c48ae29b9d012e5a189ca7553fe1061c05f5ddc1e19b6ce2c77b9cefd9830a869185187a44c7ae12ad03ec277e0711bd44bb4f96651d3e9e9ef4941ca7b97ee4854273d474489e5d15328b1d835465f46c8b3f6f39831191bf59378f876dcdf2e81ca2ccaf01c3dd7b9b6815d32d02f9fbc990ef31cfe4e3310b3ef277327f11f5e47deb3e73e98f915c89efcf7f123a6ddd829dae5cfe98a0c4f75464ca422d8d0e7794b4669fd265c10c3678f24cfcf6b5971cfdeb4a2ca7e470e56bf62c7458bc2dd92b284ef6a3ce57d32721fcb5dab7e75f193bfc693b64df5ed194548de2bef98ca95b33975ac14ede93a19f436f2512841db1ccea250b17a2e89641e3e62a1e6d968ba8b422c68f9904f5d"},
		{},
	}
	outputs := make([][]byte, len(paths))
	for i := 0; i < len(paths); i++ {
		outputs[i], err = db.securePath(paths[i])
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("input #%d: secure path %d bytes\n[%x]", i, len(outputs[i]), outputs[i])
	}
	for i := 0; i < b.N; i++ {
		index := i % len(paths)
		secure, err := db.securePath(paths[index])
		if err != nil {
			b.Fatal(err)
		}
		if len(secure) == 0 {
			b.Fatal("empty secure path")
		}
		if i < len(outputs) {
			outputs[index] = make([]byte, len(secure))
			copy(outputs[index], secure)
		}
		if !bytes.Equal(outputs[index], secure) {
			b.Fatalf("secure path mismatch (iteration=%d, index=%d)", i, index)
		}
	}
}
