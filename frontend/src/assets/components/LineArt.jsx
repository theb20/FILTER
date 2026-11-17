// Dessin technique d'une cartouche filtrante, en trait fin (filigrane / illustration).
export default function LineArt({ className = "", stroke = "#ffffff", opacity = 1 }) {
  const pleats = Array.from({ length: 14 }, (_, i) => 70 + i * 16);
  return (
    <svg
      viewBox="0 0 400 520"
      fill="none"
      stroke={stroke}
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      opacity={opacity}
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      {/* embouts */}
      <ellipse cx="200" cy="60" rx="110" ry="26" />
      <ellipse cx="200" cy="60" rx="70" ry="15" />
      <path d="M90 60v400M310 60v400" />
      <ellipse cx="200" cy="460" rx="110" ry="26" />
      <ellipse cx="200" cy="460" rx="70" ry="15" />
      {/* plis */}
      {pleats.map((x) => (
        <path key={x} d={`M${x + 20} 80v360`} strokeWidth="1" />
      ))}
      {/* cotation */}
      <path d="M340 60v400M332 60h16M332 460h16" strokeDasharray="4 5" />
      <path d="M60 520h280M60 512v16M340 512v16" strokeDasharray="4 5" />
      <circle cx="200" cy="260" r="150" strokeDasharray="2 8" />
    </svg>
  );
}
