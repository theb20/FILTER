// Petits dessins au trait à deux tons (bleu nuit + bleu d'accent), faits pour être « dessinés » par GSAP :
// chaque trait a pathLength=1, donc strokeDasharray/offset = 1 → 0 révèle le tracé. Les classes ff-* servent aux mini-mouvements.
const INK = "#2a4d7a";
const ACC = "#1478f0";

function Svg({ size = 56, children, ...rest }) {
  return (
    <svg viewBox="0 0 64 64" width={size} height={size} fill="none" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" {...rest}>
      {children}
    </svg>
  );
}
// Trait encre / trait d'accent
const L = (p) => <path pathLength="1" className="ff-draw" stroke={INK} style={{ strokeDasharray: 1, strokeDashoffset: 0 }} {...p} />;
const A = (p) => <path pathLength="1" className="ff-draw" stroke={ACC} style={{ strokeDasharray: 1, strokeDashoffset: 0 }} {...p} />;
const C = (p, c = INK) => <circle pathLength="1" className="ff-draw" stroke={c} style={{ strokeDasharray: 1, strokeDashoffset: 0 }} {...p} />;

export const Clipboard = (props) => (
  <Svg {...props}>
    {L({ d: "M18 12h-4a3 3 0 0 0-3 3v39a3 3 0 0 0 3 3h30a3 3 0 0 0 3-3V15a3 3 0 0 0-3-3h-4" })}
    {L({ d: "M22 8h14a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2H22a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2z" })}
    {A({ d: "M19 27l2.5 2.5L26 25" })}{L({ d: "M31 27h10" })}
    {A({ d: "M19 37l2.5 2.5L26 35" })}{L({ d: "M31 37h10" })}
    {A({ d: "M19 47l2.5 2.5L26 45" })}{L({ d: "M31 47h6" })}
  </Svg>
);

export const Box = (props) => (
  <Svg {...props}>
    {L({ d: "M32 8l22 11v25L32 56 10 44V19z" })}
    {L({ d: "M10 19l22 11 22-11" })}{L({ d: "M32 30v26" })}
    {A({ d: "M21 13.5l22 11v7" })}
    {A({ d: "M16 40l6 3", className: "ff-draw ff-tilt" })}
  </Svg>
);

export const Truck = (props) => (
  <Svg {...props}>
    {L({ d: "M4 18h31a2 2 0 0 1 2 2v24H4z" })}
    {L({ d: "M37 26h9l9 9v9H37" })}{L({ d: "M43 26v9h12" })}
    <g className="ff-slide">{A({ d: "M11 31h17" })}{A({ d: "M23 26l5 5-5 5" })}</g>
    <g className="ff-wheel" style={{ transformBox: "fill-box", transformOrigin: "50% 50%" }}>{C({ cx: 17, cy: 46, r: 5, fill: "#f5f6f7" })}{A({ d: "M17 43v6" })}</g>
    <g className="ff-wheel" style={{ transformBox: "fill-box", transformOrigin: "50% 50%" }}>{C({ cx: 46, cy: 46, r: 5, fill: "#f5f6f7" })}{A({ d: "M46 43v6" })}</g>
    {L({ d: "M4 56h14M24 56h30", opacity: 0.5 })}
  </Svg>
);

export const House = (props) => (
  <Svg {...props}>
    {L({ d: "M6 31L32 9l26 22" })}{L({ d: "M12 27v28h40V27" })}
    {L({ d: "M44 17v-6h6v12" })}
    {A({ d: "M26 55V42a6 6 0 0 1 12 0v13" })}
    {A({ d: "M17 36h5v6h-5zM42 36h5v6h-5z" })}
    {L({ d: "M2 55h60" })}
  </Svg>
);

export const SearchRef = (props) => (
  <Svg {...props}>
    {C({ cx: 27, cy: 27, r: 15 })}
    {L({ d: "M38 38l17 17" })}
    {A({ d: "M19 25a9 9 0 0 1 8-8" })}
    {A({ d: "M21 31h12M21 37h8", className: "ff-draw ff-slide" })}
  </Svg>
);

export const QuoteDoc = (props) => (
  <Svg {...props}>
    {L({ d: "M16 7h22l12 12v36a2 2 0 0 1-2 2H16a2 2 0 0 1-2-2V9a2 2 0 0 1 2-2z" })}
    {L({ d: "M38 7v12h12" })}
    {A({ d: "M22 29h20M22 37h20M22 45h11" })}
  </Svg>
);

export const PayWallet = (props) => (
  <Svg {...props}>
    {L({ d: "M8 18h44a4 4 0 0 1 4 4v28a4 4 0 0 1-4 4H12a4 4 0 0 1-4-4z" })}
    {L({ d: "M8 18l36-8v8" })}
    {A({ d: "M8 29h48" })}
    <g className="ff-wheel" style={{ transformBox: "fill-box", transformOrigin: "50% 50%" }}>{C({ cx: 46, cy: 43, r: 4 }, ACC)}</g>
    {L({ d: "M15 43h14" })}
  </Svg>
);

export const Headset = (props) => (
  <Svg {...props}>
    {L({ d: "M12 36V32a20 20 0 0 1 40 0v4" })}
    {L({ d: "M10 34h6a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1h-3a3 3 0 0 1-3-3z" })}
    {L({ d: "M54 34h-6a1 1 0 0 0-1 1v12a1 1 0 0 0 1 1h3a3 3 0 0 0 3-3z" })}
    {A({ d: "M51 48c0 7-8 9-17 9" })}
    {C({ cx: 31, cy: 57, r: 2.5 }, ACC)}
    {A({ d: "M26 28l4 4 8-8", className: "ff-draw ff-tilt" })}
  </Svg>
);
