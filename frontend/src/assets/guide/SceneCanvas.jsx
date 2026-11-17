import { useEffect, useRef, useState } from "react";
import { AmbientLight, DirectionalLight, PerspectiveCamera, Scene, WebGLRenderer } from "three";
import { useI18n } from "../i18n/i18nContext.js";

// Chaque scène est un module à part : seul le code de la scène affichée est chargé.
const SCENES = {
  "filter-anatomy": {
    load: () => import("./scenes/filterAnatomy.js"),
    legend: [["media", "#f7be14"], ["core", "#0275c8"], ["caps", "#00b67a"]],
  },
};

/** Moteur commun : rendu, redimensionnement, pause hors écran, parallaxe au pointeur, libération complète des ressources WebGL. */
export default function SceneCanvas({ id, animate }) {
  const { t } = useI18n();
  const host = useRef(null);
  const [failed, setFailed] = useState(false);
  const meta = SCENES[id];

  useEffect(() => {
    const el = host.current;
    if (!el || !meta) return;
    let renderer;
    try {
      renderer = new WebGLRenderer({ antialias: true, alpha: true });
    } catch {
      queueMicrotask(() => setFailed(true)); // pas de WebGL : la légende suffit
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

    const pointer = { x: 0, y: 0 };
    let runtime = null, raf = 0, last = 0, visible = true, cancelled = false;
    const reduced = !animate;

    const resize = () => {
      const w = el.clientWidth, h = el.clientHeight;
      if (!w || !h) return;
      renderer.setSize(w, h, false);
      camera.aspect = w / h;
      camera.updateProjectionMatrix();
    };
    const frame = (now) => {
      const sec = now / 1000, dt = Math.min(sec - last, 0.05);
      last = sec;
      camera.position.x += (pointer.x * 0.9 - camera.position.x) * 0.05;
      camera.position.y += (0.3 - pointer.y * 0.6 - camera.position.y) * 0.05;
      camera.lookAt(0, 0, 0);
      runtime?.update(sec, dt, pointer);
      renderer.render(scene, camera);
    };
    const loop = (now) => { raf = requestAnimationFrame(loop); frame(now); };
    const start = () => { if (!raf && !reduced && runtime) { last = performance.now() / 1000; raf = requestAnimationFrame(loop); } };
    const stop = () => { cancelAnimationFrame(raf); raf = 0; };

    resize();
    const ro = new ResizeObserver(() => { resize(); if (reduced && runtime) frame(2400); });
    ro.observe(el);
    const io = new IntersectionObserver(([e]) => { visible = e.isIntersecting; if (visible) start(); else stop(); });
    io.observe(el);
    const onMove = (e) => {
      const r = el.getBoundingClientRect();
      pointer.x = Math.max(-1, Math.min(1, ((e.clientX - r.left) / r.width) * 2 - 1));
      pointer.y = Math.max(-1, Math.min(1, ((e.clientY - r.top) / r.height) * 2 - 1));
    };
    el.addEventListener("pointermove", onMove);
    const onVis = () => (document.hidden ? stop() : visible && start());
    document.addEventListener("visibilitychange", onVis);

    meta.load().then((m) => {
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
        o.geometry?.dispose();
        const mat = o.material;
        (Array.isArray(mat) ? mat : mat ? [mat] : []).forEach((x) => x.dispose());
      });
      renderer.dispose();
      renderer.forceContextLoss();
      renderer.domElement.remove();
    };
  }, [id, animate, meta]);

  return (
    <figure className="mb-3 overflow-hidden rounded-[6px] bg-gray-100 [@media(max-height:560px)]:hidden" data-step-content>
      {!failed && <div ref={host} aria-hidden="true" className="h-40 w-full sm:h-44" />}
      <figcaption className="px-3 py-2 text-xs leading-snug text-gray-700">
        <p>{t(`guide.scene.${id}`)}</p>
        <ul className="mt-1.5 flex flex-wrap gap-x-3 gap-y-1">
          {meta.legend.map(([key, color]) => (
            <li key={key} className="flex items-center gap-1.5">
              <span className="h-2.5 w-2.5 rounded-full" style={{ background: color }} aria-hidden="true" />
              {t(`guide.scene.${id}.${key}`)}
            </li>
          ))}
        </ul>
      </figcaption>
    </figure>
  );
}
