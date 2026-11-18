// Package geoip déduit une position APPROXIMATIVE d'une adresse IP à partir d'une base LOCALE au format MMDB
// (DB-IP « IP to City Lite », licence CC BY 4.0 : mention « IP Geolocation by DB-IP » obligatoire là où la position
// est affichée). Aucune adresse IP n'est envoyée à un tiers.
package geoip

import (
	"net"

	"github.com/oschwald/maxminddb-golang"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

// Locator lit la base. Sûr en accès concurrent.
type Locator struct{ db *maxminddb.Reader }

// Open ouvre le fichier MMDB.
func Open(path string) (*Locator, error) {
	db, err := maxminddb.Open(path)
	if err != nil {
		return nil, err
	}
	return &Locator{db: db}, nil
}

// Close libère la base.
func (l *Locator) Close() error { return l.db.Close() }

// record : champs communs aux bases « City » de DB-IP et de MaxMind.
type record struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Subdivisions []struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	Location struct {
		Latitude  *float64 `maxminddb:"latitude"`
		Longitude *float64 `maxminddb:"longitude"`
	} `maxminddb:"location"`
}

func name(m map[string]string) string {
	if v := m["fr"]; v != "" {
		return v
	}
	return m["en"]
}

// Locate renvoie nil si l'adresse est invalide, absente de la base ou sans coordonnées.
func (l *Locator) Locate(ip string) *domain.GeoLocation {
	addr := net.ParseIP(ip)
	if addr == nil {
		return nil
	}
	var r record
	if err := l.db.Lookup(addr, &r); err != nil || r.Location.Latitude == nil || r.Location.Longitude == nil {
		return nil
	}
	out := &domain.GeoLocation{City: name(r.City.Names), CountryCode: r.Country.ISOCode, Country: name(r.Country.Names), Latitude: *r.Location.Latitude, Longitude: *r.Location.Longitude}
	if len(r.Subdivisions) > 0 {
		out.Region = name(r.Subdivisions[0].Names)
	}
	return out
}
