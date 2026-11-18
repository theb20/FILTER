import type { GuideTour } from "../types";
import { dashboardTour } from "./dashboard";
import { companiesTour, companyDetailTour, customerDetailTour, customersTour } from "./sales";
import {
  attributeTour, brandFormTour, brandsTour, categoriesBulkTour, categoriesTour, categoryDetailTour, productsBulkTour, categoryNewTour, productDetailTour, productNewTour, productsTour, sectorFormTour, sectorsTour,
} from "./catalog";
import { quoteDetailTour, quotesTour, shopTours } from "./shop";
import { costsTour, sourceDetailTour, sourceNewTour, sourcesTour } from "./engine";
import { accountTour, auditTour, roleFormTour, rolesTour, settingsTour, staffDetailTour, staffNewTour, staffTour } from "./admin";

/** Registre des visites : une par page ou famille de pages. Ajouter une page = un fichier de contenu + une ligne ici. */
export const TOURS: GuideTour[] = [
  dashboardTour,
  customersTour, customerDetailTour, companiesTour, companyDetailTour,
  productsTour, productsBulkTour, categoriesBulkTour, productNewTour, productDetailTour, categoriesTour, categoryNewTour, categoryDetailTour, attributeTour, brandsTour, brandFormTour, sectorsTour, sectorFormTour,
  ...shopTours, quotesTour, quoteDetailTour,
  sourcesTour, sourceNewTour, sourceDetailTour, costsTour,
  staffTour, staffNewTour, staffDetailTour, rolesTour, roleFormTour, auditTour, settingsTour, accountTour,
];
