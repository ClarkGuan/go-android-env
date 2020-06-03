package env

import "testing"

func TestCompareVersion(t *testing.T) {
	if compareVersion("", "") != 0 {
		t.Fatal("compareVersion(\"\", \"\") =", compareVersion("", ""))
	}

	if compareVersion("1", "2") >= 0 {
		t.Fatal("compareVersion(\"1\", \"2\") =", compareVersion("1", "2"))
	}

	if compareVersion("", "1") >= 0 {
		t.Fatal("compareVersion(\"\", \"1\") =", compareVersion("", "1"))
	}

	if compareVersion("2", "") <= 0 {
		t.Fatal("compareVersion(\"2\", \"\") =", compareVersion("2", ""))
	}

	if compareVersion("1.1", "1.2") >= 0 {
		t.Fatal("compareVersion(\"1.1\", \"1.2\") =", compareVersion("1.1", "1.2"))
	}

	if compareVersion("1.1", "1.1.0") != 0 {
		t.Fatal("compareVersion(\"1.1\", \"1.1.0\") =", compareVersion("1.1", "1.1.0"))
	}

	if compareVersion("1.0", "1.0.0") != 0 {
		t.Fatal("compareVersion(\"1.0\", \"1.0.0\") =", compareVersion("1.0", "1.0.0"))
	}

	if compareVersion("1", "1.2.0") >= 0 {
		t.Fatal(" compareVersion(\"1\", \"1.2.0\") =", compareVersion("1", "1.2.0"))
	}
}
