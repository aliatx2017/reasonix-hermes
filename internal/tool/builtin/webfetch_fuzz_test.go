package builtin

import (
	"net"
	"net/url"
	"testing"
)

// FuzzBlockedFetchIP verifies that blockedFetchIP never panics on arbitrary
// IP byte sequences. It accepts 4-byte (IPv4) and 16-byte (IPv6) inputs and
// constructs net.IP values from them.
func FuzzBlockedFetchIP(f *testing.F) {
	// IPv4 seeds: loopback, private, link-local, unspecified, public
	f.Add([]byte{127, 0, 0, 1})
	f.Add([]byte{192, 168, 1, 1})
	f.Add([]byte{10, 0, 0, 1})
	f.Add([]byte{172, 16, 0, 1})
	f.Add([]byte{169, 254, 169, 254})
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{8, 8, 8, 8})
	f.Add([]byte{100, 64, 0, 1})
	// IPv6 seeds (16 bytes)
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})       // ::1
	f.Add([]byte{0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}) // fe80::1

	f.Fuzz(func(t *testing.T, ipBytes []byte) {
		ip := net.IP(ipBytes)
		// Must not panic for any input including non-standard lengths.
		_ = blockedFetchIP(ip)
	})
}

// FuzzWebFetchURLValidation verifies that url.Parse + scheme validation used
// in Execute never panics on arbitrary URL strings.
func FuzzWebFetchURLValidation(f *testing.F) {
	f.Add("https://example.com/path?q=1")
	f.Add("http://example.com")
	f.Add("ftp://example.com")
	f.Add("not-a-url")
	f.Add("")
	f.Add("://missing-scheme")
	f.Add("https://[::1]/api")
	f.Add("https://169.254.169.254/latest/meta-data/")
	f.Add("https://user:pass@host:8080/path#fragment")
	f.Add("\x00\x01https://malformed")

	f.Fuzz(func(t *testing.T, rawURL string) {
		u, err := url.Parse(rawURL)
		if err != nil {
			return
		}
		// Replicate the scheme validation from Execute.
		_ = (u.Scheme != "http" && u.Scheme != "https")
	})
}
