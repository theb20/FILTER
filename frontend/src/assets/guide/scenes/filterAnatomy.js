import { BufferGeometry, Float32BufferAttribute, Group, Line, LineBasicMaterial, Mesh, MeshBasicMaterial, SphereGeometry, Vector3 } from "three";
import { buildCartridge, HEIGHT, R_OUT } from "../../animation/cartridge.js";

// Les trois pièces expliquées, avec la couleur de leur repère (la légende sous la scène reprend ces couleurs).
export const PARTS = [
  { key: "media", color: 0xf7be14, pin: [R_OUT, 0.5, 0], label: [2.9, 0.6] },
  { key: "core", color: 0x0275c8, pin: [0, 0.3, 0], label: [2.9, 1.7] },
  { key: "caps", color: 0x00b67a, pin: [R_OUT + 0.08, -HEIGHT / 2 - 0.1, 0], label: [2.9, -1.6] },
];
const SLOT = 2.4; // secondes par pièce mise en valeur

/** Anatomie d'une cartouche : média plissé, tube central, embouts. Chaque pièce s'éclaire à tour de rôle, reliée à son repère coloré. */
export default function build({ scene, reduced }) {
  const root = new Group();
  root.position.x = -0.9;
  scene.add(root);

  const cart = buildCartridge([]);
  root.add(cart);
  const by = (n) => cart.getObjectByName(n);
  const media = by("media").material, core = by("core").material, steel = by("capTop").material, gold = by("ringTop").material;

  // Repères : posés en « rayons X » (sans test de profondeur) pour être visibles même derrière le média
  const ann = new Group();
  root.add(ann);
  const pins = PARTS.map((p) => {
    const mat = new MeshBasicMaterial({ color: p.color, depthTest: false, transparent: true });
    const pin = new Mesh(new SphereGeometry(0.09, 14, 10), mat);
    pin.position.set(...p.pin);
    pin.renderOrder = 10;
    const label = new Mesh(new SphereGeometry(0.15, 16, 12), mat);
    label.position.set(p.label[0], p.label[1], 0);
    label.renderOrder = 10;
    const g = new BufferGeometry();
    g.setAttribute("position", new Float32BufferAttribute([...p.pin, p.label[0], p.label[1], 0], 3));
    const line = new Line(g, new LineBasicMaterial({ color: p.color, depthTest: false, transparent: true, opacity: 0.85 }));
    line.renderOrder = 9;
    ann.add(pin, label, line);
    return { pin, label, line, mat, base: new Vector3(1, 1, 1) };
  });

  const apply = (active, env) => {
    media.emissive.set(PARTS[0].color); media.emissiveIntensity = active === 0 ? env * 0.8 : 0;
    core.color.set(active === 1 ? PARTS[1].color : PARTS[0].color); core.opacity = active === 1 ? 0.55 + env * 0.45 : 0.4;
    steel.emissive.set(PARTS[2].color); steel.emissiveIntensity = active === 2 ? env * 0.7 : 0;
    gold.emissiveIntensity = active === 2 ? env * 0.8 : 0.4;
    pins.forEach((p, i) => {
      const on = i === active;
      p.pin.scale.setScalar(on ? 1 + env * 0.9 : 1);
      p.label.scale.setScalar(on ? 1 + env * 0.6 : 0.85);
      p.line.material.opacity = on ? 0.5 + env * 0.5 : 0.35;
    });
  };

  return {
    update: (t) => {
      const slot = Math.floor(t / SLOT) % PARTS.length;
      const frac = (t % SLOT) / SLOT;
      apply(slot, Math.sin(Math.PI * frac) ** 0.6);
      cart.rotation.y = reduced ? 0.6 : t * 0.35; // la cartouche tourne ; les repères restent fixes à l'écran
      cart.rotation.z = 0.08;
    },
  };
}
