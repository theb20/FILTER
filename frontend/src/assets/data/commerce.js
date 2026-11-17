// @ts-check
// Types des règles commerciales. Les VALEURS (TVA, zones et frais de livraison, seuils, codes promotionnels) viennent de l'API :
// elles se saisissent au backoffice (Boutique) et sont lues par store/siteStore.js et lib/promo.js. Aucune valeur n'est codée ici.

/**
 * @typedef {{ code: string, type: "percent" | "fixed", value: number, minSubtotal?: number }} PromoCode
 * @typedef {{ id: string, name: string, fee: number, days: [number, number], freeEligible: boolean }} ShippingZone
 */

export {};
