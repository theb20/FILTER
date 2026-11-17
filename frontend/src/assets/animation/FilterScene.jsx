import { useEffect, useRef, useState } from "react";
import {
  WebGLRenderer, Scene, PerspectiveCamera, Group, BufferGeometry, BufferAttribute, Points, PointsMaterial,
  AmbientLight, DirectionalLight, PointLight, Clock, MathUtils,
} from "three";
import { buildCartridge } from "./cartridge.js";
import { gsap, prefersReducedMotion } from "./gsapSetup.js";
import LineArt from "../components/LineArt.jsx";

function buildParticles(count, disposables) {
  const pos = new Float32Array(count * 3);
  const speed = new Float32Array(count);
  for (let i = 0; i < count; i++) {
    const a = Math.random() * Math.PI * 2, r = 1.6 + Math.random() * 1.4;
    pos[i * 3] = Math.cos(a) * r;
    pos[i * 3 + 1] = (Math.random() - 0.5) * 7;
    pos[i * 3 + 2] = Math.sin(a) * r;
    speed[i] = 0.3 + Math.random() * 0.9;
  }
  const geo = new BufferGeometry();
  geo.setAttribute("position", new BufferAttribute(pos, 3));
  const mat = new PointsMaterial({ color: 0xf7be14, size: 0.045, transparent: true, opacity: 0.85, depthWrite: false });
  disposables.push(geo, mat);
  return { points: new Points(geo, mat), pos, speed, geo };
}

