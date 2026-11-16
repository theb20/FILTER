// Moyens de paiement courants en Côte d'Ivoire (mobile money + cartes).
// Pastilles SVG stylisées aux couleurs de chaque marque (pas les logos officiels) :
// pour utiliser les vrais logos, déposer les fichiers dans public/img/payments/ et les référencer ici.
const METHODS = [
{ name: "Orange Money", src: "/img/payments/orange.png" },
{ name: "MTN MoMo", src: "/img/payments/mtn.png" },
{ name: "Moov Money", src: "/img/payments/moov.png" },
{ name: "Wave", src: "/img/payments/wave.png" },
{ name: "Carte bancaire", src: "/img/payments/cb.png" },

];

function Badge({ name, bg, fg, lines, italic, cards, src }) {
  if (src) {
    return <img src={src} alt={name} width="64" height="40" loading="lazy" className="h-10 w-16 rounded-[6px] bg-transparent object-contain" />;
  }
  return (
    <svg
      viewBox="0 0 64 40"
      role="img"
      aria-label={name}
      className="h-10 w-16 rounded-[6px]"
    >
      <title>{name}</title>
      <rect width="64" height="40" fill={bg} />
      {cards ? (
        <>
          <circle cx="26" cy="20" r="10" fill="#EB001B" />
          <circle cx="38" cy="20" r="10" fill="#F79E1B" fillOpacity="0.9" />
        </>
      ) : (
        lines.map((text, i) => (
          <text
            key={text}
            x="32"
            y={lines.length === 1 ? 25 : 18 + i * 12}
            textAnchor="middle"
            fill={fg}
            fontFamily="Roboto, Arial, sans-serif"
            fontWeight="700"
            fontStyle={italic ? "italic" : "normal"}
            fontSize={lines.length === 1 ? 15 : 11}
          >
            {text}
          </text>
        ))
      )}
    </svg>
  );
}

export default function PaymentMethods() {
  return (
    <ul className="flex flex-wrap gap-2">
      {METHODS.map((m) => (
        <li key={m.name}>
          <Badge {...m} />
        </li>
      ))}
    </ul>
  );
}
