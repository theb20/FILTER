import type { GuideTour } from "../types";
import { dashboardTour } from "./dashboard";
import { companiesTour, companyDetailTour, customerDetailTour, customersTour } from "./sales";
import {
  attributeTour, brandFormTour, brandsTour, categoriesTour, categoryDetailTour, categoryNewTour, productDetailTour, productNewTour, productsTour, sectorFormTour, sectorsTour,
} from "./catalog";
import { shopTours } from "./shop";
import { accountTour, auditTour, roleFormTour, rolesTour, settingsTour, staffDetailTour, staffNewTour, staffTour } from "./admin";

/** Registre des visites : une par page ou famille de pages. Ajouter une page = un fichier de contenu + une ligne ici. */
export const TOURS: GuideTour[] = [
  dashboardTour,
  customersTour, customerDetailTour, companiesTour, companyDetailTour,
  productsTour, productNewTour, productDetailTour, categoriesTour, categoryNewTour, categoryDetailTour, attributeTour, brandsTour, brandFormTour, sectorsTour, sectorFormTour,
  ...shopTours,
  staffTour, staffNewTour, staffDetailTour, rolesTour, roleFormTour, auditTour, settingsTour, accountTour,
];
