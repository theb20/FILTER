import { useEffect, useLayoutEffect, useRef } from "react";
import { AmbientLight, BoxGeometry, CircleGeometry, Color, DirectionalLight, Group, Mesh, MeshBasicMaterial, MeshStandardMaterial, OrthographicCamera, Scene, WebGLRenderer } from "three";
import { gsap, ScrollTrigger, prefersReducedMotion } from "../../animation/gsapSetup.js";

// Suivi d'étapes animé : barre de progression qui se remplit, pastilles à coche dessinée, icônes qui rebondissent (GSAP)
// et un petit colis 3D (Three.js) qui « voyage » jusqu'à l'étape en cours. Générique : on lui passe les étapes et l'étape courante
// (aujourd'hui : demandes de devis ; demain : commandes).
//
// steps   : [{ key, label, Icon }]   current : index de l'étape en cours (0 = première)   tone : "blue" | "green"
// onView : l'animation démarre quand le bloc entre à l'écran. described : chaque étape a un titre + une phrase (steps[i].desc), colonnes alignées.
export default function OrderTracker({ steps, current, title, titleAccent, meta = [], tone = "blue", onView = false, described = false }) {
  const root = useRef(null);
  const canvas = useRef(null);
  const nodes = useRef([]);
  const fill = useRef(null);
  const scene3d = useRef(null);
  const last = steps.length - 1;
  const cur = Math.max(0, Math.min(current, last));
  const done = cur >= last;

  // ── Animation GSAP : entrée puis changement d'étape ──
  useLayoutEffect(() => {
    const el = root.current;
    if (!el) return undefined;
    const pct = (cur / last) * 100;
    const reduced = prefersReducedMotion();
    const ctx = gsap.context(() => {
      if (reduced) {
        gsap.set(fill.current, { width: `${pct}%` });
        gsap.set("[data-check] path", { strokeDashoffset: 0 });
        gsap.set(".ff-draw", { strokeDashoffset: 0 });
        return;
      }
      const tl = gsap.timeline({ defaults: { ease: "power3.out" }, ...(onView ? { scrollTrigger: { trigger: el, start: "top 82%", once: true } } : {}) });
      tl.from("[data-head]", { y: -14, opacity: 0, duration: 0.5, stagger: 0.08 })
        .from("[data-node]", { scale: 0, duration: 0.5, ease: "back.out(2.2)", stagger: 0.12 }, "-=0.2")
        .fromTo(fill.current, { width: "0%" }, { width: `${pct}%`, duration: 0.5 + cur * 0.55, ease: "power2.inOut" }, "-=0.35")
        .fromTo("[data-done] [data-check] path", { strokeDashoffset: 24 }, { strokeDashoffset: 0, duration: 0.45, stagger: 0.35 }, "<0.2")
        .from("[data-label]", { y: 16, opacity: 0, duration: 0.5, stagger: 0.1 }, "-=0.9")
        .fromTo("[data-icon]", { scale: 0.7 }, { scale: 1, duration: 0.7, ease: "elastic.out(1,0.5)", stagger: 0.18 }, "<")
        .fromTo("[data-icon] .ff-draw", { strokeDashoffset: 1 }, { strokeDashoffset: 0, duration: 0.7, ease: "power1.inOut", stagger: 0.04 }, "<0.1");
      // Mini-mouvements en boucle sur l'étape en cours : flèche qui glisse, roues qui tournent, détail qui oscille.
      gsap.to("[data-active-icon] .ff-slide", { x: 4, duration: 0.7, repeat: -1, yoyo: true, ease: "sine.inOut" });
      gsap.to("[data-active-icon] .ff-wheel", { rotate: 360, duration: 1.4, repeat: -1, ease: "none" });
      gsap.to("[data-active-icon] .ff-tilt", { y: -1.5, duration: 0.6, repeat: -1, yoyo: true, ease: "sine.inOut" });
      gsap.fromTo("[data-pulse]", { scale: 1, opacity: 0.55 }, { scale: 2.1, opacity: 0, duration: 1.6, repeat: -1, ease: "power1.out" });
      gsap.to("[data-active-icon]", { y: -4, duration: 0.9, repeat: -1, yoyo: true, ease: "sine.inOut" });
    }, el);
    return () => ctx.revert();
  }, [cur, last, onView]);

  // ── Three.js : colis 3D au-dessus de l'étape courante ──
  useEffect(() => {
    const cv = canvas.current;
    if (!cv) return undefined;
    let renderer;
    try {
      renderer = new WebGLRenderer({ canvas: cv, alpha: true, antialias: true });
    } catch {
      return undefined; // pas de WebGL : le suivi reste lisible sans le colis
    }
    const reduced = prefersReducedMotion();
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    const scene = new Scene();
    const camera = new OrthographicCamera(0, 1, 1, 0, -400, 400);
    scene.add(new AmbientLight(0xffffff, 1.6));
    const sun = new DirectionalLight(0xffffff, 2.2);
    sun.position.set(-80, 140, 160);
    scene.add(sun);

    const parcel = new Group();
    const mat = new MeshStandardMaterial({ color: new Color("#d09a46"), roughness: 0.7, metalness: 0.05 });
    const body = new Mesh(new BoxGeometry(34, 26, 26), mat);
    const tape = new Mesh(new BoxGeometry(8, 26.6, 26.6), new MeshStandardMaterial({ color: "#f1d9a6", roughness: 0.6 }));
    const lid = new Mesh(new BoxGeometry(34.6, 3, 26.6), new MeshStandardMaterial({ color: "#b9833a", roughness: 0.7 }));
    lid.position.y = 13.5;
    parcel.add(body, tape, lid);
    const shadow = new Mesh(new CircleGeometry(15, 32), new MeshBasicMaterial({ color: 0x000000, transparent: true, opacity: 0.14 }));
    shadow.rotation.x = -Math.PI / 2.3;
    scene.add(parcel, shadow);

    const state = { x: 0, hop: 0, spin: 0.6 };
    const place = () => {
      const w = cv.clientWidth, h = cv.clientHeight;
      renderer.setSize(w, h, false);
      camera.right = w; camera.top = h; camera.updateProjectionMatrix();
    };
    const nodeX = (i) => {
      const n = nodes.current[i], host = cv.parentElement;
      return n && host ? n.getBoundingClientRect().left - host.getBoundingClientRect().left + n.offsetWidth / 2 : 0;
    };
    const apply = () => {
      parcel.position.set(state.x, 38 + state.hop, 0);
      parcel.rotation.set(0.45, state.spin, 0.05);
      shadow.position.set(state.x, 16, -20);
      const s = 1 - Math.min(0.35, state.hop / 120);
      shadow.scale.set(s, s, 1);
      renderer.render(scene, camera);
    };
    place();
    state.x = reduced ? nodeX(cur) : nodeX(0);
    scene3d.current = { state, nodeX, mat };
    const ro = new ResizeObserver(() => { place(); state.x = nodeX(scene3d.current.target ?? cur); apply(); });
    ro.observe(cv);

    const tick = (_, dt) => {
      if (!reduced) {
        state.spin += dt * 0.0011;
        state.hop = Math.max(state.hop, 0) * 0.9 + (Math.sin(performance.now() / 380) * 3);
      }
      apply();
    };
    gsap.ticker.add(tick);
    apply();
    return () => { gsap.ticker.remove(tick); ro.disconnect(); renderer.dispose(); scene3d.current = null; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Le colis voyage vers l'étape courante (saut en arc), et devient vert à la fin.
  useEffect(() => {
    const sc = scene3d.current;
    if (!sc) return;
    sc.target = cur;
    const x = sc.nodeX(cur);
    if (prefersReducedMotion()) { sc.state.x = x; return undefined; }
    const go = () => {
      gsap.to(sc.state, { x, duration: 0.7 + cur * 0.55, delay: 0.5, ease: "power2.inOut" });
      gsap.fromTo(sc.state, { hop: 0 }, { hop: 26, duration: 0.45, delay: 0.5, yoyo: true, repeat: 1, ease: "sine.out" });
      const c = new Color(done ? "#3fae6f" : "#d09a46");
      gsap.to(sc.mat.color, { r: c.r, g: c.g, b: c.b, duration: 0.6, delay: 0.9 });
    };
    if (!onView) { go(); return undefined; }
    const st = ScrollTrigger.create({ trigger: root.current, start: "top 82%", once: true, onEnter: go });
    return () => st.kill();
  }, [cur, done, onView]);

  const accent = tone === "green" ? "#16a34a" : "#1478f0";
  const idle = "#c3d7e3";

  return (
    <div ref={root} className="overflow-hidden rounded-[14px] bg-[#f5f6f7] p-5 shadow-[0_10px_30px_-12px_rgba(20,60,100,0.35)] sm:p-8">
      <div className="flex flex-wrap items-start justify-between gap-x-8 gap-y-2">
        <p data-head className="text-[20px] font-medium uppercase tracking-tight text-night sm:text-[22px]">{title} <span className="font-extrabold" style={{ color: accent }}>{titleAccent}</span></p>
        <dl data-head className="text-right text-[15px] text-night sm:text-[17px]">
          {meta.map(([k, v]) => <div key={k}><dt className="inline font-medium">{k} </dt><dd className="inline font-extrabold">{v}</dd></div>)}
        </dl>
      </div>

      {/* Colis 3D */}
      <div className="relative mt-4">
        <canvas ref={canvas} className="pointer-events-none block h-[84px] w-full" aria-hidden="true" />

        {/* Barre + pastilles */}
        <div className="relative flex items-center" role="list" aria-label={title}>
          <div className="absolute left-[18px] top-1/2 h-[18px] -translate-y-1/2 rounded-full" style={{ background: idle, right: described ? `calc(${100 / steps.length}% - 18px + 0px)` : 18 }} aria-hidden="true" />
          <div className="absolute left-[18px] top-1/2 h-[18px] -translate-y-1/2" style={{ right: described ? `calc(${100 / steps.length}% - 18px + 0px)` : 18 }} aria-hidden="true">
            <div ref={fill} className="h-full rounded-full" style={{ background: accent, width: 0 }} />
          </div>
          <div className={described ? "relative grid w-full items-center" : "relative flex w-full items-center justify-between"} style={described ? { gridTemplateColumns: `repeat(${steps.length}, minmax(0, 1fr))` } : undefined}>
            {steps.map((s, i) => {
              const reached = i <= cur;
              return (
                <div key={s.key} role="listitem" aria-current={i === cur ? "step" : undefined} ref={(n) => { nodes.current[i] = n; }} data-node {...(reached ? { "data-done": "" } : {})} className="relative flex h-9 w-9 items-center justify-center">
                  {i === cur && !done && <span data-pulse className="absolute inset-0 rounded-full" style={{ background: accent }} aria-hidden="true" />}
                  <span className="relative flex h-9 w-9 items-center justify-center rounded-full ring-[5px] ring-[#f5f6f7]" style={reached ? { background: accent, color: "#fff" } : { background: "#e7eef3", boxShadow: `inset 0 0 0 6px ${idle}` }}>
                    {reached && (
                      <svg data-check viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                        <path d="M5 12.5l4.5 4.5L19 7.5" strokeDasharray="24" strokeDashoffset="0" />
                      </svg>
                    )}
                  </span>
                </div>
              );
            })}
          </div>
        </div>

        {/* Étapes : icône + libellé */}
        <ul className="mt-7 grid" style={{ gridTemplateColumns: `repeat(${steps.length}, minmax(0, 1fr))` }}>
          {steps.map((s, i) => (
            <li key={s.key} data-label className={`flex gap-3 ${described ? "items-start justify-start pr-4" : `items-center ${i === 0 ? "justify-start" : i === last ? "justify-end" : "justify-center"}`} ${i <= cur ? "" : "opacity-60"}`}>
              <span data-icon {...(i === cur ? { "data-active-icon": "" } : {})} className="flex h-14 w-14 shrink-0 items-center justify-center"><s.Icon size={52} /></span>
              <span className={described ? "min-w-0" : `max-w-[6.5rem] text-[13px] leading-tight ${i === cur ? "font-bold text-night" : "text-gray-600"}`}>
                {described ? (<><strong className="block text-[16px] font-bold leading-tight text-night">{s.label}</strong><span className="mt-1 block text-[13.5px] leading-snug text-gray-500">{s.desc}</span></>) : s.label}
              </span>
              
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
