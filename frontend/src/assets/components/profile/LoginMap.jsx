import { useEffect, useRef } from "react";
import L from "leaflet";
import "leaflet/dist/leaflet.css";

// Carte des lieux de connexion (fond OpenStreetMap). Chaque lieu est un cercle : la position est APPROXIMATIVE
// (déduite de l'adresse IP, souvent la ville de l'opérateur), jamais un point exact. Le lieu choisi est mis en avant.
export default function LoginMap({ points, selected, label }) {
  const box = useRef(null);
  const map = useRef(null);
  const layer = useRef(null);

  useEffect(() => {
    if (!box.current || map.current) return undefined;
    map.current = L.map(box.current, { scrollWheelZoom: false, attributionControl: true }).setView([7.54, -5.55], 5);
    L.tileLayer("https://tile.openstreetmap.org/{z}/{x}/{y}.png", {
      maxZoom: 12,
      attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> · <a href="https://db-ip.com">IP Geolocation by DB-IP</a>',
    }).addTo(map.current);
    layer.current = L.layerGroup().addTo(map.current);
    return () => {
      map.current?.remove();
      map.current = null;
    };
  }, []);

  useEffect(() => {
    const m = map.current;
    if (!m || !layer.current) return;
    layer.current.clearLayers();
    const valid = points.filter((p) => p.location);
    for (const p of valid) {
      const on = selected && p.id === selected.id;
      L.circle([p.location.latitude, p.location.longitude], {
        radius: on ? 12000 : 8000,
        color: on ? "#e0a900" : "#232F3E",
        weight: on ? 3 : 1.5,
        fillColor: on ? "#F7BE14" : "#0275C8",
        fillOpacity: on ? 0.45 : 0.18,
      }).addTo(layer.current);
    }
    if (selected?.location) {
      m.setView([selected.location.latitude, selected.location.longitude], 9, { animate: true });
    } else if (valid.length > 0) {
      m.fitBounds(L.latLngBounds(valid.map((p) => [p.location.latitude, p.location.longitude])).pad(0.6), { maxZoom: 9 });
    }
  }, [points, selected]);

  return <div ref={box} role="img" aria-label={label} className="h-[240px] w-full overflow-hidden rounded-[8px] bg-gray-100 md:h-[300px]" />;
}
