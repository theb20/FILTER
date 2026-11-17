import type { PerspectiveCamera, Scene } from "three";

export type Pointer = { x: number; y: number };
export type SceneCtx = { scene: Scene; camera: PerspectiveCamera; /** Animations réduites : la scène se pose dans un état parlant, sans mouvement. */ reduced: boolean };
export type SceneRuntime = { update: (t: number, dt: number, pointer: Pointer) => void; stop?: () => void };
export type SceneBuild = (ctx: SceneCtx) => SceneRuntime;

export const COLORS = { action: 0xf7be14, night: 0x131921, navy: 0x232f3e, slate: 0x3e4861, ok: 0x00b67a, link: 0x0275c8, grey: 0xc3c9d3, soft: 0xe9ecf1 };
