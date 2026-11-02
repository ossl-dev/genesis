package plugins

import (
	"strings"
	"testing"
)

func TestPowerShellChecksumManifest(t *testing.T) {
	const filename = "deno-x86_64-pc-windows-msvc.zip"
	const publishedHash = "1C40C73193960FCBED9341764D43BBD35E381EDDBCF7D6CBA7C2F3C0CABE96FB"
	const published = "\r\nAlgorithm : SHA256\r\nHash      : " + publishedHash + "\r\nPath      : C:\\a\\deno\\deno\\target\\release\\" + filename + "\r\n\r\n"
	for _, text := range []string{published, "\uFEFF" + published, strings.TrimSpace(published), strings.ReplaceAll(published, "\\", "/")} {
		checksum, err := FileChecksum(text, filename)
		if err != nil || checksum != strings.ToLower(publishedHash) {
			t.Fatalf("checksum=%q error=%v", checksum, err)
		}
	}
	for _, test := range []struct{ name, text string }{
		{"wrong algorithm", strings.Replace(published, "SHA256", "SHA512", 1)},
		{"wrong asset", strings.Replace(published, filename, "deno-aarch64-pc-windows-msvc.zip", 1)},
		{"filename suffix", strings.Replace(published, filename, filename+".other", 1)},
		{"invalid hash", strings.Replace(published, publishedHash, "invalid", 1)},
		{"short hash", strings.Replace(published, publishedHash, publishedHash[:63], 1)},
		{"missing algorithm", strings.Replace(published, "Algorithm : SHA256\r\n", "", 1)},
		{"missing path", strings.Replace(published, "Path      : C:\\a\\deno\\deno\\target\\release\\"+filename+"\r\n", "", 1)},
		{"duplicate hash", strings.Replace(published, "Hash      : ", "Hash      : "+publishedHash+"\r\nHash      : ", 1)},
		{"duplicate path", strings.Replace(published, "Path      : ", "Path      : "+filename+"\r\nPath      : ", 1)},
		{"unknown field", strings.Replace(published, "Algorithm : ", "Unknown : value\r\nAlgorithm : ", 1)},
		{"split records", strings.Replace(published, "Path      : ", "\r\nPath      : ", 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if checksum, err := FileChecksum(test.text, filename); err == nil || checksum != "" {
				t.Fatalf("accepted invalid checksum=%q error=%v", checksum, err)
			}
		})
	}
	other := strings.Replace(published, filename, "other.zip", 1)
	if checksum, err := FileChecksum(other+published, filename); err != nil || checksum != strings.ToLower(publishedHash) {
		t.Fatalf("did not select matching record: %q %v", checksum, err)
	}
}

func TestPlainChecksumsRemainFilenameBound(t *testing.T) {
	for _, text := range []string{testHash, testHash + "  correct.zip\r\n", testHash + " *correct.zip\n"} {
		checksum, err := FileChecksum(text, "correct.zip")
		if err != nil || checksum != testHash {
			t.Fatalf("%q %v", checksum, err)
		}
	}
	for _, file := range []string{"other.zip", "correct.zip.other", "directory/correct.zip", `C:\directory\correct.zip`} {
		if checksum, err := FileChecksum(testHash+"  "+file, "correct.zip"); err == nil || checksum != "" {
			t.Fatalf("accepted mismatched plain manifest %q: %q %v", file, checksum, err)
		}
	}
}
