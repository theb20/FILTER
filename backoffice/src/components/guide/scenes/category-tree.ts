import { CylinderGeometry, Group, Mesh, MeshStandardMaterial, Quaternion, SphereGeometry, Vector3 } from "three";
import { COLORS, type SceneBuild } from "./types";

type Node = { p: Vector3; mesh: Mesh; mat: MeshStandardMaterial; level: number };

/** Arbre de catégories : des attributs (points jaunes) descendent du parent vers les enfants, qui s'illuminent à l'arrivée. */
const build: SceneBuild = ({ scene, reduced }) => {
  const g = new Group();
  scene.add(g);
  const pos: [number, number, number, number][] = [ // x, y, z, niveau
    [0, 1.7, 0, 0],
    [-2.3, 0.2, 0.2, 1], [0, 0.2, -0.2, 1], [2.3, 0.2, 0.2, 1],
    [-3.1, -1.4, 0, 2], [-1.5, -1.4, 0.3, 2], [0, -1.4, 0, 2], [1.5, -1.4, 0.3, 2], [3.1, -1.4, 0, 2],
  ];
  const parentOf = [-1, 0, 0, 0, 1, 1, 2, 3, 3];
  const nodeGeo = [new SphereGeometry(0.42, 24, 16), new SphereGeometry(0.32, 24, 16), new SphereGeometry(0.25, 20, 14)];
  const nodeCol = [COLORS.action, COLORS.navy, COLORS.slate];
  const nodes: Node[] = pos.map(([x, y, z, level]) => {
    const mat = new MeshStandardMaterial({ color: nodeCol[level], roughness: 0.45, metalness: 0.15, emissive: COLORS.action, emissiveIntensity: 0 });
    const mesh = new Mesh(nodeGeo[level], mat);
    mesh.position.set(x, y, z);
    g.add(mesh);
    return { p: new Vector3(x, y, z), mesh, mat, level };
  });

  // Arêtes
  const edgeMat = new MeshStandardMaterial({ color: COLORS.grey, roughness: 0.8 });
  const edges = parentOf.map((pi, i) => (pi < 0 ? null : { a: nodes[pi], b: nodes[i], i })).filter((e): e is NonNullable<typeof e> => e !== null);
  for (const e of edges) {
    const d = new Vector3().subVectors(e.b.p, e.a.p);
    const m = new Mesh(new CylinderGeometry(0.035, 0.035, d.length(), 8), edgeMat);
    m.position.copy(e.a.p).addScaledVector(d, 0.5);
    m.quaternion.copy(new Quaternion().setFromUnitVectors(new Vector3(0, 1, 0), d.normalize()));
    g.add(m);
  }

  // Attributs : deux par arête, qui voyagent de haut en bas
  const tokGeo = new SphereGeometry(0.1, 12, 10);
  const tokMat = new MeshStandardMaterial({ color: COLORS.action, emissive: COLORS.action, emissiveIntensity: 0.9 });
  const tokens = edges.flatMap((e) => [0, 0.35].map((off) => ({ e, off, mesh: new Mesh(tokGeo, tokMat) })));
  tokens.forEach((t) => { t.mesh.visible = false; g.add(t.mesh); });

  const CYCLE = 4.2;
  const state = (t: number) => {
    const c = (t % CYCLE) / CYCLE;
    nodes.forEach((n) => { n.mat.emissiveIntensity = 0; n.mesh.scale.setScalar(1); });
    for (const k of tokens) {
      // niveau 1 : première moitié du cycle ; niveau 2 : seconde moitié
      const seg0 = k.e.b.level === 1 ? 0 : 0.5;
      const local = (c - seg0 - k.off * 0.25) / 0.45;
      const on = local > 0 && local < 1;
      k.mesh.visible = on;
      if (on) k.mesh.position.lerpVectors(k.e.a.p, k.e.b.p, local);
      if (local >= 0.9 && local < 1.6) { // arrivée : le nœud s'illumine
        const pulse = Math.max(0, 1 - (local - 0.9) / 0.7);
        k.e.b.mat.emissiveIntensity = Math.max(k.e.b.mat.emissiveIntensity, pulse * 0.8);
        k.e.b.mesh.scale.setScalar(1 + pulse * 0.25);
      }
    }
  };
  // Animations réduites : tous les attributs sont posés, l'arbre « chargé ».
  if (reduced) nodes.forEach((n) => { n.mat.emissiveIntensity = n.level ? 0.5 : 0; });

  return {
    update: (t) => { if (reduced) return; state(t); g.rotation.y = Math.sin(t * 0.35) * 0.18; },
  };
};
export default build;
