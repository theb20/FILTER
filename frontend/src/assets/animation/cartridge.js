import { Group, Mesh, Shape, Path, ExtrudeGeometry, CylinderGeometry, TorusGeometry, MeshStandardMaterial, MeshBasicMaterial, DoubleSide } from "three";

export const R_OUT = 1.25, R_IN = 1.05, R_CORE = 0.6, HEIGHT = 3.2, PLEATS = 44;

// Anneau extrudé (embout de cartouche) : disque percé d'un trou central.
function ringGeometry(rOut, rIn, depth) {
  const s = new Shape();
  s.absarc(0, 0, rOut, 0, Math.PI * 2, false);
  const hole = new Path();
  hole.absarc(0, 0, rIn, 0, Math.PI * 2, true);
  s.holes.push(hole);
  const g = new ExtrudeGeometry(s, { depth, bevelEnabled: false, curveSegments: 64 });
  g.rotateX(-Math.PI / 2);
  return g;
}

// Média filtrant plissé : étoile à PLEATS branches extrudée sur la hauteur.
function pleatedGeometry() {
  const s = new Shape();
  const steps = PLEATS * 2;
  for (let i = 0; i < steps; i++) {
    const a = (i / steps) * Math.PI * 2;
    const r = i % 2 ? R_IN : R_OUT;
    const x = Math.cos(a) * r, y = Math.sin(a) * r;
    if (i) s.lineTo(x, y); else s.moveTo(x, y);
  }
  s.closePath();
  const hole = new Path();
  hole.absarc(0, 0, R_CORE, 0, Math.PI * 2, true);
  s.holes.push(hole);
  const g = new ExtrudeGeometry(s, { depth: HEIGHT, bevelEnabled: false });
  g.rotateX(-Math.PI / 2);
  g.translate(0, -HEIGHT / 2, 0);
  return g;
}

/** Cartouche filtrante : enfants nommés (media, capTop, capBottom, ringTop, ringBottom, core) pour pouvoir les mettre en valeur. */
export function buildCartridge(disposables) {
  const group = new Group();
  const track = (o) => { disposables.push(o); return o; };

  const media = new Mesh(
    track(pleatedGeometry()),
    track(new MeshStandardMaterial({ color: 0xe3d6a8, roughness: 0.85, metalness: 0.05, flatShading: true })),
  );
  const steel = track(new MeshStandardMaterial({ color: 0x2b3442, roughness: 0.35, metalness: 0.85 }));
  const capGeo = track(ringGeometry(R_OUT + 0.08, R_CORE - 0.05, 0.22));
  const top = new Mesh(capGeo, steel);
  top.position.y = HEIGHT / 2;
  const bottom = new Mesh(capGeo, steel);
  bottom.position.y = -HEIGHT / 2 - 0.22;

  const goldMat = track(new MeshStandardMaterial({ color: 0xf7be14, roughness: 0.3, metalness: 0.9, emissive: 0x4a3300, emissiveIntensity: 0.4 }));
  const ringGeo = track(new TorusGeometry(R_OUT + 0.08, 0.04, 16, 72));
  const r1 = new Mesh(ringGeo, goldMat);
  r1.rotation.x = Math.PI / 2;
  r1.position.y = HEIGHT / 2 + 0.22;
  const r2 = r1.clone();
  r2.position.y = -HEIGHT / 2 - 0.22;

  // Tube central perforé (grille dorée)
  const core = new Mesh(
    track(new CylinderGeometry(R_CORE - 0.04, R_CORE - 0.04, HEIGHT, 20, 14, true)),
    track(new MeshBasicMaterial({ color: 0xf7be14, wireframe: true, transparent: true, opacity: 0.55, side: DoubleSide })),
  );

  media.name = "media"; top.name = "capTop"; bottom.name = "capBottom"; r1.name = "ringTop"; r2.name = "ringBottom"; core.name = "core";
  group.add(media, top, bottom, r1, r2, core);
  return group;
}

