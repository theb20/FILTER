package geoip

import (
	"os"
	"testing"
)

// Exige la vraie base DB-IP (GEOIP_TEST_FILE, ex. ~/.filterfinder/geoip/dbip-city-lite.mmdb) ; ignoré sinon.
func TestLocateAvecLaBaseDBIP(t *testing.T) {
	path := os.Getenv("GEOIP_TEST_FILE")
	if path == "" {
		t.Skip("GEOIP_TEST_FILE non défini")
	}
	l, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	for _, ip := range []string{"8.8.8.8", "41.202.207.1", "102.67.240.1", "2001:4860:4860::8888"} {
		g := l.Locate(ip)
		if g == nil || g.CountryCode == "" || (g.Latitude == 0 && g.Longitude == 0) {
			t.Errorf("%s : position attendue, obtenu %+v", ip, g)
			continue
		}
		t.Logf("%s → %s, %s, %s (%s) %.3f %.3f", ip, g.City, g.Region, g.Country, g.CountryCode, g.Latitude, g.Longitude)
	}
	for _, ip := range []string{"", "pas-une-ip", "10.0.0.1", "127.0.0.1"} {
		if g := l.Locate(ip); g != nil {
			t.Errorf("%q : aucune position attendue, obtenu %+v", ip, g)
		}
	}
}
