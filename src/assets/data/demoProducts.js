// DONNÉES DE DÉMONSTRATION — aucune référence, aucun prix ni aucun stock réel.
export const CATEGORIES = [
  { slug: "hydraulic", label: "Hydraulic Filters" },
  { slug: "oil", label: "Oil Filters" },
  { slug: "air", label: "Air Filters" },
  { slug: "compressed-air", label: "Compressed Air Filters" },
  { slug: "process", label: "Process Filters" },
];

export const DEMO_PRODUCTS = [
  { id: 1, name: "Hydraulic Pressure Filter Element", brand: "Brand A", reference: "DEMO-HYD-001", category: "hydraulic", price: 128.5, stock: "In stock", crossRefs: ["DEMO-X-1001", "DEMO-X-1002"] },
  { id: 2, name: "Hydraulic Return Line Filter", brand: "Brand B", reference: "DEMO-HYD-002", category: "hydraulic", price: 84.0, stock: "In stock", crossRefs: ["DEMO-X-2001"] },
  { id: 3, name: "Spin-On Oil Filter", brand: "Brand A", reference: "DEMO-OIL-003", category: "oil", price: 24.9, stock: "In stock", crossRefs: ["DEMO-X-3001", "DEMO-X-3002"] },
  { id: 4, name: "Oil Filter Cartridge", brand: "Brand C", reference: "DEMO-OIL-004", category: "oil", price: 31.75, stock: "Backorder", crossRefs: [] },
  { id: 5, name: "Panel Air Filter", brand: "Brand B", reference: "DEMO-AIR-005", category: "air", price: 42.2, stock: "In stock", crossRefs: ["DEMO-X-5001"] },
  { id: 6, name: "Compressed Air Coalescing Filter", brand: "Brand D", reference: "DEMO-CAF-006", category: "compressed-air", price: 215.0, stock: "In stock", crossRefs: ["DEMO-X-6001"] },
  { id: 7, name: "Process Filter Bag", brand: "Brand C", reference: "DEMO-PRO-007", category: "process", price: 18.4, stock: "In stock", crossRefs: [] },
  { id: 8, name: "Process Filter Cartridge", brand: "Brand D", reference: "DEMO-PRO-008", category: "process", price: 56.0, stock: "In stock", crossRefs: ["DEMO-X-8001"] },
];

export const formatUsd = (n) =>
  new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" }).format(n);
