// Balisage structuré (schema.org) : un <script type="application/ld+json"> est une donnée, pas du code
// exécutable ; il reste donc compatible avec la CSP (script-src 'self').
export default function JsonLd({ data }) {
  return <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(data).replace(/</g, "\\u003c") }} />;
}
