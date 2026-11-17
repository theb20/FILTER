import { BoxGeometry, CylinderGeometry, Group, Mesh, MeshStandardMaterial, TorusGeometry } from "three";
import { gsap } from "../motion";
import { COLORS, type SceneBuild } from "./types";

/** Cycle de vie : un produit avance de dalle en dalle (Brouillon → À valider → Publié → Archivé) puis revient en brouillon. */
const build: SceneBuild = ({ scene, reduced }) => {
  const g = new Group();
  scene.add(g);
  const X = [-3.3, -1.1, 1.1, 3.3];
  const Y = [0, 0.25, 0.5, -0.15];
  const col = [COLORS.grey, COLORS.action, COLORS.ok, 0x7b8aa3];
  const slabGeo = new BoxGeometry(1.9, 0.3, 1.5);
  const slabs = X.map((x, i) => {
    const mat = new MeshStandardMaterial({ color: col[i], roughness: 0.55, emissive: col[i], emissiveIntensity: 0 });
    const m = new Mesh(slabGeo, mat);
    m.position.set(x, Y[i] - 0.15, 0);
    g.add(m);
    return mat;
  });

  // Le produit : une cartouche (cylindre + anneau)
  const prod = new Group();
  prod.add(new Mesh(new CylinderGeometry(0.34, 0.34, 0.62, 28), new MeshStandardMaterial({ color: COLORS.navy, roughness: 0.4, metalness: 0.3 })));
  const ring = new Mesh(new TorusGeometry(0.35, 0.05, 10, 32), new MeshStandardMaterial({ color: COLORS.action, metalness: 0.7, roughness: 0.3 }));
  ring.rotation.x = Math.PI / 2; ring.position.y = 0.3;
  prod.add(ring);
  g.add(prod);

  const st = { p: reduced ? 2 : 0 };
  const lerpAt = (arr: number[], p: number) => { const i = Math.min(Math.floor(p), 2), f = p - i; return arr[i] + (arr[i + 1] - arr[i]) * f; };
  const tl = gsap.timeline({ repeat: -1 });
  if (!reduced) {
    for (let i = 1; i <= 3; i++) tl.to(st, { p: i, duration: 0.9, ease: "power2.inOut" }).to({}, { duration: 0.75 });
    tl.to(st, { p: 0, duration: 1.6, ease: "power2.inOut" }, "+=0.2").to({}, { duration: 0.5 });
  }

  return {
    update: (t) => {
      const p = st.p, frac = p - Math.floor(p);
      prod.position.x = lerpAt(X, p);
      prod.position.y = lerpAt(Y, p) + 0.62 + Math.sin(Math.PI * frac) * (p === Math.round(p) ? 0 : 0.7);
      prod.rotation.y = t * 0.9;
      slabs.forEach((m, i) => { m.emissiveIntensity = Math.max(0, 1 - Math.abs(p - i) * 1.6) * 0.55; });
      g.rotation.x = 0.18;
    },
    stop: () => { tl.kill(); },
  };
};
export default build;
