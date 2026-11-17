import { BoxGeometry, Group, Mesh, MeshStandardMaterial, SphereGeometry, TorusGeometry } from "three";
import { gsap } from "../motion";
import { COLORS, type SceneBuild } from "./types";

/** Rôles et permissions : chaque rôle porte des cubes (permissions) ; le membre les cumule tous. */
const build: SceneBuild = ({ scene, reduced }) => {
  const g = new Group();
  scene.add(g);
  const roleA = { x: -2.8, y: 1.15 }, roleB = { x: -2.8, y: -1.15 }, member = { x: 2.2, y: 0 };
  const matFor = (c: number) => new MeshStandardMaterial({ color: c, roughness: 0.4, metalness: 0.2, emissive: c, emissiveIntensity: 0.18 });

  for (const r of [roleA, roleB]) {
    const t = new Mesh(new TorusGeometry(1.0, 0.05, 10, 56), new MeshStandardMaterial({ color: COLORS.grey }));
    t.position.set(r.x, r.y, 0);
    g.add(t);
  }
  const mem = new Mesh(new SphereGeometry(0.7, 28, 20), new MeshStandardMaterial({ color: COLORS.navy, roughness: 0.4, emissive: COLORS.action, emissiveIntensity: 0 }));
  mem.position.set(member.x, member.y, 0);
  g.add(mem);

  // Permissions : [rôle, couleur]
  const cubeGeo = new BoxGeometry(0.34, 0.34, 0.34);
  const perms: { role: typeof roleA; color: number }[] = [
    { role: roleA, color: COLORS.link }, { role: roleA, color: COLORS.link }, { role: roleA, color: COLORS.ok },
    { role: roleB, color: COLORS.ok }, { role: roleB, color: COLORS.action }, { role: roleB, color: COLORS.slate },
  ];
  const cubes = perms.map((p) => { const m = new Mesh(cubeGeo, matFor(p.color)); g.add(m); return m; });

  const st = { m: reduced ? 1 : 0 }; // 0 = chaque cube sur son rôle, 1 = tous sur le membre
  const tl = gsap.timeline({ repeat: -1 });
  if (!reduced) tl.to(st, { m: 1, duration: 1.4, ease: "power2.inOut" }, 1.2).to(st, { m: 0, duration: 1.2, ease: "power2.inOut" }, "+=1.8");

  return {
    update: (t) => {
      cubes.forEach((c, i) => {
        const p = perms[i];
        const same = perms.filter((x) => x.role === p.role);
        const slot = same.indexOf(p);
        const a = t * 0.9 + (slot / same.length) * Math.PI * 2 + (p.role === roleA ? 0 : 1.3);
        const onRole = { x: p.role.x + Math.cos(a) * 1.0, y: p.role.y + Math.sin(a) * 1.0 };
        const b = -t * 1.1 + (i / cubes.length) * Math.PI * 2;
        const onMember = { x: member.x + Math.cos(b) * 1.15, y: member.y + Math.sin(b) * 1.15 };
        c.position.set(onRole.x + (onMember.x - onRole.x) * st.m, onRole.y + (onMember.y - onRole.y) * st.m, Math.sin(a * 2) * 0.15);
        c.rotation.set(t, t * 0.8, 0);
      });
      (mem.material as MeshStandardMaterial).emissiveIntensity = st.m * 0.45;
      mem.scale.setScalar(1 + st.m * 0.12);
    },
    stop: () => { tl.kill(); },
  };
};
export default build;
