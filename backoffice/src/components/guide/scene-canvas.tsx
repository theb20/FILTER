"use client";

import { useEffect, useRef, useState } from "react";
import { AmbientLight, DirectionalLight, Mesh, PerspectiveCamera, Scene, WebGLRenderer } from "three";
import type { SceneId } from "./types";
import type { Pointer, SceneBuild, SceneRuntime } from "./scenes/types";
import { SCENE_META } from "./scenes/meta";

// Chaque scène est un module à part : seul le code de la scène affichée est chargé.
const LOADERS: Record<SceneId, () => Promise<{ default: SceneBuild }>> = {
  "category-tree": () => import("./scenes/category-tree"),
  "product-lifecycle": () => import("./scenes/product-lifecycle"),
  "roles-permissions": () => import("./scenes/roles-permissions"),
};

/** Moteur commun : rendu, redimensionnement, pause hors écran, parallaxe au pointeur, libération complète des ressources WebGL. */
export default function SceneCanvas({ id, animate }: { id: SceneId; animate: boolean }) {
  const host = useRef<HTMLDivElement>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const el = host.current;
    if (!el) return;
    let renderer: WebGLRenderer;
    try {
      renderer = new WebGLRenderer({ antialias: true, alpha: true });
    } catch {
      // Pas de WebGL : la légende sous la scène suffit, rien n'est cassé.
      queueMicrotask(() => setFailed(true));
      return;
    }
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    renderer.domElement.style.cssText = "display:block;width:100%;height:100%";
    el.appendChild(renderer.domElement);

    const scene = new Scene();
    const camera = new PerspectiveCamera(35, 2, 0.1, 50);
    camera.position.set(0, 0.3, 8);
    scene.add(new AmbientLight(0xffffff, 1.1));
    const key = new DirectionalLight(0xffffff, 2.2);
    key.position.set(3, 5, 6);
    scene.add(key);

    const pointer: Pointer = { x: 0, y: 0 };
    let runtime: SceneRuntime | null = null;
    let raf = 0, last = 0, visible = true, cancelled = false;
    const reduced = !animate;

    const resize = () => {
      const w = el.clientWidth, h = el.clientHeight;
      if (!w || !h) return;
      renderer.setSize(w, h, false);
      camera.aspect = w / h;
      camera.updateProjectionMatrix();
    };
    const frame = (now: number) => {
      const t = now / 1000, dt = Math.min(t - last, 0.05);
      last = t;
      camera.position.x += (pointer.x * 0.9 - camera.position.x) * 0.05;
      camera.position.y += (0.3 - pointer.y * 0.6 - camera.position.y) * 0.05;
      camera.lookAt(0, 0, 0);
      runtime?.update(t, dt, pointer);
      renderer.render(scene, camera);
    };
    const loop = (now: number) => { raf = requestAnimationFrame(loop); frame(now); };
    const start = () => { if (!raf && !reduced && runtime) { last = performance.now() / 1000; raf = requestAnimationFrame(loop); } };
    const stop = () => { cancelAnimationFrame(raf); raf = 0; };

    resize();
    const ro = new ResizeObserver(() => { resize(); if (reduced && runtime) frame(2400); });
    ro.observe(el);
    const io = new IntersectionObserver(([e]) => { visible = e.isIntersecting; if (visible) start(); else stop(); });
    io.observe(el);
    const onMove = (e: PointerEvent) => {
      const r = el.getBoundingClientRect();
      pointer.x = Math.max(-1, Math.min(1, ((e.clientX - r.left) / r.width) * 2 - 1));
      pointer.y = Math.max(-1, Math.min(1, ((e.clientY - r.top) / r.height) * 2 - 1));
    };
    el.addEventListener("pointermove", onMove);
    const onVis = () => (document.hidden ? stop() : visible && start());
    document.addEventListener("visibilitychange", onVis);

    LOADERS[id]().then((m) => {
      if (cancelled) return;
      runtime = m.default({ scene, camera, reduced });
      if (reduced) frame(2400); else start();
    });

    return () => {
      cancelled = true;
      stop();
      runtime?.stop?.();
      ro.disconnect(); io.disconnect();
      el.removeEventListener("pointermove", onMove);
      document.removeEventListener("visibilitychange", onVis);
      scene.traverse((o) => {
        const m = o as Mesh;
        m.geometry?.dispose();
        const mat = m.material;
        (Array.isArray(mat) ? mat : mat ? [mat] : []).forEach((x) => x.dispose());
      });
      renderer.dispose();
      renderer.forceContextLoss();
      renderer.domElement.remove();
    };
  }, [id, animate]);

  return (
    <figure className="mb-3 overflow-hidden rounded-lg border border-line bg-page [@media(max-height:560px)]:hidden" data-step-content>
      {!failed && <div ref={host} aria-hidden className="h-40 w-full sm:h-44" />}
      <figcaption className="px-3 py-2 text-xs leading-snug text-slate">{SCENE_META[id].caption}</figcaption>
    </figure>
  );
}
