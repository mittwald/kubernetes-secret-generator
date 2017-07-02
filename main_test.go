package main

import "testing"

func TestGeneratedSecretsHaveCorrectLength(t *testing.T) {
	pwd, err := generateSecret(20)

	t.Log("generated", pwd)

	if err != nil {
		t.Error(err)
	}

	if len(pwd) != 20 {
		t.Error("password length", "expected", 20, "got", len(pwd))
	}
}

func TestGeneratedSecretsAreRandom(t *testing.T) {
	one, errOne := generateSecret(32)
	two, errTwo := generateSecret(32)

	if errOne != nil { t.Error(errOne) }
	if errTwo != nil { t.Error(errTwo) }

	if one == two {
		t.Error("password equality", "got", one)
	}
}

func BenchmarkGenerateSecret(b *testing.B) {
	for i := 0; i < b.N; i ++ {
		_, err := generateSecret(32)
		if err != nil {
			b.Error(err)
		}
	}
}