// Cartouche filtrante 3D (three.js) : suit le curseur, réagit au scroll, tourne au clic,
// et se fait glisser à la main avec `drag`. Se met en pause hors écran.
export default function FilterScene({ className = "", drag = false, label = "", particles = 520 }) {
  const wrap = useRef(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const host = wrap.current;
    if (!host) return;
    const reduced = prefersReducedMotion();
    const disposables = [];
    let renderer;
    try {
      renderer = new WebGLRenderer({ antialias: true, alpha: true, powerPreference: "high-performance" });
    } catch {
      setFailed(true);
      return;
    }
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    renderer.domElement.style.cssText = "display:block;width:100%;height:100%;touch-action:pan-y;";
    host.appendChild(renderer.domElement);

    const scene = new Scene();
    const camera = new PerspectiveCamera(32, 1, 0.1, 50);
    camera.position.set(0, 0.2, 9);

    scene.add(new AmbientLight(0xffffff, 0.75));
    const key = new DirectionalLight(0xffffff, 2.2);
    key.position.set(4, 5, 6);
    const rim = new DirectionalLight(0x6ab0ff, 1.2);
    rim.position.set(-5, 1, -4);
    const glow = new PointLight(0xf7be14, 18, 12);
    glow.position.set(0, 0, 3);
    scene.add(key, rim, glow);

    const cartridge = buildCartridge(disposables);
    const pivot = new Group();
    pivot.add(cartridge);
    scene.add(pivot);
    const flow = buildParticles(particles, disposables);
    scene.add(flow.points);

    const st = { px: 0, py: 0, rx: 0, ry: 0, spin: 0, dragY: 0, dragX: 0, flow: 1, intro: reduced ? 1 : 0 };
    let dragging = false, lastX = 0, lastY = 0;
    let visible = true, raf = 0;
    const clock = new Clock();

    const resize = () => {
      const w = host.clientWidth || 1, h = host.clientHeight || 1;
      renderer.setSize(w, h, false);
      camera.aspect = w / h;
      camera.position.z = w / h < 0.8 ? 11 : 9;
      camera.updateProjectionMatrix();
    };
    resize();

    const render = () => {
      const dt = Math.min(clock.getDelta(), 0.05);
      const t = clock.elapsedTime;
      // Lissage vers la position du curseur
      st.rx = MathUtils.lerp(st.rx, st.px, 0.07);
      st.ry = MathUtils.lerp(st.ry, st.py, 0.07);
      const scrollRot = window.scrollY * 0.004;
      pivot.rotation.y = t * 0.35 * (dragging ? 0 : 1) + st.rx * 0.9 + st.spin + st.dragY + scrollRot;
      pivot.rotation.x = st.ry * 0.35 + st.dragX + Math.sin(t * 0.6) * 0.04;
      pivot.rotation.z = 0.12 + Math.sin(t * 0.4) * 0.03;
      pivot.position.y = Math.sin(t * 0.9) * 0.1 - window.scrollY * 0.0006;
      pivot.scale.setScalar(0.001 + st.intro * 0.999);
      glow.position.set(st.rx * 4, -st.ry * 3, 3);

      // Flux d'air : particules qui montent à travers le filtre, accélérées au survol
      const p = flow.pos;
      for (let i = 0; i < flow.speed.length; i++) {
        p[i * 3 + 1] += flow.speed[i] * dt * st.flow;
        if (p[i * 3 + 1] > 3.5) p[i * 3 + 1] = -3.5;
      }
      flow.geo.attributes.position.needsUpdate = true;
      flow.points.rotation.y = -t * 0.05 + st.rx * 0.3;
      renderer.render(scene, camera);
    };
    const loop = () => { raf = requestAnimationFrame(loop); render(); };
    const start = () => { if (!raf && !reduced) { clock.getDelta(); loop(); } };
    const stop = () => { cancelAnimationFrame(raf); raf = 0; };

    // Entrée animée
    const ctx = gsap.context(() => {
      if (reduced) return;
      gsap.to(st, { intro: 1, duration: 1.6, ease: "elastic.out(1,0.6)" });
      gsap.from(st, { spin: -Math.PI * 2, duration: 2, ease: "power4.out" });
    });

    const onPointerMove = (e) => {
      st.px = (e.clientX / window.innerWidth) * 2 - 1;
      st.py = (e.clientY / window.innerHeight) * 2 - 1;
      if (dragging) {
        st.dragY += (e.clientX - lastX) * 0.012;
        st.dragX = MathUtils.clamp(st.dragX + (e.clientY - lastY) * 0.006, -0.8, 0.8);
        lastX = e.clientX; lastY = e.clientY;
      }
    };
    const onDown = (e) => {
      if (!drag) return;
      dragging = true; lastX = e.clientX; lastY = e.clientY;
      renderer.domElement.setPointerCapture?.(e.pointerId);
    };
    const onUp = () => { dragging = false; };
    const onClick = () => {
      if (reduced) return;
      gsap.to(st, { spin: st.spin + Math.PI * 2, duration: 1.4, ease: "back.out(1.4)", overwrite: "auto" });
    };
    const onEnter = () => gsap.to(st, { flow: 4, duration: 0.6 });
    const onLeave = () => gsap.to(st, { flow: 1, duration: 0.8 });

    window.addEventListener("pointermove", onPointerMove, { passive: true });
    const cv = renderer.domElement;
    cv.addEventListener("pointerdown", onDown);
    window.addEventListener("pointerup", onUp);
    cv.addEventListener("click", onClick);
    cv.addEventListener("pointerenter", onEnter);
    cv.addEventListener("pointerleave", onLeave);

    const ro = new ResizeObserver(() => { resize(); if (reduced) render(); });
    ro.observe(host);
    const io = new IntersectionObserver(([en]) => { visible = en.isIntersecting; visible ? start() : stop(); });
    io.observe(host);
    const onVis = () => (document.hidden ? stop() : visible && start());
    document.addEventListener("visibilitychange", onVis);

    if (reduced) render(); else start();

    return () => {
      stop(); ctx.revert(); gsap.killTweensOf(st);
      ro.disconnect(); io.disconnect();
      document.removeEventListener("visibilitychange", onVis);
      window.removeEventListener("pointermove", onPointerMove);
      window.removeEventListener("pointerup", onUp);
      cv.removeEventListener("pointerdown", onDown);
      cv.removeEventListener("click", onClick);
      cv.removeEventListener("pointerenter", onEnter);
      cv.removeEventListener("pointerleave", onLeave);
      disposables.forEach((d) => d.dispose?.());
      renderer.dispose();
      renderer.forceContextLoss?.();
      cv.remove();
    };
  }, [drag, particles]);

  if (failed) {
    return (
      <div className={className}>
        <LineArt stroke="#9ca3af" className="h-full w-full" />
      </div>
    );
  }
  return (
    <div
      ref={wrap}
      className={className}
      {...(label ? { role: "img", "aria-label": label } : { "aria-hidden": "true" })}
      style={{ cursor: drag ? "grab" : "pointer" }}
    />
  );
}
