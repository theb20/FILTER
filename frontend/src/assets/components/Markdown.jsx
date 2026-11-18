import { Fragment } from "react";

// Rendu Markdown minimal et SÛR (aucun HTML brut) pour les pages gérées au backoffice :
// intertitres (#, ##, ###), listes (- ou 1.), paragraphes, **gras**, *italique*, [texte](https://…).
const SAFE_URL = /^(https?:\/\/|mailto:|tel:|\/)/i;

function inline(text, keyBase) {
  const out = [];
  const re = /(\*\*([^*]+)\*\*|\*([^*]+)\*|\[([^\]]+)\]\(([^)\s]+)\))/g;
  let last = 0;
  let m;
  let i = 0;
  while ((m = re.exec(text))) {
    if (m.index > last) out.push(text.slice(last, m.index));
    const key = `${keyBase}-${i++}`;
    if (m[2]) out.push(<strong key={key}>{m[2]}</strong>);
    else if (m[3]) out.push(<em key={key}>{m[3]}</em>);
    else if (SAFE_URL.test(m[5])) out.push(<a key={key} href={m[5]} className="text-link underline" {...(m[5].startsWith("/") ? {} : { target: "_blank", rel: "noopener noreferrer" })}>{m[4]}</a>);
    else out.push(m[4]);
    last = m.index + m[0].length;
  }
  if (last < text.length) out.push(text.slice(last));
  return out;
}

/** Un paragraphe Markdown rendu en ligne (gras, italique, liens sûrs), sans style de couleur : hérite du parent. */
export function MarkdownInline({ text }) {
  return text.split("\n").map((l, j) => <Fragment key={j}>{j > 0 && <br />}{inline(l, `i${j}`)}</Fragment>);
}

export default function Markdown({ source }) {
  const blocks = source.replace(/\r/g, "").split(/\n{2,}/).map((b) => b.trim()).filter(Boolean);
  return (
    <div className="space-y-4 text-[16px] leading-relaxed text-gray-800">
      {blocks.map((b, i) => {
        const h = b.match(/^(#{1,3})\s+(.+)$/);
        if (h) {
          const Tag = h[1].length === 1 ? "h2" : h[1].length === 2 ? "h2" : "h3";
          return <Tag key={i} className={Tag === "h2" ? "pt-2 text-[24px] font-bold" : "pt-1 text-[19px] font-bold"}>{inline(h[2], `h${i}`)}</Tag>;
        }
        const lines = b.split("\n");
        if (lines.every((l) => /^[-*]\s+/.test(l))) {
          return <ul key={i} className="list-disc space-y-1 pl-6">{lines.map((l, j) => <li key={j}>{inline(l.replace(/^[-*]\s+/, ""), `u${i}-${j}`)}</li>)}</ul>;
        }
        if (lines.every((l) => /^\d+[.)]\s+/.test(l))) {
          return <ol key={i} className="list-decimal space-y-1 pl-6">{lines.map((l, j) => <li key={j}>{inline(l.replace(/^\d+[.)]\s+/, ""), `o${i}-${j}`)}</li>)}</ol>;
        }
        return <p key={i}>{lines.map((l, j) => <Fragment key={j}>{j > 0 && <br />}{inline(l, `p${i}-${j}`)}</Fragment>)}</p>;
      })}
    </div>
  );
}
