// DONNÉES DE DÉMONSTRATION — aucune référence, aucun prix ni aucun stock réel.
// Les noms et libellés sont traduits via i18n (clés prod.<id>, cat.<slug>, stock.*).
export const CATEGORIES = [
  { slug: "hydraulic" },
  { slug: "oil" },
  { slug: "air" },
  { slug: "compressed-air" },
  { slug: "process" },
];

export const DEMO_PRODUCTS = [
  { id: 1, brand: "Brand A", reference: "DEMO-HYD-001", category: "hydraulic", price: 128.5, inStock: true, crossRefs: ["DEMO-X-1001", "DEMO-X-1002"] },
  { id: 2, brand: "Brand B", reference: "DEMO-HYD-002", category: "hydraulic", price: 84.0, inStock: true, crossRefs: ["DEMO-X-2001"] },
  { id: 3, brand: "Brand A", reference: "DEMO-OIL-003", category: "oil", price: 24.9, inStock: true, crossRefs: ["DEMO-X-3001", "DEMO-X-3002"] },
  { id: 4, brand: "Brand C", reference: "DEMO-OIL-004", category: "oil", price: 31.75, inStock: false, crossRefs: [] },
  { id: 5, brand: "Brand B", reference: "DEMO-AIR-005", category: "air", price: 42.2, inStock: true, crossRefs: ["DEMO-X-5001"] },
  { id: 6, brand: "Brand D", reference: "DEMO-CAF-006", category: "compressed-air", price: 215.0, inStock: true, crossRefs: ["DEMO-X-6001"] },
  { id: 7, brand: "Brand C", reference: "DEMO-PRO-007", category: "process", price: 18.4, inStock: true, crossRefs: [] },
  { id: 8, brand: "Brand D", reference: "DEMO-PRO-008", category: "process", price: 56.0, inStock: true, crossRefs: ["DEMO-X-8001"] },
];
